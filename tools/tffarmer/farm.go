package main

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg"
	"github.com/urfave/cli"
)

func registerFarm(c *cli.Context) error {

	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("A farm name needs to be specified")
	}

	var farmID pkg.Identifier
	var err error
	seedPath := c.String("seed")
	if seedPath != "" {
		farmID, err = loadFarmID(seedPath)
		if err != nil {
			return err
		}
	}
	if farmID == nil {
		farmID, err = generateKeyPair(seedPath)
		if err != nil {
			return err
		}
	}

	if _, err := idStore.RegisterFarm(farmID, name, "", []string{}); err != nil {
		return err
	}
	fmt.Println("Farm registered successfully")
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Identity: %s\n", farmID.Identity())
	return nil
}
