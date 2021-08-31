package flist

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	defaultRoot = "/var/cache/modules/flist"
)

const mib = 1024 * 1024

type commander interface {
	Command(name string, arg ...string) *exec.Cmd
}

type cmd func(name string, arg ...string) *exec.Cmd

func (c cmd) Command(name string, args ...string) *exec.Cmd {
	return c(name, args...)
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
	run        string

	storage   volumeAllocator
	commander commander
	system    system
}

func newFlister(root string, storage volumeAllocator, commander commander, system system) pkg.Flister {
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
	for _, path := range []string{"flist", "cache", "mountpoint", "ro", "pid", "log", "run"} {
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
		run:        filepath.Join(root, "run"),

		storage:   storage,
		commander: commander,
		system:    system,
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
func (f *flistModule) mountRO(url, storage string) (string, error) {
	// this should return always the flist mountpoint. which is used
	// as a base for all RW mounts.
	sublog := log.With().Str("url", url).Str("storage", storage).Logger()
	sublog.Info().Msg("request to mount flist")

	hash, err := f.FlistHash(url)
	if err != nil {
		return "", errors.Wrap(err, "failed to get flist hash")
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

	flistPath, err := f.downloadFlist(url)
	if err != nil {
		sublog.Err(err).Msg("fail to download flist")
		return "", err
	}

	pidPath := filepath.Join(f.pid, hash) + ".pid"
	logPath := filepath.Join(f.log, hash) + ".log"
	var args []string

	args = append(args,
		"-cache", f.cache,
		"-meta", flistPath,
		"-storage-url", storage,
		"-daemon",
		"-pid", pidPath,
		"-log", logPath,
		// this is always read-only
		"-ro",
		mountpoint,
	)

	sublog.Info().Strs("args", args).Msg("starting 0-fs daemon")
	cmd := f.commander.Command("g8ufs", args...)

	var out []byte
	if out, err = cmd.CombinedOutput(); err != nil {
		sublog.Err(err).Str("out", string(out)).Msg("fail to start 0-fs daemon")
		return "", err
	}

	// wait for the daemon to be ready
	// we check the pid file is created
	if err := waitPidFile(time.Second*5, pidPath, true); err != nil {
		sublog.Error().Err(err).Msg("pid file of 0-fs daemon not created")
		return "", err
	}

	// and scan the logs after "mount ready"
	if err := waitMountedLog(time.Second*5, logPath); err != nil {
		sublog.Error().Err(err).Msg("0-fs daemon did not start properly")
		return "", err
	}

	syscall.Sync()

	// the track file is a symlink to the process pid
	// if the link is broken, then the fs has exited gracefully
	// otherwise we can get the fs pid from the track path
	// if the pid does not exist of the target does not exist
	// we can clean up the named mount
	trackPath := filepath.Join(f.run, hash)
	if err = os.Remove(trackPath); err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, "failed to clean up pid file")
	}

	if err = os.Symlink(pidPath, trackPath); err != nil {
		sublog.Error().Err(err).Msg("failed track fs pid")
	}

	syscall.Sync()

	return mountpoint, nil
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

func (f *flistModule) mountOverlay(ctx context.Context, name, ro string, size gridtypes.Unit) error {
	mountpoint, err := f.mountpath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return errors.Wrap(err, "failed to create overlay mountpoint")
	}

	// check if the filesystem doesn't already exists
	backend, err := f.storage.VolumeLookup(ctx, name)
	newAllocation := false
	if err != nil {
		log.Info().Msgf("create new subvolume %s", name)
		// and only create a new one if it doesn't exist
		if size == 0 {
			// sanity check in case type is not set always use hdd
			return fmt.Errorf("invalid mount option, missing disk type")
		}
		newAllocation = true
		backend, err = f.storage.VolumeCreate(ctx, name, size)
		if err != nil {
			return errors.Wrap(err, "failed to create read-write subvolume for 0-fs")
		}
	}

	defer func() {
		// in case of an error (mount is never fully completed)
		// we need to deallocate the filesystem

		if newAllocation && err != nil {
			_ = f.storage.VolumeDelete(ctx, name)
		}
	}()

	log.Debug().Msgf("backend: %+v", backend)
	rw := filepath.Join(backend.Path, "rw")
	wd := filepath.Join(backend.Path, "wd")
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

func (f *flistModule) Mount(name, url string, opt pkg.MountOptions) (string, error) {
	sublog := log.With().Str("name", name).Str("url", url).Str("storage", opt.Storage).Logger()
	sublog.Info().Msgf("request to mount flist: %+v", opt)

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

	ro, err := f.mountRO(url, opt.Storage)
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
	return mountpoint, f.mountOverlay(ctx, name, ro, opt.Limit)
}

func (f *flistModule) mountpath(name string) (string, error) {
	mountpath := filepath.Join(f.mountpoint, name)
	if filepath.Dir(mountpath) != f.mountpoint {
		return "", errors.New("invalid mount name")
	}

	return mountpath, nil
}

func (f *flistModule) flistMountpath(hash string) (string, error) {
	mountpath := filepath.Join(f.ro, hash)
	if filepath.Dir(mountpath) != f.ro {
		return "", errors.New("invalid mount name")
	}

	return mountpath, nil
}

// ErrAlreadyMounted is returned when checking if a path has already
// something mounted on it
var ErrAlreadyMounted = errors.New("path is already mounted")
var ErrTransportEndpointIsNotConencted = errors.New("transport endpoint is not connected")

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

func (f *flistModule) getPid(pidPath string) (int64, error) {
	pid, err := ioutil.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseInt(string(pid), 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid pid value in file, expected int")
	}

	return value, nil
}

func (f *flistModule) getMountOptions(pidPath string) (options, error) {
	pid, err := f.getPid(pidPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pid from file: %s", pidPath)
	}

	return f.getMountOptionsForPID(pid)
}

func (f *flistModule) getMountOptionsForPID(pid int64) (options, error) {
	cmdline, err := ioutil.ReadFile(path.Join("/proc", fmt.Sprint(pid), "cmdline"))
	if err != nil {
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
	base := filepath.Base(name)
	pidPath := filepath.Join(f.pid, base) + ".pid"

	opts, err := f.getMountOptions(pidPath)
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
	resp, err := http.Get(md5URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		hash, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		cleanhash := strings.TrimSpace(string(hash))
		return cleanhash, nil
	}

	return "", fmt.Errorf("fail to fetch hash, response: %v", resp.StatusCode)
}

// downloadFlist downloads an flits from a URL
// if the flist location also provide and md5 hash of the flist
// this function will use it to avoid downloading an flist that is
// already present locally
func (f *flistModule) downloadFlist(url string) (string, error) {
	// first check if the md5 of the flist is available
	hash, err := f.FlistHash(url)
	if err != nil {
		return "", err
	}

	flistPath := filepath.Join(f.flist, strings.TrimSpace(string(hash)))
	file, err := os.Open(flistPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		defer file.Close()

		log.Info().Str("url", url).Msg("flist already in on the filesystem")
		// flist is already present locally, verify it's still valid
		equal, err := md5Compare(hash, file)
		if err != nil {
			return "", err
		}

		if equal {
			return flistPath, nil
		}

		log.Info().Str("url", url).Msg("flist on filesystem is corrupted, re-downloading it")
	}

	// we don't have the flist locally yet, let's download it
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fail to download flist: %v", resp.Status)
	}

	return f.saveFlist(resp.Body)
}

// saveFlist save the flist contained in r
// it save the flist by its md5 hash
// to avoid loading the full flist in memory to compute the hash
// it uses a MultiWriter to write the flist in a temporary file and fill up
// the md5 hash then it rename the file to the hash
func (f *flistModule) saveFlist(r io.Reader) (string, error) {
	tmp, err := ioutil.TempFile(f.flist, "*_flist_temp")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	h := md5.New()
	mr := io.MultiWriter(tmp, h)
	if _, err := io.Copy(mr, r); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", h.Sum(nil))
	path := filepath.Join(f.flist, hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}

	return path, nil
}

func md5Compare(hash string, r io.Reader) (bool, error) {
	h := md5.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return false, err
	}
	return strings.Compare(fmt.Sprintf("%x", h.Sum(nil)), hash) == 0, nil
}

// waitPidFile wait for a file pointed by path to be created or deleted
// for at most timeout duration
// is exists is true, it waits for the file to exists
// else it waits for the file to be deleted
func waitPidFile(timeout time.Duration, path string, exists bool) error {
	const delay = time.Millisecond * 100

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := os.Stat(path)
			// check exist but the file is not there yet,
			// or check NOT exist but it is still there
			// we try again
			if (exists && os.IsNotExist(err)) || (!exists && err == nil) {
				time.Sleep(delay)
			} else if err != nil && !os.IsNotExist(err) {
				//another error that is NOT IsNotExist.
				return err
			} else {
				return nil
			}
		}
	}
}

func waitMountedLog(timeout time.Duration, logfile string) error {
	const target = "mount ready"
	const delay = time.Millisecond * 500

	f, err := os.Open(logfile)
	if err != nil {
		return err
	}
	defer f.Close()
	br := bufio.NewReader(f)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// this goroutine looks for "mount ready"
	// in the logs of the 0-fs
	cErr := make(chan error)
	go func(ctx context.Context, r io.Reader, cErr chan<- error) {
		for {
			select {
			case <-ctx.Done():
				// ensure we don't leak the goroutine
				cErr <- ctx.Err()
			default:
				line, err := br.ReadString('\n')
				if err != nil {
					time.Sleep(delay)
					continue
				}

				if !strings.Contains(line, target) {
					time.Sleep(delay)
					continue
				}
				// found
				cErr <- nil
				return
			}
		}
	}(ctx, br, cErr)

	return <-cErr
}

func forceStop(pid int) error {
	slog := log.With().Int("pid", pid).Logger()
	slog.Info().Msg("trying to force stop by killing the process")

	p, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}

	if err := p.Signal(syscall.SIGTERM); err != nil {
		slog.Error().Err(err).Msg("failed to send SIGTERM to process")
	}

	time.Sleep(time.Second)
	if pidExist(p) {
		slog.Info().Msgf("process didn't stop gracefully, lets kill it")
		if err := p.Signal(syscall.SIGKILL); err != nil {
			slog.Error().Err(err).Msg("failed to send SIGKILL to process")
		}
	}

	return nil
}

func pidExist(p *os.Process) bool {
	// https://github.com/golang/go/issues/14146#issuecomment-176888204
	return p.Signal(syscall.Signal(0)) == nil
}

var _ pkg.Flister = (*flistModule)(nil)
