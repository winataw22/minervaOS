package main

import (
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"os"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/tnodb"
	"github.com/urfave/cli"
)

var (
	idStore identity.IDStore
	db      network.TNoDB
)

func main() {

	app := cli.NewApp()
	app.Usage = "Create and manage a Threefold farm"
	app.Version = "0.0.1"
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug logging",
		},
		cli.StringFlag{
			Name:   "bcdb, b",
			Usage:  "URL of the BCDB",
			Value:  "https://explorer.devnet.grid.tf",
			EnvVar: "BCDB_URL",
		},
	}
	app.Before = func(c *cli.Context) error {
		debug := c.Bool("debug")
		if !debug {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		url := c.String("bcdb")
		idStore = identity.NewHTTPIDStore(url)
		db = tnodb.NewHTTPTNoDB(url)

		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "farm",
			Usage: "Manage and create farms",
			Subcommands: []cli.Command{
				{
					Name:      "register",
					Usage:     "register a new farm",
					Category:  "identity",
					ArgsUsage: "farm_name",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "seed",
							Usage: "path to the farmer seed. Specify this if you already have a seed generated for your farm",
						},
					},
					Action: registerFarm,
				},
			},
		},
		{
			Name:  "network",
			Usage: "Manage network of a farm and hand out allocation to the grid",
			Subcommands: []cli.Command{
				{
					Name:     "configure-public",
					Category: "network",
					Usage: `configure the public interface of a node.
You can specify multime time the ip and gw flag to configure multiple IP on the public interface`,
					ArgsUsage: "node ID",
					Flags: []cli.Flag{
						cli.StringSliceFlag{
							Name:  "ip",
							Usage: "ip address to set to the exit interface",
						},
						cli.StringSliceFlag{
							Name:  "gw",
							Usage: "gw address to set to the exit interface",
						},
						cli.StringFlag{
							Name:  "iface",
							Usage: "name of the interface to use as public interface",
						},
					},
					Action: configPublic,
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func loadFarmID(seedPath string) (pkg.Identifier, error) {
	if seedPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		seedPath = filepath.Join(cwd, "farm.seed")
	}

	log.Debug().Msgf("loading seed from %s", seedPath)
	farmID, err := identity.LoadKeyPair(seedPath)
	if err != nil {
		return nil, err
	}

	return farmID, nil
}

func generateKeyPair(seedPath string) (pkg.Identifier, error) {
	log.Debug().Msg("generating new key pair")
	keypair, err := identity.GenerateKeyPair()
	if err != nil {
		log.Error().Err(err).Msg("fail to generate key pair for farm identity")
		return nil, err
	}

	if seedPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		seedPath = filepath.Join(cwd, "farm.seed")
	}

	if err := keypair.Save(seedPath); err != nil {
		log.Error().Err(err).Msg("fail to save identity seed on disk")
		return nil, err
	}

	return keypair, nil
}
