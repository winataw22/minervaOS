package main

import (
	"fmt"
	"net/url"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis"

	"github.com/threefoldtech/zos/pkg/provision"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"os"

	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/tnodb"
	"github.com/urfave/cli"
)

var (
	client clientIface
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
			Name:   "bcdb, u",
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

		var err error
		client, err = getClient(c.String("bcdb"))
		if err != nil {
			return err
		}

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
			Subcommands: []cli.Command{
				{
					Name:  "network",
					Usage: "Manage private networks",
					Subcommands: []cli.Command{
						{
							Name:  "create",
							Usage: "create a new user network",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:  "name",
									Usage: "name of your network",
								},
								cli.StringFlag{
									Name:  "cidr",
									Usage: "private ip range to use in the network",
								},
							},
							Action: cmdCreateNetwork,
						},
						{
							Name:  "add-node",
							Usage: "add a node to a existing network",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:  "node",
									Usage: "node ID of the node to add to the network",
								},
								cli.StringFlag{
									Name:  "subnet",
									Usage: "subnet to use on this node. The subnet needs to be included in the IP range of the network",
								},
								cli.UintFlag{
									Name:  "port",
									Usage: "Wireguar port to use. if not specified, tfuser will automatically check BCDB for free fort to use",
								},
							},
							Action: cmdsAddNode,
						},
						{
							Name:  "remove-node",
							Usage: "prints the wg-quick configuration file for a certain user in the network",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:  "node",
									Usage: "node ID to remove from the network",
								},
							},
							Action: cmdsRemoveNode,
						},
					},
				},
				{
					Name:  "container",
					Usage: "Generate container provisioning schema",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "flist",
							Usage: "URL to the flist",
						},
						cli.StringFlag{
							Name:  "storage",
							Usage: "URL to the flist storage backend",
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
						cli.StringFlag{
							Name:  "ip",
							Usage: "ip address to assign to the container",
						},
					},
					Action: generateContainer,
				},
				{
					Name:  "storage",
					Usage: "Generate volumes and 0-db namespace provisioning schema",
					Subcommands: []cli.Command{
						{
							Name:    "volume",
							Aliases: []string{"vol"},
							Flags: []cli.Flag{
								cli.Uint64Flag{
									Name:  "size, s",
									Usage: "Size of the volume in GiB",
									Value: 1,
								},
								cli.StringFlag{
									Name:  "type, t",
									Usage: "Type of disk to use, HHD or SSD",
								},
							},
							Action: generateVolume,
						},
						{
							Name:  "zdb",
							Usage: "reserve a 0-db namespace",
							Flags: []cli.Flag{
								cli.Uint64Flag{
									Name:  "size, s",
									Usage: "Size of the volume in GiB",
									Value: 1,
								},
								cli.StringFlag{
									Name:  "type, t",
									Usage: "Type of disk to use, HHD or SSD",
								},
								cli.StringFlag{
									Name:  "mode, m",
									Usage: "0-DB mode (user, seq)",
								},
								cli.StringFlag{
									Name:  "password, p",
									Usage: "optional password",
								},
								cli.BoolFlag{
									Name:  "public",
									Usage: "TODO",
								},
							},
							Action: generateZDB,
						},
					},
				},
				{
					Name:  "debug",
					Usage: "Enable debug mode on a node. In this mode the forward its logs to the specified redis endpoint",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name: "endpoint",
						},
						cli.StringFlag{
							Name:  "channel",
							Usage: "name of the redis pubsub channel to use, if empty the node will push to {nodeID}-logs",
						},
					},
					Action: generateDebug,
				},
			},
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

type reserver interface {
	Reserve(r *provision.Reservation, nodeID pkg.Identifier) (string, error)
}
type clientIface interface {
	network.TNoDB
	reserver
}

func getClient(addr string) (clientIface, error) {
	type client struct {
		network.TNoDB
		reserver
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return client{
			tnodb.NewHTTPTNoDB(addr),
			provision.NewHTTPStore(addr),
		}, nil
	case "tcp":
		c, err := gedis.New(addr, "default", "")
		return client{
			c,
			c,
		}, err
	default:
		return nil, fmt.Errorf("unsupported address scheme for BCDB: %s", u.Scheme)
	}
}
