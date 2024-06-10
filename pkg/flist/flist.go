package flist

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	defaultRoot  = "/var/cache/modules/flist"
	mib          = 1024 * 1024
	md5HexLength = 32

	defaultNamespace = "ndmz"
	publicNamespace  = "public"

	defaultHubCallTimeout = 20 * time.Second
)

var (
	// ErrAlreadyMounted is returned when checking if a path has already
	// something mounted on it
	ErrAlreadyMounted                  = errors.New("path is already mounted")
	ErrNotMountPoint                   = errors.New("path is not a mountpoint")
	ErrTransportEndpointIsNotConencted = errors.New("transport endpoint is not connected")
	ErrZFSProcessNotFound              = errors.New("0-fs process not found")
	ErrHashNotSupported                = errors.New("hash not supported by flist host")
	ErrHashInvalidLen                  = errors.New("invalid hash length")
)

// Hash type
type Hash string

// Path type
type Path string

type commander interface {
	Command(name string, arg ...string) *exec.Cmd
	GetNamespace(name string) (ns.NetNS, error)
}

type cmd func(name string, arg ...string) *exec.Cmd

func (c cmd) Command(name string, args ...string) *exec.Cmd {
	return c(name, args...)
}

func (c cmd) GetNamespace(name string) (ns.NetNS, error) {
	return namespace.GetByName(name)
}

type system interface {
	Mount(source string, target string, fstype string, flags uintptr, data string) (err error)
	Unmount(target string, flags int) error
}

type defaultSystem struct{}

func (d *defaultSystem) Mount(source string, target string, fstype string, flags uintptr, data string) (err error) {
	return syscall.Mount(source, target, fstype, flags, data)
}

func (d *defaultSystem) Unmount(target string, flags int) error {
	return syscall.Unmount(target, flags)
}

type volumeAllocator interface {
	// CreateFilesystem creates a filesystem with a given size. The filesystem
	// is mounted, and the path to the mountpoint is returned. The filesystem
	// is only attempted to be created in a pool of the given type. If no
	// more space is available in such a pool, `ErrNotEnoughSpace` is returned.
	// It is up to the caller to handle such a situation and decide if he wants
	// to try again on a different devicetype
	VolumeCreate(ctx context.Context, name string, size gridtypes.Unit) (pkg.Volume, error)

	// VolumeUpdate changes the size of an already existing volume
	VolumeUpdate(ctx context.Context, name string, size gridtypes.Unit) error

	// ReleaseFilesystem signals that the named filesystem is no longer needed.
	// The filesystem will be unmounted and subsequently removed.
	// All data contained in the filesystem will be lost, and the
	// space which has been reserved for this filesystem will be reclaimed.
	VolumeDelete(ctx context.Context, name string) error

	// Path return the filesystem named name
	// if no filesystem with this name exists, an error is returned
	VolumeLookup(ctx context.Context, name string) (pkg.Volume, error)
}

type flistModule struct {
	// root directory where all
	// the working file of the module will be located
	root string

	// underneath are the path for each
	// sub folder used by the flist module
	flist      string
	cache      string
	mountpoint string
	ro         string
	pid        string
	log        string

	storage   volumeAllocator
	commander commander
	system    system

	httpClient *http.Client
}

func newFlister(root string, storage volumeAllocator, commander commander, system system) *flistModule {
	if root == "" {
		root = defaultRoot
	}

	if err := os.MkdirAll(root, 0755); err != nil {
		panic(err)
	}

	// ensure we have proper permission for existing directory
	if err := os.Chmod(root, 0755); err != nil {
		panic(err)
	}

	// prepare directory layout for the module
	for _, path := range []string{"flist", "cache", "mountpoint", "ro", "pid", "log"} {
		p := filepath.Join(root, path)
		if err := os.MkdirAll(p, 0755); err != nil {
			panic(err)
		}
		// ensure we have proper permission for existing directory
		if err := os.Chmod(p, 0755); err != nil {
			panic(err)
		}
	}

	return &flistModule{
		root:       root,
		flist:      filepath.Join(root, "flist"),
		cache:      filepath.Join(root, "cache"),
		mountpoint: filepath.Join(root, "mountpoint"),
		ro:         filepath.Join(root, "ro"),
		pid:        filepath.Join(root, "pid"),
		log:        filepath.Join(root, "log"),

		storage:   storage,
		commander: commander,
		system:    system,

		httpClient: &http.Client{
			Timeout: defaultHubCallTimeout,
		},
	}
}

type options []string

func (o options) Find(k string) int {
	for i, x := range o {
		if strings.EqualFold(x, k) {
			return i
		}
	}

	return -1
}

// New creates a new flistModule
func New(root string, storage *stubs.StorageModuleStub) pkg.Flister {
	return newFlister(root, storage, cmd(exec.Command), &defaultSystem{})
}

// MountRO mounts an flist in read-only mode. This mount then can be shared between multiple rw mounts
// TODO: how to know that this ro mount is no longer used, hence can be unmounted and cleaned up?
func (f *flistModule) mountRO(url, storage, nsName string) (string, error) {
	// this should return always the flist mountpoint. which is used
	// as a base for all RW mounts.
	sublog := log.With().Str("url", url).Str("storage", storage).Logger()
	sublog.Info().Msg("request to mount flist")

	hash, flistPath, err := f.downloadFlist(url, nsName)
	if err != nil {
		sublog.Err(err).Msg("fail to download flist")
		return "", err
	}

	mountpoint, err := f.flistMountpath(hash)
	if err != nil {
		return "", err
	}

	err = f.valid(mountpoint)
	if err == ErrAlreadyMounted {
		return mountpoint, nil
	} else if err != nil {
		return "", errors.Wrap(err, "failed to validate mountpoint")
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create flist mountpoint")
	}
	// otherwise, we need to mount this flist in ro mode

	env, err := environment.Get()
	if err != nil {
		return "", errors.Wrap(err, "failed to parse node environment")
	}

	if storage == "" {
		storage = env.FlistURL
	}

	logPath := filepath.Join(f.log, string(hash)) + ".log"
	flistExt := filepath.Ext(url)
	args := []string{
		"--cache", f.cache,
		"--meta", string(flistPath),
		"--daemon",
		"--log", logPath,
	}

	var command string
	if flistExt == ".flist" {
		args = append([]string{
			"--storage-url", storage,
			// this is always read-only
			"--ro",
		}, args...)
		command = "g8ufs"
	} else if flistExt == ".fl" {
		args = append([]string{
			"mount",
		}, args...)
		command = "rfs"
	} else {
		return "", errors.Errorf("unknown extension: '%s'", flistExt)
	}

	args = append(args, mountpoint)

	// we do get the namespace via the commander
	// only to be able to mock it via tests.
	// by default this will be an actual namespace
	// returned from the system.
	// tests can return a mock namespace, or nil
	// if nil the code will execute on host namespace
	netNs, err := f.commander.GetNamespace(nsName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get network namespace to run mount")
	}

	run := func(_ ns.NetNS) error {
		// this command then will look something like
		// ip netns exec <ns> (rfs|g8ufs) [mount] --cache C --meta M --daemon --log L [g8ufs specific flags] mountpoint
		cmd := f.commander.Command(command, args...)
		log.Debug().Stringer("command", cmd).Msg("starting mount")

		var out []byte
		if out, err = cmd.CombinedOutput(); err != nil {
			sublog.Err(err).Str("out", string(out)).Msg("failed to start 0-fs daemon")
			return err
		}

		return nil
	}

	if netNs != nil {
		err = netNs.Do(run)
	} else {
		err = run(nil)
	}

	return mountpoint, err
}

func (f *flistModule) mountBind(ctx context.Context, name, ro string) error {
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return errors.Wrap(err, "failed to mount bind the flist")
	}

	// mount overlay as
	err = f.system.Mount(ro,
		mountpoint,
		"bind",
		syscall.MS_BIND,
		"",
	)
	if err != nil {
		return errors.Wrap(err, "failed to create mount bind")
	}
	defer func() {
		if err != nil {
			_ = f.system.Unmount(mountpoint, 0)
		}
	}()

	err = f.waitMountpoint(mountpoint, 3)
	return err
}

func (f *flistModule) mountOverlay(ctx context.Context, name, ro string, opt *pkg.MountOptions) error {
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return errors.Wrap(err, "failed to create overlay mountpoint")
	}

	persisted := opt.PersistedVolume
	if len(persisted) == 0 {
		// no persisted volume provided, hence
		// we need to create one, or find one that is already
		// there
		newAllocation := false
		var volume pkg.Volume
		defer func() {
			// in case of an error (mount is never fully completed)
			// we need to deallocate the filesystem

			if newAllocation && err != nil {
				_ = f.storage.VolumeDelete(ctx, name)
			}
		}()

		// check if the filesystem doesn't already exists
		volume, err = f.storage.VolumeLookup(ctx, name)
		if err != nil {
			log.Info().Msgf("create new subvolume %s", name)
			// and only create a new one if it doesn't exist
			if opt.Limit == 0 {
				// sanity check in case type is not set always use hdd
				return fmt.Errorf("invalid mount option, missing disk type")
			}
			newAllocation = true
			volume, err = f.storage.VolumeCreate(ctx, name, opt.Limit)
			if err != nil {
				return errors.Wrap(err, "failed to create read-write subvolume for 0-fs")
			}
		}

		persisted = volume.Path
	}

	log.Debug().Str("persisted-path", persisted).Str("name", name).Msg("using persisted path for mount")
	rw := filepath.Join(persisted, "rw")
	wd := filepath.Join(persisted, "wd")
	for _, d := range []string{rw, wd} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return errors.Wrapf(err, "failed to create overlay directory: %s", d)
		}
	}

	log.Debug().Str("ro", ro).Str("rw", rw).Str("wd", wd).Msg("mounting overlay")
	// mount overlay as
	err = f.system.Mount("overlay",
		mountpoint,
		"overlay",
		syscall.MS_NOATIME,
		fmt.Sprintf(
			"lowerdir=%s,upperdir=%s,workdir=%s",
			ro, rw, wd,
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to mount overlay")
	}

	return nil
}

func (f *flistModule) Exists(name string) (bool, error) {
	// mount overlay
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return false, errors.Wrap(err, "invalid mountpoint")
	}

	if err := f.valid(mountpoint); err == ErrAlreadyMounted {
		return true, nil
	} else if err != nil {
		return false, errors.Wrap(err, "validating of mount point failed")
	}

	return false, nil
}

func (f *flistModule) Mount(name, url string, opt pkg.MountOptions) (string, error) {
	return f.mountInNamespace(name, url, opt, defaultNamespace)
}

func (f *flistModule) mountInNamespace(name, url string, opt pkg.MountOptions, namespace string) (string, error) {
	sublog := log.With().Str("name", name).Str("url", url).Str("storage", opt.Storage).Logger()
	sublog.Info().Msgf("request to mount flist: %+v", opt)

	defer func() {
		if err := f.cleanUnusedMounts(); err != nil {
			log.Error().Err(err).Msg("failed to run clean up")
		}
	}()
	// mount overlay
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return "", errors.Wrap(err, "invalid mountpoint")
	}

	if err := f.valid(mountpoint); err == ErrAlreadyMounted {
		return mountpoint, nil
	} else if err != nil {
		return "", errors.Wrap(err, "validating of mount point failed")
	}

	ro, err := f.mountRO(url, opt.Storage, namespace)
	if err != nil {
		return "", errors.Wrap(err, "ro mount of flist failed")
	}

	if err := f.waitMountpoint(ro, 3); err != nil {
		return "", errors.Wrap(err, "failed to wait for flist mount")
	}
	ctx := context.Background()

	if opt.ReadOnly {
		sublog.Debug().Msg("mount bind")
		return mountpoint, f.mountBind(ctx, name, ro)
	}

	// otherwise
	sublog.Debug().Msg("mount overlay")
	return mountpoint, f.mountOverlay(ctx, name, ro, &opt)
}

func (f *flistModule) UpdateMountSize(name string, limit gridtypes.Unit) (string, error) {
	// mount overlay
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return "", errors.Wrap(err, "invalid mountpoint")
	}

	if err := f.isMountpoint(mountpoint); err != nil {
		return "", errors.Wrap(err, "flist not mounted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = f.storage.VolumeUpdate(ctx, name, limit)
	return mountpoint, err
}

func (f *flistModule) mountpath(name string) (string, error) {
	mountpath := filepath.Join(f.mountpoint, name)
	if filepath.Dir(mountpath) != f.mountpoint {
		return "", errors.New("invalid mount name")
	}

	return mountpath, nil
}

func (f *flistModule) flistMountpath(hash Hash) (string, error) {
	mountpath := filepath.Join(f.ro, string(hash))
	if filepath.Dir(mountpath) != f.ro {
		return "", errors.New("invalid mount name")
	}

	return mountpath, nil
}

// valid checks that this mount path is free, and can be used
func (f *flistModule) valid(path string) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil && strings.Contains(err.Error(), ErrTransportEndpointIsNotConencted.Error()) {
		return f.system.Unmount(path, 0)
	} else if err != nil {
		return errors.Wrapf(err, "failed to check mountpoint: %s", path)
	}

	if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	if err := f.isMountpoint(path); err == nil {
		return ErrAlreadyMounted
	}

	return nil
}

func (f *flistModule) waitMountpoint(path string, seconds int) error {
	for ; seconds >= 0; seconds-- {
		<-time.After(1 * time.Second)
		if err := f.isMountpoint(path); err == nil {
			return nil
		}
	}

	return fmt.Errorf("was not mounted in time")
}

func (f *flistModule) isMountpoint(path string) error {
	log.Debug().Str("mnt", path).Msg("testing mountpoint")
	return f.commander.Command("mountpoint", path).Run()
}

func (f *flistModule) getMountOptionsForPID(pid int64) (options, error) {
	cmdline, err := os.ReadFile(path.Join("/proc", fmt.Sprint(pid), "cmdline"))
	if os.IsNotExist(err) {
		return nil, ErrZFSProcessNotFound
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to read mount (%d) cmdline", pid)
	}

	parts := bytes.Split(cmdline, []byte{0})

	result := make(options, 0, len(parts))
	for _, part := range parts {
		result = append(result, string(part))
	}

	return result, nil
}

func (f *flistModule) HashFromRootPath(name string) (string, error) {
	path, err := f.mountpath(filepath.Base(name))
	if err != nil {
		return "", err
	}

	info, err := f.resolve(path)
	if err != nil {
		return "", err
	}

	// this can either be an overlay mount.
	opts, err := f.getMountOptionsForPID(info.Pid)
	if err != nil {
		return "", err
	}

	for _, opt := range opts {
		// if option start with the flist meta path
		if strings.HasPrefix(opt, f.flist) {
			// extracting hash (dirname) from argument
			return filepath.Base(opt), nil
		}
	}

	return "", fmt.Errorf("could not find rootfs hash name")
}

func (f *flistModule) Unmount(name string) error {
	defer func() {
		if err := f.cleanUnusedMounts(); err != nil {
			log.Error().Err(err).Msg("failed to run clean up")
		}
	}()
	// this will
	// - unmount the overlay mount
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return err
	}

	if f.valid(mountpoint) == ErrAlreadyMounted {
		if err := f.system.Unmount(mountpoint, 0); err != nil {
			log.Error().Err(err).Str("path", mountpoint).Msg("fail to umount flist")
		}
	}

	if err := os.RemoveAll(mountpoint); err != nil {
		log.Error().Err(err).Str("mnt", mountpoint).Msg("failed to remove mount point")
	}
	// - delete the volume, this should be done only for RW (TODO)
	// mounts, but for now it's still safe to try to remove the subvolume anyway
	// this will work only for rw mounts.
	if err := f.storage.VolumeDelete(context.Background(), name); err != nil {
		log.Error().Err(err).Msg("fail to clean up subvolume")
	}

	// TODO: is the ro flist still in use? if yes then we can safely clear it up
	// now.
	return nil
}

// FlistHash returns md5 of flist if available (requesting the hub)
func (f *flistModule) FlistHash(url string) (string, error) {
	// first check if the md5 of the flist is available
	md5URL := url + ".md5"

	resp, con, err := f.downloadInNamespace(defaultNamespace, md5URL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get flist hash from '%s'", md5URL)
	}

	defer func() {
		resp.Body.Close()
		con.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrHashNotSupported
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get flist hash: %s", resp.Status)
	}

	hash, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	hashStr := strings.TrimSpace(string(hash))
	if len(hashStr) != md5HexLength {
		return "", ErrHashInvalidLen
	}

	return hashStr, nil
}

func (f *flistModule) downloadFlist(url, namespace string) (Hash, Path, error) {
	// the problem here is that the same url (to an flist) might
	// be completely differnet flists. because the flist was update
	// on remote. so we can't optimize the download by avoiding redownloading
	// the flist if the same url was downloaded before.
	// this is also why flists are stored locally with hashes.
	// While the hub allows us to get the md5sum of an flist, other hosts
	// don't do that. So we will always need to download the flist anyway, maybe
	// optimize in case of the hub.

	// for now we re-download every time and compute the hash on the fly

	// we don't have the flist locally yet, let's download it

	resp, con, err := f.downloadInNamespace(namespace, url)
	if err != nil {
		return "", "", err
	}

	defer func() {
		resp.Body.Close()
		if con != nil {
			con.Close()
		}
	}()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("fail to download flist: %v", resp.Status)
	}

	return f.saveFlist(resp.Body)
}

// saveFlist save the flist contained in r
// it save the flist by its md5 hash
// to avoid loading the full flist in memory to compute the hash
// it uses a MultiWriter to write the flist in a temporary file and fill up
// the md5 hash then it rename the file to the hash
func (f *flistModule) saveFlist(r io.Reader) (Hash, Path, error) {
	tmp, err := os.CreateTemp(f.flist, "*_flist_temp")
	if err != nil {
		return "", "", err
	}
	defer tmp.Close()

	h := md5.New()
	mr := io.MultiWriter(tmp, h)
	if _, err := io.Copy(mr, r); err != nil {
		return "", "", err
	}

	hash := fmt.Sprintf("%x", h.Sum(nil))
	path := filepath.Join(f.flist, hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", "", err
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", "", err
	}

	return Hash(hash), Path(path), nil
}

var _ pkg.Flister = (*flistModule)(nil)

func (f *flistModule) downloadInNamespace(name, u string) (resp *http.Response, con net.Conn, err error) {
	if len(name) == 0 {
		resp, err = f.httpClient.Get(u)
		return
	}

	namespace, err := namespace.GetByName(name)
	if err != nil {
		return resp, con, errors.Wrapf(err, "failed to get namespace %s", name)
	}

	err = namespace.Do(func(_ ns.NetNS) error {
		hostPort, err := parseURL(u)
		if err != nil {
			return err
		}

		con, err = net.Dial("tcp", hostPort)
		if err != nil {
			return errors.Wrap(err, "failed to start tcp connection")
		}

		cl := http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return con, nil
				},
			},
		}

		resp, err = cl.Get(u)
		return err
	})

	return
}

func parseURL(u string) (hostPort string, err error) {
	rawURL, err := url.Parse(u)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse url")
	}

	port := rawURL.Port()
	if port == "" {
		p, err := net.LookupPort("tcp", rawURL.Scheme)
		if err != nil {
			return "", errors.Wrap(err, "failed to lookup url port")
		}
		port = fmt.Sprint(p)
	}

	hostPort = net.JoinHostPort(rawURL.Hostname(), port)
	return hostPort, nil
}
