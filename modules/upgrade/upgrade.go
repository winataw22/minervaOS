package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/threefoldtech/zosv2/modules"
)

// FIXME: not sure about the public interface of this package yet
// most probably will need to run as a daemon too
func Run(w Watcher, p Publisher) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for upgrade := range w.Watch(ctx, p) {
		if err := applyUpgrade(upgrade); err != nil {
			return err
		}
	}
	return nil
}

func applyUpgrade(upgrade Upgrade) error {
	flister, err := modules.Flist()
	if err != nil {
		return err
	}

	path, err := flister.Mount(upgrade.Flist, "")
	if err != nil {
		return err
	}
	defer func() {
		if err := flister.Umount(path); err != nil {
			log.Printf("failt to umount flist at %s: %v", path, err)
		}
	}()

	// copy file from path to /
	// TODO: what upgrade that fails mid way ?
	return mergeFs(path, "/")
	// TODO:
	// identify which module has been updated
	// if present call migration
	// restart the required module
}

func mergeFs(upgradeRoot, fsRoot string) error {
	return filepath.Walk(upgradeRoot, func(path string, info os.FileInfo, err error) error {
		// trim flist mountpoint from flist path
		destPath := ""
		if path == upgradeRoot {
			destPath = "/"
		} else {
			destPath = path[len(upgradeRoot):]
			if destPath[0] != filepath.Separator {
				destPath = fmt.Sprintf("/%s", path)
			}
		}

		// change root
		p, err := changeRoot(fsRoot, destPath)
		if err != nil {
			return err
		}
		// create directories
		if info.IsDir() {
			if err := os.MkdirAll(p, info.Mode().Perm()); err != nil {
				return err
			}
			return nil
		}

		// upgrade flist should only container directory and regular files
		if !info.Mode().IsRegular() {
			log.Printf("skip %s: not a regular file", path)
			return nil
		}

		// copy the file to final destination
		return copyFile(p, path, info.Mode().Perm())
	})
}

func copyFile(dst, src string, perm os.FileMode) error {
	fSrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fSrc.Close()

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, perm)
	if err != nil {
		return err
	}
	defer fDst.Close()

	_, err = io.Copy(fDst, fSrc)
	return err
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
