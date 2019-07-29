package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/urfave/cli"
)

func cmdsGenerateID(c *cli.Context) error {
	k, err := identity.GenerateKeyPair()
	if err != nil {
		return err
	}

	output := c.String("output")

	if err := identity.SerializeSeed(k, c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}
	fmt.Printf("new identity generated: %s\n", k.Identity())
	fmt.Printf("seed saved at %s\n", output)
	return nil
}
