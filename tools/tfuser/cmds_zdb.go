package main

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func generateZDB(c *cli.Context) error {
	var (
		size     = c.Uint64("size")
		mode     = c.String("mode")
		password = c.String("password")
		disktype = c.String("type")
		public   = c.Bool("Public")
	)

	if pkg.DeviceType(disktype) != pkg.HDDDevice && pkg.DeviceType(disktype) != pkg.SSDDevice {
		return fmt.Errorf("volume type can only 'HHD' or 'SSD'")
	}

	if mode != pkg.ZDBModeSeq && mode != pkg.ZDBModeUser {
		return fmt.Errorf("mode can only 'user' or 'seq'")
	}

	if size < 1 { //TODO: upper bound ?
		return fmt.Errorf("size cannot be less than 1")
	}

	zdb := provision.ZDB{
		Size:     size,
		DiskType: pkg.DeviceType(disktype),
		Mode:     pkg.ZDBMode(mode),
		Password: password,
		Public:   public,
	}

	p, err := embed(zdb, provision.ZDBReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), p)
}
