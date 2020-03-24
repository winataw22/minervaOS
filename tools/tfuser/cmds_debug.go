package main

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func generateDebug(c *cli.Context) error {
	endpoint := c.String("endpoint")
	host, p, err := net.SplitHostPort(endpoint)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return errors.Wrap(err, "port format not valid")
	}

	s := provision.Debug{
		Host:    host,
		Port:    port,
		Channel: c.String("channel"),
	}

	r, err := embed(s, provision.DebugReservation, c.String("node"))
	if err != nil {
		return err
	}

	return writeWorkload(c.GlobalString("output"), r)
}
