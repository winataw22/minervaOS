package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"

	"github.com/urfave/cli"
)

var (
	day             = time.Hour * 24
	defaultDuration = day * 30
)

func cmdsProvision(c *cli.Context) error {
	var (
		schema   []byte
		path     = c.String("schema")
		nodeIDs  = c.StringSlice("node")
		seedPath = c.String("seed")
		d        = c.String("duration")
		err      error
	)

	duration, err := time.ParseDuration(d)
	if err != nil {
		nrDays, err := strconv.Atoi(d)
		if err != nil {
			return errors.Wrap(err, "unsupported duration format")
		}
		duration = time.Duration(nrDays) * day
	}

	keypair, err := identity.LoadKeyPair(seedPath)
	if err != nil {
		return errors.Wrapf(err, "could not find seed file at %s", seedPath)
	}

	if path == "-" {
		schema, err = ioutil.ReadAll(os.Stdin)
	} else {
		schema, err = ioutil.ReadFile(path)
	}
	if err != nil {
		return errors.Wrap(err, "could not find provision schema")
	}

	r := &provision.Reservation{}
	if err := json.Unmarshal(schema, r); err != nil {
		return errors.Wrap(err, "failed to read the provision schema")
	}

	r.Duration = duration
	r.Created = time.Now()

	// set the user ID into the reservation schema
	r.User = keypair.Identity()

	if err := r.Sign(keypair.PrivateKey); err != nil {
		return errors.Wrap(err, "failed to sign the reservation")
	}

	if err := output(path, r); err != nil {
		return errors.Wrapf(err, "failed to write provision schema to %s after signature", path)
	}

	for _, nodeID := range nodeIDs {
		id, err := store.Reserve(r, modules.StrIdentifier(nodeID))
		if err != nil {
			return errors.Wrap(err, "failed to send reservation")
		}

		fmt.Printf("Reservation for %v send to node %s\n", duration, nodeID)
		fmt.Printf("Resource: %v\n", id)
	}

	return nil
}

func embed(schema interface{}, t provision.ReservationType) (*provision.Reservation, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	r := &provision.Reservation{
		Type: t,
		Data: raw,
	}

	return r, nil
}
