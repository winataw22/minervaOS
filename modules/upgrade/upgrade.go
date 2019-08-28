package upgrade

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules/zinit"

	"github.com/blang/semver"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules"
)

type hookType string

const (
	hookPreCopy   hookType = "pre-copy"
	hookPostCopy  hookType = "post-copy"
	hookMigrate   hookType = "migrate"
	hookPostStart hookType = "post-start"
)

const (
	provisionModuleName = "provisiond"
)

// Upgrader is the component that is responsible
// to keep 0-OS up to date
type Upgrader struct {
	root    string
	version semver.Version
	flister modules.Flister
	zinit   *zinit.Client
}

// New creates a new UpgradeModule object
func New(root string, flister modules.Flister, zinit *zinit.Client) *Upgrader {
	if err := os.MkdirAll(root, 0770); err != nil {
		return nil
	}

	version, err := ensureVersionFile(root)
	if err != nil {
		return nil
	}

	log.Info().Msgf("current version %s", version.String())
	return &Upgrader{
		version: version,
		root:    root,
		flister: flister,
		zinit:   zinit,
	}
}

func ensureVersionFile(root string) (version semver.Version, err error) {
	versionPath := filepath.Join(root, "version")
	version, err = readVersion(versionPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error().Err(err).Msg("read version")
			return version, err
		}
		log.Info().Msg("no version found, assuming fresh install")
		// the file doesn't exist yet. So we are on a fresh system
		version = semver.MustParse("0.0.1")
		if err := writeVersion(versionPath, version); err != nil {
			log.Error().Err(err).Msg("fail to write version")
			return version, err
		}
	}
	return version, nil
}

func (u *Upgrader) enterSelfUpgrade(version semver.Version) error {
	path := filepath.Join(u.root, "selfupgraded")
	if err := ioutil.WriteFile(path, []byte(version.String()), 0400); err != nil {
		return errors.Wrap(err, "fail to write selfupgrade file")
	}
	return nil
}

func (u *Upgrader) isInSelfUpgrade(version semver.Version) (bool, error) {
	path := filepath.Join(u.root, "selfupgraded")

	defer os.Remove(path)

	v, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return true, err
	}

	return string(v) == version.String(), nil
}

// Upgrade is the method that does a full upgrade flow
// first check if a new version is available
// if yes, applies the upgrade
func (u *Upgrader) Upgrade(p Publisher) error {

	ok, latest, err := isNewVersionAvailable(u.version, p)
	if err != nil {
		return err
	}
	if !ok {
		// no new version available
		return nil
	}

	toApply, err := versionsToApply(u.version, latest, p)
	if err != nil {
		return err
	}

	for _, version := range toApply {
		upgrade, err := p.Get(version)
		if err != nil {
			log.Error().
				Err(err).
				Str("version", version.String()).
				Msg("fail to retrieve upgrade from publisher")
			return errors.Wrap(err, "fail to retrieve upgrade from publisher")
		}

		log.Info().
			Str("curent version", u.version.String()).
			Str("new version", version.String()).
			Msg("start upgrade")

		// during an upgrade we always stop provisiond to avoid
		// modifying the system while we upgraded
		if err := u.zinit.Stop(provisionModuleName); err != nil {
			log.Error().Err(err).Msgf("failed to stop %s", provisionModuleName)
			return errors.Wrapf(err, "failed to stop %s", provisionModuleName)
		}
		defer func() {
			if err := u.zinit.Start(provisionModuleName); err != nil {
				log.Error().Err(err).Msgf("failed to start %s", provisionModuleName)
			}
		}()

		if err := u.applyUpgrade(version, upgrade); err != nil {
			log.Error().
				Err(err).
				Str("version", version.String()).
				Msg("fail to apply upgrade")
			break
		}

		u.version = version
		if err := writeVersion(filepath.Join(u.root, "version"), version); err != nil {
			log.Error().
				Err(err).
				Str("version", version.String()).
				Msg("fail to write version to disks")
		}
	}

	return nil
}

func isNewVersionAvailable(current semver.Version, p Publisher) (bool, semver.Version, error) {
	latest, err := p.Latest()
	if err != nil {
		log.Error().
			Err(err).
			Msg("fail to get latest version from publisher")
		return false, latest, err
	}

	if current.Equals(latest) {
		log.Info().
			Str("version", current.String()).
			Msg("current and latest version match, nothing to do")
		return false, latest, nil
	}
	if current.GT(latest) {
		log.Warn().
			Str("current version", current.String()).
			Str("latest version", latest.String()).
			Msg("current version is higher then latest reported by publisher")
		return false, latest, nil
	}

	log.Info().
		Str("current version", current.String()).
		Str("new version", latest.String()).
		Msg("new version available")
	return true, latest, nil
}

func versionsToApply(current, latest semver.Version, p Publisher) ([]semver.Version, error) {

	versions, err := p.List()
	if err != nil {
		log.Error().
			Err(err).
			Msg("fail to list available version from publisher")
		return nil, err
	}
	semver.Sort(semver.Versions(versions))

	latestFound := false
	toApply := []semver.Version{}
	for _, v := range versions {
		// if the v is a higher version as the current version
		if current.Compare(v) < 0 {
			toApply = append(toApply, v)
		}

		if v.Equals(latest) {
			latestFound = true
			break
		}
	}
	if !latestFound {
		return nil, fmt.Errorf("latest version has not been found in available versions of the publisher")
	}

	return toApply, nil
}

func (u *Upgrader) applyUpgrade(version semver.Version, upgrade Upgrade) error {
	log.Info().Str("flist", upgrade.Flist).Msg(("start applying upgrade"))

	flistRoot, err := u.flister.Mount(upgrade.Flist, upgrade.Storage)
	if err != nil {
		return err
	}
	defer func() {
		if err := u.flister.Umount(flistRoot); err != nil {
			log.Error().Err(err).Msgf("fail to umount flist at %s: %v", flistRoot, err)
		}
	}()

	if err := executeHook(filepath.Join(flistRoot, string(hookPreCopy))); err != nil {
		log.Error().Err(err).Msg("fail to execute pre-copy script")
		return err
	}

	files, err := listDir(flistRoot)
	if err != nil {
		return err
	}

	log.Debug().Strs("files", files).Msg("prepare to copy new files")
	if err := mergeFs(files, "/", flistRoot); err != nil {
		return err
	}

	if err := executeHook(filepath.Join(flistRoot, string(hookPostCopy))); err != nil {
		log.Error().Err(err).Msg("fail to execute post-copy script")
		return err
	}

	services := servicesToRestart(files, flistRoot)
	log.Info().Strs("services", services).Msg("services to upgrade")

	for _, service := range services {
		if service == "upgraded" {
			inUpgrade, err := u.isInSelfUpgrade(version)
			if err != nil {
				return err
			}

			if !inUpgrade {
				log.Info().Msg("start self upgrade")
				if err := u.enterSelfUpgrade(version); err != nil {
					return err
				}
				log.Info().Msg("upgraded will now exit")
				log.Info().Msg("it will be restarted by zinit and then continue the test of the upgrade")
				// exit and wait for zinit to restart us
				os.Exit(0)
			}
		}
	}

	for _, service := range services {
		if service == "upgraded" {
			// we already dealt with upgraded
			continue
		}
		log.Info().Str("service", service).Msg("stop service")
		if err := u.zinit.Stop(service); err != nil {
			return err
		}
	}
	for _, service := range services {
		if service == "upgraded" {
			// we already dealt with upgraded
			continue
		}
		if err := u.waitServiceStop(time.Millisecond*500, service); err != nil {
			return errors.Wrap(err, "failed to stop service")
		}
	}
	log.Info().Msg("all services stopped")

	if err := executeHook(filepath.Join(flistRoot, string(hookMigrate))); err != nil {
		log.Error().Err(err).Msg("fail to execute migrate script")
		return err
	}

	for _, service := range services {
		log.Info().Str("service", service).Msg("restart service")
		if err := u.zinit.Start(service); err != nil {
			return err
		}
	}

	if err := executeHook(filepath.Join(flistRoot, string(hookPostStart))); err != nil {
		log.Error().Err(err).Msg("fail to execute post-start script")
		return err
	}

	log.Info().Str("flist", upgrade.Flist).Msg(("upgrade applied"))

	return nil
}

func executeHook(path string) error {
	name := filepath.Base(path)
	log.Info().Msgf("prepare to execute %s hook", name)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("script", path).Msgf("%s upgrade hook script not found, skipping", name)
			return nil
		}
		return err
	}

	if !isExecutable(info.Mode().Perm()) {
		return fmt.Errorf("%s exists but is not executable", name)
	}

	cmd := exec.Command(path)
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Info().Str("script", path).Msgf("%s upgrade hook script executed", name)

	return nil
}

// servicesToRestart look into the files of an upgrade flist
// and check if the file located in the /bin directory have a matching init service
// it retruns the name of all the init services that matches
func servicesToRestart(files []string, mountpoint string) []string {
	services := []string{}
	for _, file := range files {

		if mountpoint != "" {
			file = strings.TrimPrefix(file, mountpoint)

		}
		if file[:4] != "/bin" {
			continue
		}
		name := filepath.Base(file)
		if exists(fmt.Sprintf("/etc/zinit/%s.yaml", name)) || exists(fmt.Sprintf("/etc/zinit/%sd.yaml", name)) {
			services = append(services, name)
		}
	}
	return services
}

func trimMounpoint(mountpoint, path string) string {
	if mountpoint[len(mountpoint)-1] == filepath.Separator {
		mountpoint = mountpoint[:len(mountpoint)-1]
	}
	return path[len(mountpoint):]
}

func mergeFs(files []string, destination string, flistRoot string) error {

	skippingFiles := []string{
		fmt.Sprintf("/%s", string(hookPreCopy)),
		fmt.Sprintf("/%s", string(hookPostCopy)),
		fmt.Sprintf("/%s", string(hookMigrate)),
		fmt.Sprintf("/%s", string(hookPostStart)),
		"/bin/upgraded",
	}

	for _, flistPath := range files {

		dest, err := changeRoot(destination, trimMounpoint(flistRoot, flistPath))
		if err != nil {
			return err
		}

		// don't copy hook scripts
		if isIn(dest, skippingFiles) {
			continue
		}

		// make sure the directory of the file exists
		if err := os.MkdirAll(filepath.Dir(dest), 0770); err != nil {
			return err
		}

		info, err := os.Stat(flistPath)
		if err != nil {
			return err
		}

		// upgrade flist should only container directory and regular files
		if !info.Mode().IsRegular() {
			log.Info().Msgf("skip %s: not a regular file", flistPath)
			continue
		}

		// copy the file to final destination
		if err := copyFile(dest, flistPath); err != nil {
			return err
		}
	}
	return nil
}

func isIn(target string, list []string) bool {
	for _, x := range list {
		if target == x {
			return true
		}
	}
	return false
}

func copyFile(dst, src string) error {
	log.Info().Str("source", src).Str("destination", dst).Msg("copy file")

	var (
		isNew  = false
		dstOld string
	)

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		// case where this is a new file
		// we just need to copy from flist to root
		isNew = true
	}

	if !isNew {
		dstOld = dst + ".old"
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}
	}

	fSrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fSrc.Close()

	stat, err := fSrc.Stat()
	if err != nil {
		return err
	}

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, stat.Mode().Perm())
	if err != nil {
		return err
	}
	defer fDst.Close()

	if _, err = io.Copy(fDst, fSrc); err != nil {
		return err
	}

	if !isNew {
		return os.Remove(dstOld)
	}
	return nil
}

var errNotAbsolute = errors.New("path is not absolute")

// changeRoot changes the root of path by base
// both base and path needs to be absolute
func changeRoot(base, path string) (string, error) {
	if !filepath.IsAbs(base) {
		return "", errNotAbsolute
	}
	if !filepath.IsAbs(path) {
		return "", errNotAbsolute
	}

	ss := strings.SplitN(path, string(filepath.Separator), 2)
	if len(ss) > 1 {
		return filepath.Join(base, ss[1]), nil
	}
	return base, nil
}

func (u *Upgrader) waitServiceStop(dureation time.Duration, service string) error {
	log.Info().Str("service", service).Msg("waiting for service to stop")
	timeout := time.After(time.Second * 10)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached while waiting for service %s to be stopped", service)
		default:
			status, err := u.zinit.Status(service)
			if err != nil {
				return err
			}
			if status.State == zinit.ServiceStatusRunning {
				time.Sleep(time.Millisecond * 500)
				continue
			}
			return nil
		}
	}
}
