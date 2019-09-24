package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/blang/semver"

	"github.com/urfave/cli"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := cli.NewApp()
	app.Usage = "upgradectl help to generate proper upgraded files for upgraded"
	app.Flags = []cli.Flag{}

	app.Commands = []cli.Command{
		{
			Name:        "release",
			Aliases:     []string{"r"},
			Usage:       "release an flist to given name and version",
			Description: "This command simply moves the given `flist` to `<release>:<version>.flist` and makes sure that `<release>:latest.flist` points to it.",
			ArgsUsage:   "<version>",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "release, r",
					Usage: "published release name (output)",
				},
				cli.StringFlag{
					Name:  "flist, f",
					Usage: "the flist name to release (input)",
				},
				cli.StringFlag{
					Name:  "jwt, t",
					Usage: "iyo token",
				},
			},
			Action: release,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func release(c *cli.Context) error {
	var (
		flist   = c.String("flist")
		release = c.String("release")
		version = c.Args().First()
		jwt     = c.String("jwt")
	)

	if flist == "" {
		return fmt.Errorf("flist must be specified")
	}

	if release == "" {
		return fmt.Errorf("release must be specified")
	}

	if jwt == "" {
		return fmt.Errorf("jwt must be specified")
	}

	if version == "" {
		return fmt.Errorf("version must be specified")
	}

	v, err := semver.Parse(version)
	if err != nil {
		return err
	}

	hub, err := NewHub(jwt)
	if err != nil {
		return err
	}

	releaseName := fmt.Sprintf("%s:latest.flist", release)
	flistDest := fmt.Sprintf("%s:%s.flist", release, v.String())
	if flist != flistDest {
		if err := hub.Rename(flist, flistDest); err != nil {
			return errors.Wrap(err, "failed to rename flist")
		}
	}

	return hub.Link(flistDest, releaseName)
}
