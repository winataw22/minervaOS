package flist

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
)

const (
	defaultStorage = "zdb://hub.grid.tf:9900"
	defaultRoot    = "/var/cache/modules/flist"
)

const mib = 1024 * 1024

type flistModule struct {
	// root directory where all
	// the working file of the module will be located
	root string

	// underneath are the path for each
	// sub folder used by the flist module
	flist      string
	cache      string
	mountpoint string
	pid        string
	log        string

	storage modules.VolumeAllocater
}

// New creates a new flistModule
func New(root string, storage modules.VolumeAllocater) modules.Flister {
	if root == "" {
		root = defaultRoot
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		panic(err)
	}

	// prepare directory layout for the module
	for _, path := range []string{"flist", "cache", "mountpoint", "pid", "log"} {
		if err := os.MkdirAll(filepath.Join(root, path), 0770); err != nil {
			panic(err)
		}
	}

	return &flistModule{
		root:       root,
		flist:      filepath.Join(root, "flist"),
		cache:      filepath.Join(root, "cache"),
		mountpoint: filepath.Join(root, "mountpoint"),
		pid:        filepath.Join(root, "pid"),
		log:        filepath.Join(root, "log"),

		storage: storage,
	}
}

// Mount implements the Flister.Mount interface
func (f *flistModule) Mount(url, storage string) (string, error) {
	sublog := log.With().Str("url", url).Str("storage", storage).Logger()
	sublog.Info().Msg("request to mount flist")

	if storage == "" {
		storage = defaultStorage
	}

	flistPath, err := f.downloadFlist(url)
	if err != nil {
		sublog.Err(err).Msg("fail to download flist")
		return "", err
	}

	rnd, err := random()
	if err != nil {
		sublog.Error().Err(err).Msg("fail to generate random id for the mount")
		return "", err
	}

	path, err := f.storage.CreateFilesystem(rnd, 256*mib, modules.SSDDevice)
	if err != nil {
		return "", errors.Wrap(err, "failed to create read-write subvolume for 0-fs")
	}

	mountpoint := filepath.Join(f.mountpoint, rnd)
	if err := os.MkdirAll(mountpoint, 0770); err != nil {
		return "", err
	}
	pidPath := filepath.Join(f.pid, rnd) + ".pid"
	logPath := filepath.Join(f.log, rnd) + ".log"

	args := []string{
		"-backend", path,
		"-cache", f.cache,
		"-meta", flistPath,
		"-storage-url", storage,
		"-daemon",
		"-pid", pidPath,
		"-log", logPath,
		mountpoint,
	}
	sublog.Info().Strs("args", args).Msg("starting 0-fs daemon")
	cmd := exec.Command("g8ufs", args...)

	if out, err := cmd.CombinedOutput(); err != nil {
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

	return mountpoint, nil
}

// Umount implements the Flister.Umount interface
func (f *flistModule) Umount(path string) error {
	log.Info().Str("path", path).Msg("request unmount flist")

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("specified path is not a directory")
	}

	if !strings.HasPrefix(path, f.root) {
		return fmt.Errorf("trying to unmount a directory outside of the flist module boundaries")
	}

	if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
		log.Error().Err(err).Str("path", path).Msg("fail to umount flist")
	}
	_, name := filepath.Split(path)
	pidPath := filepath.Join(f.pid, name) + ".pid"
	if err := waitPidFile(time.Second*2, pidPath, false); err != nil {
		log.Error().Err(err).Str("path", path).Msg("0-fs daemon did not stopped properly")
		return err
	}

	// clean up working dirs
	logPath := filepath.Join(f.log, name) + ".log"
	for _, path := range []string{logPath, path} {
		if err := os.RemoveAll(path); err != nil {
			log.Error().Err(err).Msg("fail to remove %s")
			return err
		}
	}
	// clean up subvolume
	if err := f.storage.ReleaseFilesystem(name); err != nil {
		return errors.Wrap(err, "fail to clean up subvolume")
	}

	return nil
}

// downloadFlist downloads an flits from a URL
// if the flist location also provide and md5 hash of the flist
// this function will use it to avoid downloading an flist that is
// already present locally
func (f *flistModule) downloadFlist(url string) (string, error) {
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

		flistPath := filepath.Join(f.flist, strings.TrimSpace(string(hash)))
		_, err = os.Stat(flistPath)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil {
			log.Info().Str("url", url).Msg("flist already in cache")
			// flist is already present locally, just return its path
			return flistPath, nil
		}
	}

	log.Info().Str("url", url).Msg("flist not in cache, downloading")
	// we don't have the flist locally yet, let's download it
	resp, err = http.Get(url)
	if err != nil {
		return "", err
	}

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
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return "", err
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}

	return path, nil
}

func random() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return fmt.Sprintf("%x", b), err
}

// waitPidFile wait for a file pointed by path to be created or deleted
// for at most timeout duration
// is exists is true, it waits for the file to exists
// else it waits for the file to be deleted
func waitPidFile(timeout time.Duration, path string, exists bool) error {
	const delay = time.Millisecond * 100
	cErr := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				cErr <- ctx.Err()
			default:
				_, err := os.Stat(path)
				if exists {
					if err != nil {
						time.Sleep(delay)
						continue
					}
					cErr <- nil
				} else {
					if err == nil {
						time.Sleep(delay)
						continue
					}
					cErr <- nil
				}
			}
		}
	}()

	return <-cErr
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

var _ modules.Flister = (*flistModule)(nil)
