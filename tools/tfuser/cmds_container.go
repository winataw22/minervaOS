package main

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/urfave/cli"
)

func generateContainer(c *cli.Context) error {

	envs, err := splitEnvs(c.StringSlice("envs"))
	if err != nil {
		return err
	}

	mounts, err := splitMounts(c.StringSlice("mounts"))
	if err != nil {
		return err
	}

	container := provision.Container{
		FList:        c.String("flist"),
		FlistStorage: c.String("storage"),
		Env:          envs,
		Entrypoint:   c.String("entrypoint"),
		Interactive:  c.Bool("corex"),
		Mounts:       mounts,
		Network: provision.Network{
			NetwokID: modules.NetID(c.String("network")),
		},
	}

	if err := validateContainer(container); err != nil {
		return err
	}

	p, err := embed(container, provision.ContainerReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), p)
}

func validateContainer(c provision.Container) error {
	if c.FList == "" {
		return fmt.Errorf("flist cannot be empty")
	}
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
