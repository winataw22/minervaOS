package main

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/threefoldtech/zosv2/modules/provision"
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

	if disktype != modules.HDDDevice && disktype != modules.SSDDevice {
		return fmt.Errorf("volume type can only 'HHD' or 'SSD'")
	}

	if mode != modules.ZDBModeSeq && mode != modules.ZDBModeUser {
		return fmt.Errorf("mode can only 'user' or 'seq'")
	}

	if size < 1 { //TODO: upper bound ?
		return fmt.Errorf("size cannot be less than 1")
	}

	zdb := provision.ZDB{
		Size:     size,
		DiskType: modules.DeviceType(disktype),
		Mode:     modules.ZDBMode(mode),
		Password: password,
		Public:   public,
	}

	p, err := embed(zdb, provision.ZDBReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), p)
}
