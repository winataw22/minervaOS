package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/threefoldtech/zosv2/modules/provision"

	"github.com/threefoldtech/zosv2/modules/identity"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/urfave/cli"
)

func cmdCreateNetwork(c *cli.Context) error {
	network, err := createNetwork(c.String("node"))
	if err != nil {
		return err
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), r)
}

func cmdsAddNode(c *cli.Context) error {
	var (
		network = &modules.Network{}
		input   = c.GlobalString("input")
		port    = c.Uint("port")
		err     error
	)

	network, err = loadNetwork(input)
	if err != nil {
		return err
	}

	for _, nodeID := range c.StringSlice("node") {
		network, err = addNode(network, nodeID, uint16(port))
		if err != nil {
			return errors.Wrap(err, "failed to add the node into the network object")
		}
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), r)
}
func cmdsAddUser(c *cli.Context) error {
	var (
		network = &modules.Network{}
		input   = c.GlobalString("input")
		userID  = c.String("user")
		err     error
	)

	if userID == "" {
		k, err := identity.GenerateKeyPair()
		if err != nil {
			return err
		}
		userID = k.Identity()
	}

	network, err = loadNetwork(input)
	if err != nil {
		return err
	}

	network, err = addUser(network, userID)
	if err != nil {
		return errors.Wrap(err, "failed to add the node into the network object")
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), r)
}

func loadNetwork(name string) (network *modules.Network, err error) {
	network = &modules.Network{}

	if name == "" {
		return nil, fmt.Errorf("schema name cannot be empty")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := &provision.Reservation{}
	if err := json.NewDecoder(f).Decode(r); err != nil {
		return nil, errors.Wrapf(err, "failed to decode json encoded reservation at %s", name)
	}

	if err := json.Unmarshal(r.Data, network); err != nil {
		return nil, errors.Wrapf(err, "failed to decode json encoded network at %s", name)
	}
	return network, nil
}
