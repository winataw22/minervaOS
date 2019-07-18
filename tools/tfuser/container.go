package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/urfave/cli"
)

func createContainer(c *cli.Context) error {

	envs, err := splitEnvs(c.StringSlice("envs"))
	if err != nil {
		return err
	}

	mounts, err := splitMounts(c.StringSlice("mounts"))
	if err != nil {
		return err
	}

	container := provision.Container{
		FList:       c.String("flist"),
		Env:         envs,
		Entrypoint:  c.String("entrypoint"),
		Interactive: c.Bool("corex"),
		Mounts:      mounts,
		Network: provision.Network{
			NetwokID: c.String("network"),
		},
	}

	fmt.Printf("reservation:\n%+v\n", container)
	asn, err := confirm("do you want to reserve this container? [Y/n]")
	if err != nil {
		return err
	}
	if asn != "y" {
		return nil
	}

	raw, err := json.Marshal(container)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := provision.Reservation{
		ID:   id.String(),
		Type: provision.ContainerReservation,
		Data: raw,
	}

	nodeID := c.Args().First()
	if nodeID == "" {
		return fmt.Errorf("missing argument, node ID must be specified")
	}

	if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
		return err
	}

	fmt.Printf("container reservation sent\n")
	return nil
}

func splitEnvs(envs []string) (map[string]string, error) {
	out := make(map[string]string, len(envs))

	for _, env := range envs {
		ss := strings.SplitN(env, "=", 2)
		if len(ss) != 2 {
			return nil, fmt.Errorf("envs flag mal formatted: %v", env)
		}
		out[ss[0]] = ss[1]
	}

	return out, nil
}

func splitMounts(mounts []string) ([]provision.Mount, error) {
	out := make([]provision.Mount, 0, len(mounts))

	for _, mount := range mounts {
		ss := strings.SplitN(mount, ":", 2)
		if len(ss) != 2 {
			return nil, fmt.Errorf("mounts flag mal formatted: %v", mount)
		}

		out = append(out, provision.Mount{
			VolumeID:   ss[0],
			Mountpoint: ss[1],
		})
	}

	return out, nil
}
