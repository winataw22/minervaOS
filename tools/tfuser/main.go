package main

import (
	"github.com/threefoldtech/zosv2/modules/provision"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"os"

	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
	"github.com/urfave/cli"
)

var (
	db    network.TNoDB
	store provision.ReservationStore
)

func main() {

	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Usage = "Let you provision capacity on the ThreefoldGrid 2.0"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug logging",
		},

		cli.StringFlag{
			Name:  "tnodb, u",
			Usage: "URL of the TNODB",
			Value: "https://tnodb.dev.grid.tf",
		},

		cli.StringFlag{
			Name:  "provision, p",
			Usage: "URL of the provision store",
			Value: "https://tnodb.dev.grid.tf",
		},
	}
	app.Before = func(c *cli.Context) error {
		debug := c.Bool("debug")
		if !debug {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		db = tnodb.NewHTTPHTTPTNoDB(c.String("tnodb"))
		store = provision.NewhHTTPStore(c.String("provision"))

		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "network",
			Usage: "Manage private networks",
			Subcommands: []cli.Command{
				{
					Name:  "create",
					Usage: "create a new user network",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "farm",
							Usage: "ID of the exit farm to use for this network",
						},
						cli.StringSliceFlag{
							Name:  "node",
							Usage: "node ID of the node where to install this network, you can specify multiple time this flag",
						},
					},
					Action: createNetwork,
				},
				{
					Name:  "add",
					Usage: "add a node to a existing network",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "network",
							Usage: "ID of the network",
						},
						cli.StringSliceFlag{
							Name:  "node",
							Usage: "node ID of the node where to install this network, you can specify multiple time this flag",
						},
					},
					Action: addMember,
				},
				{
					Name:  "local",
					Usage: "add a user to a private network. Use this command if you want to be able to connect to a network from your own computer",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "network",
							Usage: "ID of the network",
						},
						cli.StringFlag{
							Name:  "user",
							Usage: "user ID",
						},
					},
					Action: addLocal,
				},
			},
		},
		{
			Name:  "container",
			Usage: "Provision containers",
			Subcommands: []cli.Command{
				{

					Name:      "create",
					Usage:     "create a container",
					ArgsUsage: "node ID where to deploy the container",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "flist",
							Usage: "URL to the flist",
						},
						cli.StringFlag{
							Name:  "entrypoint",
							Usage: "optional entrypoint. If specified it overwrites the entrypoint from the flist",
						},
						cli.BoolFlag{
							Name:  "corex",
							Usage: "enable coreX",
						},
						cli.StringFlag{
							Name:  "network",
							Usage: "network ID the container needs to be part of",
						},
						cli.StringSliceFlag{
							Name:  "mounts",
							Usage: "list of volume to mount into the container",
						},
						cli.StringSliceFlag{
							Name:  "envs",
							Usage: "environment variable to set into the container",
						},
					},
					Action: createContainer,
				},
			},
		},
		{
			Name:  "storage",
			Usage: "Provision volumes and 0-db",
			Subcommands: []cli.Command{
				{

					Name:      "create",
					Usage:     "create a storage volume",
					ArgsUsage: "node ID where to create the volume",
					Flags: []cli.Flag{
						cli.Uint64Flag{
							Name:  "size",
							Usage: "Size of the volume in GiB",
						},
						cli.StringFlag{
							Name:  "type",
							Usage: "Type of disk to use, HHD or SSD",
						},
					},
					Action: createVolume,
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}
