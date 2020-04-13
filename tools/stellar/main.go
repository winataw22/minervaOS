package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Usage = "Create and sign Stellar multisig transactions"
	app.Version = "0.0.1"
	app.EnableBashCompletion = true

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
			Name:  "create",
			Usage: "Create multisig transaction",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "seed",
					Usage:    "Stellar secret key",
					Required: true,
				},
				cli.StringFlag{
					Name:     "network",
					Usage:    "Stellar network type",
					Required: true,
				},
				cli.StringFlag{
					Name:     "asset",
					Usage:    "Stellar asset",
					Required: true,
				},
				cli.StringFlag{
					Name:     "destination",
					Usage:    "Destination address",
					Required: true,
				},
				cli.StringFlag{
					Name:     "from",
					Usage:    "From escrow account address",
					Required: true,
				},
				cli.StringFlag{
					Name:     "amount",
					Usage:    "Amount to transfer",
					Required: true,
				},
			},
			Action: create,
		},
		{
			Name:  "sign",
			Usage: "Sign and submit transaction",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "seed",
					Usage:    "Stellar secret key",
					Required: true,
				},
				cli.StringFlag{
					Name:     "network",
					Usage:    "Stellar network type",
					Required: true,
				},
				cli.StringFlag{
					Name:     "transaction",
					Usage:    "Transaction to sign",
					Required: true,
				},
			},
			Action: signAndSubmit,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}
