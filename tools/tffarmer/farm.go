package main

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/urfave/cli"
)

func registerFarm(c *cli.Context) (err error) {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("farm name needs to be specified")
	}

	addrs := c.StringSlice("addresses")
	email := c.String("email")
	iyo := c.String("iyo_organization")

	addresses := make([]directory.WalletAddress, len(addrs))
	for i := range addrs {
		addresses[i].Asset, addresses[i].Address, err = splitAddressCode(addrs[i])
		if err != nil {
			return err
		}
	}

	farm := directory.Farm{
		Name:            name,
		ThreebotId:      int64(userid.ThreebotID),
		Email:           schema.Email(email),
		IyoOrganization: iyo,
		WalletAddresses: addresses,
	}

	farm.ID, err = db.FarmRegister(farm)
	if err != nil {
		return err
	}

	fmt.Println("Farm updated successfully")
	fmt.Println(formatFarm(farm))
	return nil
}

func updateFarm(c *cli.Context) error {
	id := c.Int64("id")
	farm, err := db.FarmGet(schema.ID(id))
	if err != nil {
		return err
	}

	addrs := c.StringSlice("addresses")
	email := c.String("email")
	iyo := c.String("iyo_organization")

	if len(addrs) > 0 {
		addresses := make([]directory.WalletAddress, len(addrs))
		for i := range addrs {
			addresses[i].Asset, addresses[i].Address, err = splitAddressCode(addrs[i])
			if err != nil {
				return err
			}
		}
		farm.WalletAddresses = addresses
	}

	if email != "" {
		farm.Email = schema.Email(email)
	}

	if iyo != "" {
		farm.IyoOrganization = iyo
	}

	if err := db.FarmUpdate(farm); err != nil {
		return err
	}

	fmt.Println("Farm registered successfully")
	fmt.Println(formatFarm(farm))
	return nil
}

func splitAddressCode(addr string) (string, string, error) {
	ss := strings.Split(addr, ":")
	if len(ss) != 2 {
		return "", "", fmt.Errorf("wrong format for wallet address %s, should be 'asset:address'", addr)
	}

	return ss[0], ss[1], nil
}
func formatFarm(farm directory.Farm) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "ID: %d\n", farm.ID)
	fmt.Fprintf(b, "Name: %s\ns", farm.Name)
	fmt.Fprintf(b, "Email: %s\n", farm.Email)
	fmt.Fprintf(b, "Farmer TheebotID: %d\n", farm.ThreebotId)
	fmt.Fprintf(b, "IYO organization: %s\n", farm.IyoOrganization)
	fmt.Fprintf(b, "Wallet addresses:\n")
	for _, a := range farm.WalletAddresses {
		fmt.Fprintf(b, "%s:%s\n", a.Asset, a.Address)
	}
	return b.String()
}
