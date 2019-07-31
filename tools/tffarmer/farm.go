package main

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/urfave/cli"
)

func registerFarm(c *cli.Context) error {

	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("A farm name needs to be specified")
	}

	var farmID modules.Identifier
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

	if err := idStore.RegisterFarm(farmID, name); err != nil {
		return err
	}
	fmt.Println("Farm registered successfully")
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Identity: %s\n", farmID.Identity())
	return nil
}
