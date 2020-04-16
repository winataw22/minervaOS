package main

import (
	"fmt"

	"github.com/urfave/cli"
)

func markFree(c *cli.Context) error {
	nodes := c.StringSlice("nodes")
	free := c.Bool("free")

	for _, id := range nodes {
		fmt.Printf("node %s ", id)
		if err := db.NodeSetFreeToUse(id, free); err != nil {
			fmt.Printf(" error %v\n", err)
			return err
		}
		fmt.Printf("free to use: %v\n", free)
	}
	return nil
}
