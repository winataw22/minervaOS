package main

import (
	"fmt"
	"net/url"

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
							Usage: "Add a node to a existing network",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:  "node",
									Usage: "Node ID of the node to add to the network",
								},
								cli.StringFlag{
									Name:  "subnet",
									Usage: "Subnet to use on this node. The subnet needs to be included in the IP range of the network",
								},
								cli.UintFlag{
									Name:  "port",
									Usage: "Wireguard port to use. if not specified, tfuser will automatically check BCDB for free fort to use",
								},
								cli.BoolFlag{
									Name:  "force-hidden",
									Usage: "Forcibly mark the node as hidden, even if it is registered with public endpoints",
								},
							},
							Action: cmdsAddNode,
						},
						{
							Name:  "remove-node",
							Usage: "Removes a Network Resource from the network",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:  "node",
									Usage: "Node ID to remove from the network",
								},
							},
							Action: cmdsRemoveNode,
						},
						{
							Name:   "graph",
							Usage:  "create a dot graph of the network",
							Action: cmdGraphNetwork,
						},
						{
							Name:  "add-access",
							Usage: "Add external access to the network",
							Flags: []cli.Flag{
								cli.StringFlag{
									Name:  "node",
									Usage: "Node ID of the node which will act as an access point",
								},
								cli.StringFlag{
									Name:  "subnet",
									Usage: "Local subnet which will have access to the network",
								},
								cli.BoolFlag{
									Name:  "ip4",
									Usage: "Use an IPv4 connection instead of IPv6 for the wireguard endpoint to the access node",
								},
								cli.StringFlag{
									Name:  "wgpubkey",
									Usage: "The wireguard public key of the external node, encoded in base64. If this flag is provided, you will need to manually set your private key in the generated wireguard config. If not provided, a keypair will be generated.",
								},
							},
							Action: cmdsAddAccess,
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
						cli.UintFlag{
							Name:  "cpu",
							Usage: "limit the amount of CPU allocated to the container",
						},
						cli.Uint64Flag{
							Name:  "memory",
							Usage: "limit the amount of memory a container can allocate",
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
								cli.StringFlag{
									Name:  "node, n",
									Usage: "node ID. Required if password is set to encrypt the password",
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
					Name:  "kubernetes",
					Usage: "Provision a vm running a kubernetes server or agent on a node",
					Flags: []cli.Flag{
						cli.UintFlag{
							Name:  "size",
							Usage: "Size of the VM, only 1 (small) and 2 (medium) are supported",
							Value: 1,
						},
						cli.StringFlag{
							Name:  "network-id",
							Usage: "ID of the network resource in which the vm will be created",
						},
						cli.StringFlag{
							Name:  "ip",
							Usage: "Ip address of the vm in the network resource",
						},
						cli.StringFlag{
							Name:  "secret, s",
							Usage: "Cluster token to set for kubernetes, this is encrypted by the nodes public key",
						},
						cli.StringFlag{
							Name:  "node, n",
							Usage: "node ID. Required if password is set to encrypt the password",
						},
						cli.StringSliceFlag{
							Name:  "master-ips",
							Usage: "IP address(es) of the master node(s) (multiple in HA node). If this flag is not set, this instance will be set up as a master node",
						},
						cli.StringSliceFlag{
							Name:  "ssh-keys",
							Usage: "Ssh keys to authorize for the vm. Can be either a full ssh key, or a \"github:username\" form which will pull the ssh keys from github",
						},
					},
					Action: generateKubernetes,
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
		{
			Name:  "delete",
			Usage: "Mark a workload as to be deleted",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "workload id",
				},
			},
			Action: cmdsDeleteReservation,
		},
		{
			Name:  "live",
			Usage: "show you all the reservations that are still alive",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "seed",
					Usage:  "path to the file container the seed of the user private key",
					EnvVar: "SEED_PATH",
				},
				cli.IntFlag{
					Name:  "start",
					Usage: "start scrapping at that reservation ID",
				},
				cli.IntFlag{
					Name:  "end",
					Usage: "end scrapping at that reservation ID",
					Value: 500,
				},
				cli.BoolFlag{
					Name:  "expired",
					Usage: "include expired reservations",
				},
			},
			Action: cmdsLive,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

type reserveDeleter interface {
	Reserve(r *provision.Reservation) (string, error)
	Delete(id string) error
}
type clientIface interface {
	network.TNoDB
	reserveDeleter
}

func getClient(addr string) (clientIface, error) {
	type client struct {
		network.TNoDB
		reserveDeleter
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
		c, err := gedis.New(addr, "")
		return client{
			c,
			c,
		}, err
	default:
		return nil, fmt.Errorf("unsupported address scheme for BCDB: %s", u.Scheme)
	}
}
