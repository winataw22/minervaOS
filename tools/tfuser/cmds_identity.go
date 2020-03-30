package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/phonebook"
	"github.com/urfave/cli"
)

func cmdsGenerateID(c *cli.Context) error {
	output := c.String("output")
	name := c.String("name")
	email := c.String("email")
	description := c.String("description")
	mnemonic := c.String("mnemonic")

	ui := &identity.UserIdentity{}

	// Try to load the destination file (not allow overwrite)
	err := ui.Load(output)
	if err == nil {
		fmt.Printf("A seed already exists at: %s\n", output)
		fmt.Printf("Identity: %s\n", ui.Key().Identity())
		fmt.Printf("Threebot ID: %d\n", ui.ThreebotID)
		return nil
	}

	if mnemonic != "" {
		log.Info().Msg("building key using existing mnemonic")

		err = ui.FromMnemonic(mnemonic)
		if err != nil {
			return err
		}

	} else {
		k, err := identity.GenerateKeyPair()
		if err != nil {
			return err
		}

		ui = identity.NewUserIdentity(k, 0)
	}

	user := phonebook.User{
		Name:        name,
		Email:       email,
		Pubkey:      hex.EncodeToString(ui.Key().PublicKey),
		Description: description,
	}

	log.Debug().Msg("initializing client with created key")
	bcdb, err = client.NewClient(bcdbAddr, ui)
	if err != nil {
		return err
	}

	log.Debug().Msg("register user")
	id, err := bcdb.Phonebook.Create(user)
	if err != nil {
		return errors.Wrap(err, "failed to register user")
	}

	// Update UserData with created id
	ui.ThreebotID = uint64(id)

	// Saving new seed struct
	if err := ui.Save(c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}

	fmt.Printf("Your ID is: %d\n", id)
	fmt.Printf("Seed saved to: %s\n", output)

	return nil
}

func cmdsConvertID(c *cli.Context) error {
	source := c.String("source")
	destination := c.String("target")
	tid := c.Int("tid")

	log.Info().Str("source", source).Msg("loading original seed")

	// Load original seed file
	kp, err := identity.LoadLegacyKeyPair(source)
	if err != nil {
		log.Fatal().Err(err).Msg("load key pair")
	}

	// Create new object
	ui := identity.NewUserIdentity(kp, uint64(tid))

	// Save new object
	err = ui.Save(destination)
	if err != nil {
		log.Fatal().Err(err).Msg("saving seed file")
	}

	// Load new key to ensure loads works
	log.Info().Msg("reloading new seed to check")

	newkey := &identity.UserIdentity{}
	err = newkey.Load(destination)
	if err != nil {
		log.Fatal().Err(err).Msg("load user identity")
	}

	if bytes.Equal(newkey.Key().PrivateKey, kp.PrivateKey) {
		log.Info().Msg("keys matches")

	} else {
		log.Error().Msg("keys doesn't matches")
	}

	return nil
}

func cmdsImportID(c *cli.Context) error {
	tid := c.Uint64("tid")
	mnemonic := c.String("mnemonic")
	ui := &identity.UserIdentity{ThreebotID: tid}

	log.Info().Msgf("building key using existing mnemonic '%s'", mnemonic)

	if err := ui.FromMnemonic(mnemonic); err != nil {
		return err
	}

	// Saving new seed struct
	output := c.String("output")
	if err := ui.Save(output); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}

	fmt.Printf("ThreeBot ID  : %d\n", ui.ThreebotID)
	fmt.Printf("Public Key   : %s\n", hex.EncodeToString(ui.Key().PublicKey))
	fmt.Printf("Mnemonic     : %s\n", ui.Mnemonic)
	fmt.Printf("Seed saved to: %s\n", output)
	return nil

}

func cmdsShowID(c *cli.Context) error {
	ui := &identity.UserIdentity{}
	err := ui.Load(mainSeed)

	if err != nil {
		return err
	}

	fmt.Printf("Identify File: %s\n", mainSeed)
	fmt.Printf("ThreeBot ID  : %d\n", ui.ThreebotID)
	fmt.Printf("Public Key   : %s\n", hex.EncodeToString(ui.Key().PublicKey))
	fmt.Printf("Mnemonic     : %s\n", ui.Mnemonic)

	return nil
}
