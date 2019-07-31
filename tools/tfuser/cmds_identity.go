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

	_, err = identity.LoadSeed(output)
	if err == nil {
		fmt.Printf("a seed already exists at %s\n", output)
		fmt.Printf("identity: %s\n", k.Identity())
		return nil
	}

	if err := k.Save(c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}
	fmt.Printf("new identity generated: %s\n", k.Identity())
	fmt.Printf("seed saved at %s\n", output)
	return nil
}
