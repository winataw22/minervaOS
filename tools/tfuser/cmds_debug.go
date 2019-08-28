package main

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/provision"
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

	r, err := embed(s, provision.DebugReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), r)
}
