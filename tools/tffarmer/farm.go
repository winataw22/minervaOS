package main

import (
	"fmt"

	"github.com/urfave/cli"
)

func registerFarm(c *cli.Context) error {

	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("farm name needs to be specified")
	}

	tid := c.Uint64("tid")

	farmID, err := idStore.RegisterFarm(tid, name, "", []string{})
	if err != nil {
		return err
	}

	fmt.Println("Farm registered successfully")
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Farm ID: %d\n", farmID)
	return nil
}
