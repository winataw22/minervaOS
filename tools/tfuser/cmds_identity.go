package main

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/phonebook"
	"github.com/urfave/cli"
)

func cmdsGenerateID(c *cli.Context) error {

	output := c.String("output")
	name := c.String("name")
	email := c.String("email")
	description := c.String("description")

	k, err := identity.LoadKeyPair(output)
	if err == nil {
		fmt.Printf("a seed already exists at %s\n", output)
		fmt.Printf("identity: %s\n", k.Identity())
		return nil
	}

	k, err = identity.GenerateKeyPair()
	if err != nil {
		return err
	}

	user := phonebook.User{
		Name:        name,
		Email:       email,
		Signature:   hex.EncodeToString(k.PublicKey),
		Description: description,
	}
	id, err := bcdb.Phonebook.Create(user)
	if err != nil {
		return errors.Wrap(err, "failed to register user")
	}

	if err := k.Save(c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}

	fmt.Printf("Your ID is: %d\n", id)
	fmt.Printf("seed saved to: %s\n", output)
	return nil
}
