package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/g0rbe/go-chattr"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

const (
	// vdiskVolumeName is the name of the volume used to store vdisks
	vdiskVolumeName = "vdisks"

	mib = 1024 * 1024
)

type vdiskModule struct {
	module *Module
}

// NewVDiskModule creates a new disk allocator
func NewVDiskModule(module *Module) (pkg.VDiskModule, error) {
	return &vdiskModule{module: module}, nil
}

func (d *vdiskModule) findDisk(id string) (string, error) {
	pools, err := d.module.VDiskPools()
	if err != nil {
		return "", errors.Wrapf(err, "failed to find disk with id '%s'", id)
	}

	for _, pool := range pools {
		path, err := d.safePath(pool, id)
		if err != nil {
			return "", err
		}

		if _, err := os.Stat(path); err == nil {
			// file exists
			return path, nil
		}
	}

	return "", os.ErrNotExist
}

// AllocateDisk with given size, return path to virtual disk (size in MB)
func (d *vdiskModule) Allocate(id string, size int64) (string, error) {
	path, err := d.findDisk(id)
	if err == nil {
		return path, errors.Wrapf(os.ErrExist, "disk with id '%s' already exists", id)
	}

	base, err := d.module.VDiskFindCandidate(uint64(size))
	if err != nil {
		return "", errors.Wrapf(err, "failed to find a candidate to host vdisk of size '%d'", size)
	}

	path, err = d.safePath(base, id)
	if err != nil {
		return "", err
	}

	defer func() {
		// clean up disk file if error
		if err != nil {
			os.RemoveAll(path)
		}
	}()

	var file *os.File
	file, err = os.Create(path)
	if err != nil {
		return "", err
	}

	defer file.Close()
	if err = chattr.SetAttr(file, chattr.FS_NOCOW_FL); err != nil {
		return "", err
	}

	err = syscall.Fallocate(int(file.Fd()), 0, 0, size*mib)
	return path, err
}

func (d *vdiskModule) safePath(base, id string) (string, error) {
	path := filepath.Join(base, id)
	// this to avoid passing an `injection` id like '../name'
	// and end up deleting a file on the system. so only delete
	// allocated disks
	location := filepath.Dir(path)
	if filepath.Clean(location) != base {
		return "", fmt.Errorf("invalid disk id: '%s'", id)
	}

	return path, nil
}

// DeallocateVDisk removes a virtual disk
func (d *vdiskModule) Deallocate(id string) error {
	path, err := d.findDisk(id)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// DeallocateVDisk removes a virtual disk
func (d *vdiskModule) Exists(id string) bool {
	_, err := d.findDisk(id)

	return err == nil
}

// Inspect return info about the disk
func (d *vdiskModule) Inspect(id string) (disk pkg.VDisk, err error) {
	path, err := d.findDisk(id)

	if err != nil {
		return disk, err
	}

	disk.Path = path
	stat, err := os.Stat(path)
	if err != nil {
		return disk, err
	}

	disk.Size = stat.Size()
	return
}

func (d *vdiskModule) List() ([]pkg.VDisk, error) {
	pools, err := d.module.VDiskPools()
	if err != nil {
		return nil, err
	}
	var disks []pkg.VDisk
	for _, pool := range pools {

		items, err := ioutil.ReadDir(pool)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list virtual disks")
		}

		for _, item := range items {
			if item.IsDir() {
				continue
			}

			disks = append(disks, pkg.VDisk{
				Path: filepath.Join(pool, item.Name()),
				Size: item.Size(),
			})
		}

		return disks, nil
	}

	return disks, nil
}
