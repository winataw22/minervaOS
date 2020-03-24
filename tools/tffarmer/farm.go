package main

import (
	"fmt"

	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"github.com/urfave/cli"
)

func registerFarm(c *cli.Context) error {

	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("farm name needs to be specified")
	}

	tid := c.Uint64("tid")

	farmID, err := db.FarmRegister(directory.Farm{
		Name:            name,
		ThreebotId:      int64(tid),
		WalletAddresses: []string{"fake"},
	})
	if err != nil {
		return err
	}

	fmt.Println("Farm registered successfully")
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Farm ID: %d\n", farmID)
	return nil
}
