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
	db     client.Directory
	userid = &identity.UserIdentity{}
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
			Name:  "seed",
			Usage: "seed filename",
			Value: "user.seed",
		},
		cli.StringFlag{
			Name:   "bcdb, b",
			Usage:  "URL of the BCDB",
			Value:  "https://explorer.devnet.grid.tf/explorer",
			EnvVar: "BCDB_URL",
		},
	}

	app.Before = func(c *cli.Context) error {
		var err error

		debug := c.Bool("debug")
		if !debug {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		err = userid.Load(c.String("seed"))
		if err != nil {
			return err
		}

		url := c.String("bcdb")
		cl, err := client.NewClient(url, userid)
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
						cli.StringSliceFlag{
							Name:     "addresses",
							Usage:    "wallet address",
							Required: true,
						},
						cli.StringSliceFlag{
							Name:     "email",
							Usage:    "email address of the farmer. It is used to send communication to the farmer and for the minting",
							Required: true,
						},
						cli.StringSliceFlag{
							Name:     "iyo_organization",
							Usage:    "the It'sYouOnline organization used by your farm in v1",
							Required: false,
						},
					},
					Action: registerFarm,
				},
				{
					Name:     "update",
					Usage:    "update an existing farm",
					Category: "identity",
					Flags: []cli.Flag{
						cli.Int64Flag{
							Name:     "id",
							Usage:    "farm ID",
							Required: true,
						},
						cli.StringSliceFlag{
							Name:     "addresses",
							Usage:    "wallet address. the format is 'asset:address: e.g: 'TFT:GBUPOYJ7I4D4TYSFXPJNLSATHCCF2QDDQCIIIXBG7CV7S2U36UMAQENV'",
							Required: false,
						},
						cli.StringSliceFlag{
							Name:     "email",
							Usage:    "email address of the farmer. It is used to send communication to the farmer and for the minting",
							Required: false,
						},
						cli.StringSliceFlag{
							Name:     "iyo_organization",
							Usage:    "the It'sYouOnline organization used by your farm in v1",
							Required: false,
						},
					},
					Action: updateFarm,
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
		{
			Name:  "nodes",
			Usage: "Manage nodes from a farm",
			Subcommands: []cli.Command{
				{
					Name:     "free",
					Category: "nodes",
					Usage:    "mark some nodes as free to use",
					Flags: []cli.Flag{
						cli.StringSliceFlag{
							Name:  "nodes",
							Usage: "node IDs. can be specified multiple time",
						},
						cli.BoolFlag{
							Name:  "free",
							Usage: "if set, the node is marked free, it not the node is mark not free",
						},
					},
					Action: markFree,
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}
