package upgrade

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"

	"github.com/threefoldtech/0-fs/meta"
	"github.com/threefoldtech/0-fs/rofs"
	"github.com/threefoldtech/0-fs/storage"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/upgrade/hub"
	"github.com/threefoldtech/zos/pkg/zinit"

	"github.com/rs/zerolog/log"
)

var (
	// ErrRestartNeeded is returned if upgraded requires a restart
	ErrRestartNeeded = fmt.Errorf("restart needed")

	// services that can't be uninstalled with normal procedure
	protected = []string{"identityd", "redis"}
)

const (
	service = "upgrader"

	defaultHubStorage  = "zdb://hub.grid.tf:9900"
	defaultZinitSocket = "/var/run/zinit.sock"

	checkForUpdateEvery = 60 * time.Minute
	checkJitter         = 10 // minutes

	ZosRepo    = "tf-zos"
	ZosPackage = "zos.flist"
)

// Upgrader is the component that is responsible
// to keep 0-OS up to date
type Upgrader struct {
	boot         Boot
	zinit        *zinit.Client
	root         string
	noZosUpgrade bool
	hub          hub.HubClient
	storage      storage.Storage
}

// UpgraderOption interface
type UpgraderOption func(u *Upgrader) error

// NoZosUpgrade option, enable or disable
// the update of zos binaries.
// enabled by default
func NoZosUpgrade(o bool) UpgraderOption {
	return func(u *Upgrader) error {
		u.noZosUpgrade = o

		return nil
	}
}

// Storage option overrides the default hub storage url
// default value is hub.grid.tf
func Storage(url string) func(u *Upgrader) error {
	return func(u *Upgrader) error {
		storage, err := storage.NewSimpleStorage(url)
		if err != nil {
			return errors.Wrap(err, "failed to initialize hub storage")
		}
		u.storage = storage
		return nil
	}
}

// Zinit option overrides the default zinit socket
func Zinit(socket string) func(u *Upgrader) error {
	return func(u *Upgrader) error {
		zinit := zinit.New(socket)
		u.zinit = zinit
		return nil
	}
}

// NewUpgrader creates a new upgrader instance
func NewUpgrader(root string, opts ...UpgraderOption) (*Upgrader, error) {
	u := &Upgrader{
		root: root,
	}

	for _, dir := range []string{u.fileCache(), u.flistCache()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrap(err, "failed to prepare cache directories")
		}
	}

	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, err
		}
	}

	if u.storage == nil {
		// no storage option was set. use default
		if err := Storage(defaultHubStorage)(u); err != nil {
			return nil, err
		}
	}

	if u.zinit == nil {
		if err := Zinit(defaultZinitSocket)(u); err != nil {
			return nil, err
		}
	}

	return u, nil
}

func (u *Upgrader) Run(ctx context.Context) error {
	method := u.boot.DetectBootMethod()
	if method == BootMethodOther {
		// we need to do an update one time to fetch all
		// binaries required by the system except for the zos
		// binaries
		// then we should block forever
		log.Info().Msg("system is not booted from the hub")
		if app.IsFirstBoot(service) {
			remote, err := u.remote()
			if err != nil {
				return errors.Wrap(err, "failed to get remote tag")
			}

			if err := u.updateTo(remote); err != nil {
				return errors.Wrap(err, "failed to run update")
			}
		}
		// to avoid redoing the binary installation
		// when service is restarted
		if err := app.MarkBooted(service); err != nil {
			return errors.Wrap(err, "failed to mark system as booted")
		}

		log.Info().Msg("update is disabled")
		<-ctx.Done()
		return nil
	}

	for {
		err := u.update()
		if errors.Is(err, ErrRestartNeeded) {
			return err
		} else if err != nil {
			log.Error().Err(err).Msg("failed while checking for updates")
			<-time.After(10 * time.Second)
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(u.nextUpdate()):
		}

	}
}

func (u *Upgrader) Version() semver.Version {
	return u.boot.Version()
}

func (u *Upgrader) nextUpdate() time.Duration {
	jitter := rand.Intn(checkJitter)
	next := checkForUpdateEvery + (time.Duration(jitter) * time.Minute)
	log.Info().Str("after", next.String()).Msg("checking for update after")
	return next
}

func (u *Upgrader) remote() (remote hub.TagLink, err error) {
	mode := u.boot.RunMode()
	// find all taglinks that matches the same run mode (ex: development)
	matches, err := u.hub.Find(
		ZosRepo,
		hub.MatchName(mode.String()),
		hub.MatchType(hub.TypeTagLink),
	)

	if err != nil {
		return remote, err
	}

	if len(matches) != 1 {
		return remote, fmt.Errorf("can't find taglink that matches '%s'", mode.String())
	}

	return hub.NewTagLink(matches[0]), nil
}

func (u *Upgrader) update() error {
	// here we need to do a normal full update cycle
	current, err := u.boot.Current()
	if err != nil {
		log.Error().Err(err).Msg("failed to get info about current version, update anyway")
	}

	remote, err := u.remote()
	if err != nil {
		return errors.Wrap(err, "failed to get remote tag")
	}

	// obviously a remote tag need to match the current tag.
	// if the remote is different, we actually run the update and exit.
	if remote.Target == current.Target {
		// nothing to do!
		return nil
	}

	log.Info().Str("version", filepath.Base(remote.Target)).Msg("updating system...")
	if err := u.updateTo(remote); err != nil {
		return errors.Wrapf(err, "failed to update to new tag '%s'", remote.Target)
	}

	if err := u.boot.Set(remote); err != nil {
		return err
	}

	return ErrRestartNeeded
}

func (u *Upgrader) updateTo(link hub.TagLink) error {
	repo, tag, err := link.Destination()
	if err != nil {
		return errors.Wrap(err, "failed to get destination tag")
	}

	packages, err := u.hub.ListTag(repo, tag)
	if err != nil {
		return errors.Wrapf(err, "failed to list tag '%s' packages", tag)
	}

	var later [][]string
	for _, pkg := range packages {
		pkgRepo, name, err := pkg.Destination(repo)
		if pkg.Name == ZosPackage {
			// this is the last to do
			log.Debug().Str("repo", pkgRepo).Str("name", name).Msg("schedule package for later")
			later = append(later, []string{pkgRepo, name})
			continue
		}

		if err != nil {
			return errors.Wrapf(err, "failed to find target for package '%s'", pkg.Target)
		}

		// install package
		if err := u.install(pkgRepo, name); err != nil {
			return errors.Wrapf(err, "failed to install package %s/%s", pkgRepo, name)
		}
	}

	if u.noZosUpgrade {
		return nil
	}

	// probably check flag for zos installation
	for _, pkg := range later {
		repo, name := pkg[0], pkg[1]
		if err := u.install(repo, name); err != nil {
			return errors.Wrapf(err, "failed to install package %s/%s", repo, name)
		}
	}

	return nil
}

func (u *Upgrader) flistCache() string {
	return filepath.Join(u.root, "cache", "flist")
}

func (u *Upgrader) fileCache() string {
	return filepath.Join(u.root, "cache", "files")
}

// getFlist accepts fqdn of flist as `<repo>/<name>.flist`
func (u *Upgrader) getFlist(repo, name string) (meta.Walker, error) {
	db, err := u.hub.Download(u.flistCache(), repo, name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download flist")
	}

	store, err := meta.NewStore(db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load flist db")
	}

	walker, ok := store.(meta.Walker)
	if !ok {
		store.Close()
		return nil, errors.Wrap(err, "flist database of unsupported type")
	}

	return walker, nil
}

// install from a single flist.
func (u *Upgrader) install(repo, name string) error {
	log.Info().Str("repo", repo).Str("name", name).Msg("start installing package")

	store, err := u.getFlist(repo, name)
	if err != nil {
		return errors.Wrapf(err, "failed to process flist: %s/%s", repo, name)
	}
	defer store.Close()

	if err := safe(func() error {
		// copy is done in a safe closer to avoid interrupting
		// the installation
		return u.copyRecursive(store, "/")
	}); err != nil {
		return errors.Wrapf(err, "failed to install flist: %s/%s", repo, name)
	}

	services, err := u.servicesFromStore(store)
	if err != nil {
		return errors.Wrap(err, "failed to list services from flist")
	}

	return u.ensureRestarted(services...)
}

func (u *Upgrader) servicesFromStore(store meta.Walker) ([]string, error) {
	const zinitPath = "/etc/zinit"

	var services []string
	err := store.Walk(zinitPath, func(path string, info meta.Meta) error {
		if info.IsDir() {
			return nil
		}
		dir := filepath.Dir(path)
		if dir != zinitPath {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}

		services = append(services,
			strings.TrimSuffix(info.Name(), ".yaml"))
		return nil
	})

	if err == meta.ErrNotFound {
		return nil, nil
	}

	return services, err
}

func (u *Upgrader) ensureRestarted(service ...string) error {
	// remove protected function from list, these never restarted
	service = slices.DeleteFunc(service, func(e string) bool {
		return slices.Contains(protected, e)
	})

	log.Debug().Strs("services", service).Msg("ensure services")
	if len(service) == 0 {
		return nil
	}

	log.Debug().Strs("services", service).Msg("restarting services")
	if err := u.zinit.StopMultiple(20*time.Second, service...); err != nil {
		// we log here so we don't leave the node in a bad state!
		// by just trying to start as much services as we can
		log.Error().Err(err).Msg("failed to stop all services")
	}

	for _, name := range service {
		log.Info().Str("service", name).Msg("starting service")
		if err := u.zinit.Forget(name); err != nil {
			log.Warn().Err(err).Str("service", name).Msg("could not forget service")
		}

		if err := u.zinit.Monitor(name); err != nil && err != zinit.ErrAlreadyMonitored {
			log.Error().Err(err).Str("service", name).Msg("could not monitor service")
		}

		// this has no effect if Monitor already worked with no issue
		// but we do it anyway for services that could not be forgotten (did not stop)
		// so we start them again
		if err := u.zinit.Start(name); err != nil {
			log.Error().Err(err).Str("service", name).Msg("could not start service")
		}
	}

	return nil
}

func (u *Upgrader) copyRecursive(store meta.Walker, destination string, skip ...string) error {
	return store.Walk("", func(path string, info meta.Meta) error {

		dest := filepath.Join(destination, path)
		if isIn(dest, skip) {
			if info.IsDir() {
				return meta.ErrSkipDir
			}
			log.Debug().Str("file", dest).Msg("skipping file")
			return nil
		}

		if info.IsDir() {
			if err := os.MkdirAll(dest, os.FileMode(info.Info().Access.Mode)); err != nil {
				return err
			}
			return nil
		}

		stat := info.Info()

		switch stat.Type {
		case meta.RegularType:
			// regular file (or other types that we don't handle)
			return u.copyFile(dest, info)
		case meta.LinkType:
			//fmt.Println("link target", stat.LinkTarget)
			target := stat.LinkTarget
			if filepath.IsAbs(target) {
				// if target is absolute, we make sure it's under destination
				// other wise use relative path
				target = filepath.Join(destination, stat.LinkTarget)
			}

			if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
				return err
			}

			return os.Symlink(target, dest)
		default:
			log.Debug().Str("type", info.Info().Type.String()).Msg("ignoring not suppored file type")
		}

		return nil
	})

}

func isIn(target string, list []string) bool {
	for _, x := range list {
		if target == x {
			return true
		}
	}
	return false
}

func (u *Upgrader) copyFile(dst string, src meta.Meta) error {
	log.Info().Str("source", src.Name()).Str("destination", dst).Msg("copy file")

	var (
		isNew  = false
		dstOld string
	)

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		// case where this is a new file
		// we just need to copy from flist to root
		isNew = true
	}

	var err error
	if !isNew {
		dstOld = dst + ".old"
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}

		defer func() {
			if err == nil {
				if err := os.Remove(dstOld); err != nil {
					log.Error().Err(err).Str("file", dstOld).Msg("failed to clean up backup file")
				}
				return
			}

			if err := os.Rename(dstOld, dst); err != nil {
				log.Error().Err(err).Str("file", dst).Msg("failed to restore file after a failed download")
			}
		}()
	}

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, os.FileMode(src.Info().Access.Mode))
	if err != nil {
		return err
	}
	defer fDst.Close()

	cache := rofs.NewCache(u.fileCache(), u.storage)
	fSrc, err := cache.CheckAndGet(src)
	if err != nil {
		return err
	}

	if _, err := io.Copy(fDst, fSrc); err != nil {
		return err
	}

	return nil
}

// safe makes sure function call not interrupted
// with a signal while execution
func safe(fn func() error) error {
	ch := make(chan os.Signal, 4)
	defer close(ch)
	defer signal.Stop(ch)

	// try to upgraded to latest
	// but mean while also make sure the daemon can not be killed by a signal
	signal.Notify(ch)
	return fn()
}
