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
	db    network.TNoDBUtils
	store *provision.HTTPStore
)

func main() {

	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Usage = "Let you provision capacity on the ThreefoldGrid 2.0"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug logging",
		},

		cli.StringFlag{
			Name:   "tnodb, u",
			Usage:  "URL of the TNODB",
			Value:  "https://tnodb.dev.grid.tf",
			EnvVar: "TNODB_URL",
		},

		cli.StringFlag{
			Name:   "provision, p",
			Usage:  "URL of the provision store",
			Value:  "https://tnodb.dev.grid.tf",
			EnvVar: "PROVISION_URL",
		},
	}
	app.Before = func(c *cli.Context) error {
		debug := c.Bool("debug")
		if !debug {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		db = tnodb.NewHTTPTNoDB(c.String("tnodb"))
		store = provision.NewHTTPStore(c.String("provision"))

		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "id",
			Usage: "generate a user identity",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "output,o",
					Usage: "output path of the identity seed",
					Value: "user.seed",
				},
			},
			Action: cmdsGenerateID,
		},
		{
			Name:    "generate",
			Aliases: []string{"gen"},
			Usage:   "Group of command to generate provisioning schemas",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "schema,s",
					Usage: `location of the generated schema. 
					For the network sub-commands add-node and add-user this flag is
					also used to read the network schema before modifying it`,
				},
			},
			// Subcommands: []cli.Command{
			// 	{
			// 		Name:  "network",
			// 		Usage: "Manage private networks",
			// 		Subcommands: []cli.Command{
			// 			{
			// 				Name:  "create",
			// 				Usage: "create a new user network",
			// 				Flags: []cli.Flag{
			// 					cli.StringFlag{
			// 						Name:  "node",
			// 						Usage: "node ID of the exit node to use for this network",
			// 					},
			// 				},
			// 				Action: cmdCreateNetwork,
			// 			},
			// 			{
			// 				Name:  "add-node",
			// 				Usage: "add a node to a existing network",
			// 				Flags: []cli.Flag{
			// 					cli.StringSliceFlag{
			// 						Name:  "node",
			// 						Usage: "node ID of the node where to install this network, you can specify multiple time this flag",
			// 					},
			// 				},
			// 				Action: cmdsAddNode,
			// 			},
			// 			{
			// 				Name:  "remove-node",
			// 				Usage: "prints the wg-quick configuration file for a certain user in the network",
			// 				Flags: []cli.Flag{
			// 					cli.StringFlag{
			// 						Name:  "node",
			// 						Usage: "node ID to remove from the network",
			// 					},
			// 				},
			// 				Action: cmdsRemoveNode,
			// 			},
			// 			{
			// 				Name:  "add-user",
			// 				Usage: "prints the wg-quick configuration file for a certain user in the network",
			// 				Flags: []cli.Flag{
			// 					cli.StringFlag{
			// 						Name:  "user",
			// 						Usage: "user ID",
			// 					},
			// 				},
			// 				Action: cmdsAddUser,
			// 			},
			// 			{
			// 				Name:  "wg",
			// 				Usage: "add a user to a private network. Use this command if you want to be able to connect to a network from your own computer",
			// 				Flags: []cli.Flag{
			// 					cli.StringFlag{
			// 						Name:  "user",
			// 						Usage: "user ID, if not specified, a user ID will be generated automatically",
			// 					},
			// 					cli.StringFlag{
			// 						Name:  "key",
			// 						Usage: "private key. this is usually given by the 'user' command",
			// 					},
			// 				},
			// 				Action: cmdsWGQuick,
			// 			},
			// 		},
			// 	},
			// 	{
			// 		Name:  "container",
			// 		Usage: "Generate container provisioning schema",
			// 		Flags: []cli.Flag{
			// 			cli.StringFlag{
			// 				Name:  "flist",
			// 				Usage: "URL to the flist",
			// 			},
			// 			cli.StringFlag{
			// 				Name:  "storage",
			// 				Usage: "URL to the flist storage backend",
			// 			},
			// 			cli.StringFlag{
			// 				Name:  "entrypoint",
			// 				Usage: "optional entrypoint. If specified it overwrites the entrypoint from the flist",
			// 			},
			// 			cli.BoolFlag{
			// 				Name:  "corex",
			// 				Usage: "enable coreX",
			// 			},
			// 			cli.StringFlag{
			// 				Name:  "network",
			// 				Usage: "network ID the container needs to be part of",
			// 			},
			// 			cli.StringSliceFlag{
			// 				Name:  "mounts",
			// 				Usage: "list of volume to mount into the container",
			// 			},
			// 			cli.StringSliceFlag{
			// 				Name:  "envs",
			// 				Usage: "environment variable to set into the container",
			// 			},
			// 		},
			// 		Action: generateContainer,
			// 	},
			// 	{
			// 		Name:  "storage",
			// 		Usage: "Generate volumes and 0-db namespace provisioning schema",
			// 		Subcommands: []cli.Command{
			// 			{
			// 				Name:    "volume",
			// 				Aliases: []string{"vol"},
			// 				Flags: []cli.Flag{
			// 					cli.Uint64Flag{
			// 						Name:  "size, s",
			// 						Usage: "Size of the volume in GiB",
			// 						Value: 1,
			// 					},
			// 					cli.StringFlag{
			// 						Name:  "type, t",
			// 						Usage: "Type of disk to use, HHD or SSD",
			// 					},
			// 				},
			// 				Action: generateVolume,
			// 			},
			// 			{
			// 				Name:  "zdb",
			// 				Usage: "reserve a 0-db namespace",
			// 				Flags: []cli.Flag{
			// 					cli.Uint64Flag{
			// 						Name:  "size, s",
			// 						Usage: "Size of the volume in GiB",
			// 						Value: 1,
			// 					},
			// 					cli.StringFlag{
			// 						Name:  "type, t",
			// 						Usage: "Type of disk to use, HHD or SSD",
			// 					},
			// 					cli.StringFlag{
			// 						Name:  "mode, m",
			// 						Usage: "0-DB mode (user, seq)",
			// 					},
			// 					cli.StringFlag{
			// 						Name:  "password, p",
			// 						Usage: "optional password",
			// 					},
			// 					cli.BoolFlag{
			// 						Name:  "public",
			// 						Usage: "TODO",
			// 					},
			// 				},
			// 				Action: generateZDB,
			// 			},
			// 		},
			// 	},
			// 	{
			// 		Name:  "debug",
			// 		Usage: "Enable debug mode on a node. In this mode the forward its logs to the specified redis endpoint",
			// 		Flags: []cli.Flag{
			// 			cli.StringFlag{
			// 				Name: "endpoint",
			// 			},
			// 			cli.StringFlag{
			// 				Name:  "channel",
			// 				Usage: "name of the redis pubsub channel to use, if empty the node will push to {nodeID}-logs",
			// 			},
			// 		},
			// 		Action: generateDebug,
			// 	},
			// },
		},
		{
			Name:  "provision",
			Usage: "Provision a workload",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "schema",
					Usage: "path to the provisioning schema, use - to read from stdin",
					Value: "provision.json",
				},
				cli.StringSliceFlag{
					Name:  "node",
					Usage: "Node ID where to deploy the workload",
				},
				cli.StringFlag{
					Name:  "duration",
					Usage: "duration of the reservation. By default is number of days. But also support notation with duration suffix like m for minute or h for hours",
				},
				cli.StringFlag{
					Name:   "seed",
					Usage:  "path to the file container the seed of the user private key",
					EnvVar: "SEED_PATH",
				},
			},
			Action: cmdsProvision,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}
