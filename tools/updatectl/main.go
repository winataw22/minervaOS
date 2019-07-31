package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules/upgrade"

	"github.com/blang/semver"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Usage = "upgradectl help to generate proper upgraded files for upgraded"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dir,d",
			Usage: "set working directory",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "release",
			Aliases:   []string{"r"},
			ArgsUsage: "version",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "flist, f",
					Usage: "url of the upgrade flist",
				},
				cli.StringFlag{
					Name:  "tx, t",
					Usage: "transaction ID",
				},
				cli.StringFlag{
					Name:  "storage, s",
					Usage: "URL to the 0-db storing the data of the flist",
					Value: "zdb://hub.grid.tf:9900",
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
		tx      = c.String("tx")
		version = c.Args().First()
		storage = c.String("storage")
		dir     = c.GlobalString("dir")
	)

	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			return err
		}
	}

	if flist == "" {
		return fmt.Errorf("flist must be specified")
	}

	if version == "" {
		return fmt.Errorf("version must be specified")
	}

	v, err := semver.Parse(version)
	if err != nil {
		return err
	}

	// TODO: validate tx

	u := upgrade.Upgrade{
		Flist: flist,
		// Signature: TODO
		TransactionID: tx,
		Storage:       storage,
	}

	if err := appendVersion(v); err != nil {
		return err
	}

	return writeUpgrade(v, u)
}

func writeUpgrade(v semver.Version, u upgrade.Upgrade) error {
	b, err := json.Marshal(u)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(v.String(), b, 0660); err != nil {
		return err
	}

	b, err = json.Marshal(v)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("latest", b, 0660)
}

func appendVersion(v semver.Version) error {
	versions := []semver.Version{}
	b, err := ioutil.ReadFile("versions")
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// versions list does not exist yet, let's created it
		if os.IsNotExist(err) {
			versions = append(versions, v)
			b, err := json.Marshal(versions)
			if err != nil {
				return err
			}
			return ioutil.WriteFile("versions", b, 0660)
		}
	}

	if err := json.Unmarshal(b, &versions); err != nil {
		return err
	}

	versions = append(versions, v)
	b, err = json.Marshal(versions)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("versions", b, 0660)
}
