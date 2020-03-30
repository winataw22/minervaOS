package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/container/stats"
	"github.com/threefoldtech/zos/pkg/provision"
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

	cap := provision.ContainerCapacity{
		CPU:    c.Uint("cpu"),
		Memory: c.Uint64("memory"),
	}

	var sts []stats.Aggregator
	if s := c.String("stats"); s != "" {
		// validating stdout argument
		_, _, err := logger.RedisParseURL(s)
		if err != nil {
			return err
		}

		ss := stats.Aggregator{
			Type: stats.RedisType,
			Data: stats.Redis{
				Endpoint: s,
			},
		}

		sts = append(sts, ss)
	}

	var logs []logger.Logs
	if lo := c.String("stdout"); lo != "" {
		// validating stdout argument
		_, _, err := logger.RedisParseURL(lo)
		if err != nil {
			return err
		}

		// copy stdout to stderr
		lr := lo

		// check if stderr is specified
		if nlr := c.String("stderr"); nlr != "" {
			// validating stderr argument
			_, _, err := logger.RedisParseURL(nlr)
			if err != nil {
				return nil
			}

			lr = nlr
		}

		lg := logger.Logs{
			Type: "redis",
			Data: logger.LogsRedis{
				Stdout: lo,
				Stderr: lr,
			},
		}

		logs = append(logs, lg)
	}

	container := provision.Container{
		FList:        c.String("flist"),
		FlistStorage: c.String("storage"),
		Env:          envs,
		Entrypoint:   c.String("entrypoint"),
		Interactive:  c.Bool("corex"),
		Mounts:       mounts,
		Network: provision.Network{
			NetworkID: pkg.NetID(c.String("network")),
			IPs: []net.IP{
				net.ParseIP(c.String("ip")),
			},
			PublicIP6: c.Bool("public6"),
		},
		Capacity:        cap,
		Logs:            logs,
		StatsAggregator: sts,
	}

	if err := validateContainer(container); err != nil {
		return err
	}

	p, err := embed(container, provision.ContainerReservation, c.String("node"))
	if err != nil {
		return err
	}

	return writeWorkload(c.GlobalString("output"), p)
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
