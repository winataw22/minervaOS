package main

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"os"

	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/urfave/cli"
)

var (
	db client.Directory
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

		kp, err := identity.LoadKeyPair("user.seed")
		if err != nil {
			return err
		}

		url := c.String("bcdb")
		cl, err := client.NewClient(url, kp)
		if err != nil {
			return errors.Wrap(err, "failed to create client to bcdb")
		}

		db = cl.Directory

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
						cli.Uint64Flag{
							Name:  "tid",
							Usage: "threebot id",
						},
						cli.StringSliceFlag{
							Name:     "address",
							Usage:    "wallet address",
							Required: true,
						},
						cli.StringSliceFlag{
							Name:     "asset",
							Usage:    "wallet address asset (TFT, FreeTFT)",
							Required: true,
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
