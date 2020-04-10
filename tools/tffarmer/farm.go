package main

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/urfave/cli"
)

func registerFarm(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("farm name needs to be specified")
	}

	addrs := c.StringSlice("address")
	assets := c.StringSlice("asset")
	email := c.String("email")
	iyo := c.String("iyo_organization")

	addresses := make([]directory.WalletAddress, len(addrs))
	for i := range addrs {
		addresses[i].Address = addrs[i]
		addresses[i].Asset = assets[i]
	}

	farmID, err := db.FarmRegister(directory.Farm{
		Name:            name,
		ThreebotId:      int64(userid.ThreebotID),
		Email:           schema.Email(email),
		IyoOrganization: iyo,
		WalletAddresses: addresses,
	})
	if err != nil {
		return err
	}

	fmt.Println("Farm registered successfully")
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Farm ID: %d\n", farmID)
	return nil
}
