package main

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"os"

	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()

	var (
		idStore identity.IDStore
		db      network.TNoDB
	)

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug logging",
		},
		cli.StringFlag{
			Name:  "tnodb, u",
			Usage: "URL of the TNODB",
		},
	}
	app.Before = func(c *cli.Context) error {
		debug := c.Bool("debug")
		if !debug {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		url := c.String("tnodb")
		idStore = identity.NewHTTPIDStore(url)
		db = tnodb.NewHTTPHTTPTNoDB(url)

		return nil
	}
	app.Commands = []cli.Command{
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
			Action: func(c *cli.Context) error {

				name := c.Args().First()
				if name == "" {
					return fmt.Errorf("A farm name needs to be specified")
				}

				var farmID identity.Identifier
				var err error
				seedPath := c.String("seed")
				if seedPath != "" {
					farmID, err = loadFarmID(seedPath)
					if err != nil {
						return err
					}
				}
				if farmID == nil {
					farmID, err = generateKeyPair(seedPath)
					if err != nil {
						return err
					}
				}

				if err := idStore.RegisterFarm(farmID, name); err != nil {
					return err
				}
				fmt.Println("Farm registered successfully")
				fmt.Printf("Name: %s\n", name)
				fmt.Printf("Identity: %s\n", farmID.Identity())
				return nil
			},
		},
		{
			Name:      "give-alloc",
			Category:  "network",
			Usage:     "register an allocation to the TNoDB",
			ArgsUsage: "allocation prefix (ip/mask)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "seed",
					Usage: "path to the farmer seed. Specify this if you already have a seed generated for your farm",
				},
			},
			Action: func(c *cli.Context) error {

				farmID, err := loadFarmID(c.String("seed"))
				if err != nil {
					log.Error().Err(err).Msg("impossible to load farm id, user register command first")
					return err
				}

				alloc := c.Args().First()
				_, allocation, err := net.ParseCIDR(alloc)
				if err != nil {
					log.Error().Err(err).Msg("prefix format not valid, use ip/mask")
					return err
				}

				if err := db.RegisterAllocation(farmID, allocation); err != nil {
					log.Error().Err(err).Msg("failed to register prefix")
					return err
				}

				fmt.Println("prefix registered successfully")
				return nil
			},
		},
		{
			Name:      "get-alloc",
			Category:  "network",
			Usage:     "get an allocation for a tenant network",
			ArgsUsage: "farm_id",
			Action: func(c *cli.Context) error {

				farm := c.Args().First()
				alloc, err := db.RequestAllocation(strID(farm))
				if err != nil {
					log.Error().Err(err).Msg("failed to get an allocation")
					return err
				}

				fmt.Printf("allocation received: %s\n", alloc.String())
				return nil
			},
		},
		{
			Name:      "configure-public",
			Category:  "network",
			Usage:     "configure the public interface of a node",
			ArgsUsage: "node ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "ip",
					Usage: "ip address to set to the exit interface",
				},
				cli.StringFlag{
					Name:  "gw",
					Usage: "gw address to set to the exit interface",
				},
				cli.StringFlag{
					Name:  "iface",
					Usage: "name of the interface to use as public interface",
				},
			},
			Action: func(c *cli.Context) error {
				ip, ipnet, err := net.ParseCIDR(c.String("ip"))
				if err != nil {
					return err
				}
				ipnet.IP = ip
				gw := net.ParseIP(c.String("gw"))
				iface := c.String("iface")
				node := c.Args().First()

				if err := db.ConfigurePublicIface(strID(node), ipnet, gw, iface); err != nil {
					return err
				}
				fmt.Printf("public interface configured on node %s\n", node)
				return nil
			},
		},
		{
			Name:      "select-exit",
			Category:  "network",
			Usage:     "mark a node as being an exit",
			ArgsUsage: "node ID",
			Action: func(c *cli.Context) error {
				node := c.Args().First()

				if err := db.SelectExitNode(strID(node)); err != nil {
					return err
				}
				fmt.Printf("Node %s marked as exit node\n", node)
				return nil
			},
		},
		{
			Name:      "create-network",
			Category:  "network",
			Usage:     "create a new user network",
			ArgsUsage: "ID of the exit farm",
			Action: func(c *cli.Context) error {
				farmID := c.Args().First()
				network, err := db.CreateNetwork(farmID)
				if err != nil {
					log.Error().Err(err).Msg("failed to create network")
					return err
				}
				b, err := json.Marshal(network)
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

type strID string

func (f strID) Identity() string {
	return string(f)
}

func loadFarmID(seedPath string) (identity.Identifier, error) {

	if seedPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		seedPath = filepath.Join(cwd, "farm.seed")
	}

	log.Debug().Msgf("loading seed from %s", seedPath)
	farmID, err := identity.LoadSeed(seedPath)
	if err != nil {
		return nil, err
	}

	return farmID, nil
}

func generateKeyPair(seedPath string) (identity.Identifier, error) {

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

	if err := identity.SerializeSeed(keypair, seedPath); err != nil {
		log.Error().Err(err).Msg("fail to save identity seed on disk")
		return nil, err
	}

	return keypair, nil
}
