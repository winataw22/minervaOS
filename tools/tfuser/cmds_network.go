package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/threefoldtech/zosv2/modules/provision"

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

	return output(c.GlobalString("schema"), r)
}

func cmdsAddNode(c *cli.Context) error {
	var (
		network = &modules.Network{}
		schema  = c.GlobalString("schema")
		err     error
	)

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	for _, nodeID := range c.StringSlice("node") {
		network, err = addNode(network, nodeID)
		if err != nil {
			return errors.Wrap(err, "failed to add the node into the network object")
		}
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(schema, r)
}
func cmdsAddUser(c *cli.Context) error {
	var (
		network    = &modules.Network{}
		schema     = c.GlobalString("schema")
		userID     = c.String("user")
		privateKey string
		err        error
	)

	if userID == "" {
		return fmt.Errorf("user ID cannot be empty. generate an identity using the `id` command")
	}

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	network, privateKey, err = addUser(network, userID)
	if err != nil {
		return errors.Wrap(err, "failed to add the node into the network object")
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	fmt.Printf("wireguard private key: %s\n", privateKey)
	fmt.Printf("save this key somewhere, you will need it to generate the wg-quick configuration file with the `wg` command\n")

	return output(schema, r)
}

func cmdsWGQuick(c *cli.Context) error {
	var (
		network    = &modules.Network{}
		schema     = c.GlobalString("schema")
		userID     = c.String("user")
		privateKey = c.String("key")
		err        error
	)

	if privateKey == "" {
		return fmt.Errorf("private key cannot be empty")
	}

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	out, err := genWGQuick(network, userID, privateKey)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

func cmdsRemoveNode(c *cli.Context) error {
	var (
		network = &modules.Network{}
		schema  = c.GlobalString("schema")
		nodeID  = c.String("node")
		err     error
	)

	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	network, err = removeNode(network, nodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to remove node %s from the network object", nodeID)
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(schema, r)
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
