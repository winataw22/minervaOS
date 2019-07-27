package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"

	"github.com/urfave/cli"
)

func cmdsProvision(c *cli.Context) error {
	var (
		schema  []byte
		path    = c.String("schema")
		nodeIDs = c.StringSlice("node")
		err     error
	)

	if path == "-" {
		schema, err = ioutil.ReadAll(os.Stdin)
	} else {
		schema, err = ioutil.ReadFile(path)
	}
	if err != nil {
		return err
	}

	r := provision.Reservation{}
	if err := json.Unmarshal(schema, &r); err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r.ID = id.String()

	if err := output(path, r); err != nil {
		return err
	}

	for _, nodeID := range nodeIDs {
		if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
			return err
		}
		fmt.Printf("reservation send for node %s\n", nodeID)
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
