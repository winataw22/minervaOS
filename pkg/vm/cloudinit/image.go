package cloudinit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/google/shlex"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// CreateImage creates a fat32 image for cloud-init that
// has
func CreateImage(file string, cfg Configuration) error {
	if _, err := os.Stat(file); err == nil {
		if err := os.Remove(file); err != nil {
			return errors.Wrapf(err, "failed to clear up disk file '%s'", file)
		}
	}

	device, err := diskfs.Create(file, 2*fat32.MB, diskfs.Raw)
	if err != nil {
		return errors.Wrap(err, "failed to create vfat disk")
	}

	defer device.File.Close()

	fs, err := device.CreateFilesystem(disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: "cidata",
	})

	if err != nil {
		return errors.Wrap(err, "failed to create vafat filesystem")
	}

	write := func(path string, object interface{}) error {
		meta, err := fs.OpenFile(path, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return errors.Wrapf(err, "failed to create %s file", path)
		}
		defer meta.Close()

		if _, err := meta.Write([]byte("#cloud-config\n")); err != nil {
			return errors.Wrap(err, "failed to write config header")
		}

		if err := yaml.NewEncoder(meta).Encode(object); err != nil {
			return errors.Wrapf(err, "failed to write %s file", path)
		}
		return nil
	}

	if err := write("/meta-data", cfg.Metadata); err != nil {
		return err
	}

	if err := write("/network-config", marsh{
		"version": 2,
		"ethernets": func() marsh {
			m := marsh{}
			for _, ifc := range cfg.Network {
				m[ifc.Name] = ifc
			}
			return m
		}(),
	}); err != nil {
		return err
	}

	if err := write("/user-data", marsh{
		"users":  cfg.Users,
		"mounts": cfg.Mounts,
	}); err != nil {
		return err
	}

	// finally the zosrc file
	rc, err := fs.OpenFile("/zosrc", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return errors.Wrapf(err, "failed to create /zosrc file")
	}
	defer rc.Close()

	return cfg.Extension.write(rc)
}

// transpiled from https://github.com/python/cpython/blob/3.10/Lib/shlex.py#L325
func quote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func (e *Extension) write(file io.Writer) error {
	for k, v := range e.Environment {
		if _, err := fmt.Fprintf(file, "export %s=%s\n", k, quote(v)); err != nil {
			return err
		}
	}

	parts, err := shlex.Split(e.Entrypoint)
	if err != nil {
		return errors.Wrap(err, "invalid entrypoint")
	}
	if len(parts) != 0 {
		if _, err := fmt.Fprintf(file, "init=%s\n", quote(parts[0])); err != nil {
			return err
		}
	}
	if len(parts) > 1 {
		var buf bytes.Buffer
		buf.WriteString("set --")
		for _, part := range parts[1:] {
			buf.WriteRune(' ')
			buf.WriteString(quote(part))
		}
		if _, err := fmt.Fprint(file, buf.String()); err != nil {
			return err
		}

		fmt.Fprint(file, "\n")
	}
	return nil
}
