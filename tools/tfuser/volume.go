package main

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/urfave/cli"
)

func createVolume(c *cli.Context) error {
	s := c.Uint64("size")
	t := c.String("type")
	if t != "HDD" && t != "SSD" {
		return fmt.Errorf("volume type can only HHD or SSD")
	}

	v := provision.Volume{
		Size: s,
		Type: provision.DiskType(t),
	}

	fmt.Printf("reservation:\n%+v\n", v)
	asn, err := confirm("do you want to reserve this volume? [Y/n]")
	if err != nil {
		return err
	}
	if asn != "y" {
		return nil
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := provision.Reservation{
		ID:   id.String(),
		Type: provision.VolumeReservation,
		Data: raw,
	}

	nodeID := c.Args().First()
	if nodeID == "" {
		return fmt.Errorf("missing argument, node ID must be specified")
	}

	if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
		return err
	}

	fmt.Printf("volume reservation sent\n")
	return nil
}
