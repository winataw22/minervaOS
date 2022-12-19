package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {

	app := cli.App{
		Name:  "admin",
		Usage: "threefold farmer admin tool",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "mnemonics",
				Usage: "farmer mnemonics",
				EnvVars: []string{
					"MNEMONICS",
				},
				Required: true,
			},
			&cli.StringFlag{
				Name:  "network",
				Usage: "network to use one of (production, testing, qa, development)",
				Value: "production",
				EnvVars: []string{
					"NETWORK",
				},
				Required: true,
			},
			&cli.StringFlag{
				Name:  "key-type",
				Usage: "choose your key type, one of (sr25519, ed25519)",
				Value: "sr25519",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "network",
				Usage: "manage node networks",
				Subcommands: []*cli.Command{
					{
						Name:  "public-config",
						Usage: "Show node public config if exists",
						Flags: []cli.Flag{
							&cli.Uint64Flag{
								Name:     "node",
								Usage:    "node id",
								Required: true,
							},
						},
						Action: networkShowPublicConfig,
					},
					{
						Name:  "public-exit",
						Usage: "Show node public config if exists",
						Flags: []cli.Flag{
							&cli.Uint64Flag{
								Name:     "node",
								Usage:    "node id",
								Required: true,
							},
						},
						Subcommands: []*cli.Command{
							{
								Name:   "get",
								Usage:  "get public exit setup (dual or single setup)",
								Action: networkShowPublicExit,
							},
							{
								Name:   "list",
								Usage:  "list possible exit nics on the node",
								Action: networkListPublicExit,
							},
							{
								Name:      "set",
								Usage:     "set exit to one of possible nics. if `zos` will sent node to single nic setup",
								Action:    networkSetPublicExit,
								ArgsUsage: "[nic]",
							},
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

}
