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
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()

	idStore := identity.NewHTTPIDStore("http://localhost:8080")
	db := tnodb.NewHTTPHTTPTNoDB("http://localhost:8080")
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable debug logging",
		},
	}
	app.Before = func(c *cli.Context) error {
		debug := c.Bool("debug")
		if !debug {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
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
				seedPath := c.String("seed")

				keyPair, err := generateKeyPair(seedPath)
				if err != nil {
					return err
				}
				name := c.Args().First()
				if name == "" {

				}
				farm := identity.NewFarm(name, keyPair)
				if err := idStore.RegisterFarm(farm, name); err != nil {
					return err
				}
				fmt.Println("farm registered successfully")
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
			ArgsUsage: "farm_id",
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
				cli.StringFlag{
					Name:  "node",
					Usage: "node id to use as exit node",
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
				node := c.String("node")

				if err := db.ConfigureExitNode(strID(node), ipnet, gw, iface); err != nil {
					return err
				}
				fmt.Printf("exit node configured\n")
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
	keypair, err := identity.LoadSeed(seedPath)
	if err != nil {
		return nil, err
	}
	farm := identity.NewFarm("", keypair)
	return farm, nil
}

func generateKeyPair(seedPath string) (*identity.KeyPair, error) {

	if seedPath != "" {
		log.Debug().Msgf("loading seed from %s", seedPath)
		keypair, err := identity.LoadSeed(seedPath)
		if err != nil {
			return nil, err
		}
		return keypair, nil
	}

	log.Debug().Msg("generating new key pair")
	keypair, err := identity.GenerateKeyPair()
	if err != nil {
		log.Error().Err(err).Msg("fail to generate key pair for farm identity")
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(cwd, "farm.seed")

	if err := identity.SerializeSeed(keypair, path); err != nil {
		log.Error().Err(err).Msg("fail to save identity seed on disk")
		return nil, err
	}

	return keypair, nil
}
