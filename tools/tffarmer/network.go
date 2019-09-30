package main

import (
	"fmt"
	"net"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/types"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func giveAlloc(c *cli.Context) error {

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
}

func configPublic(c *cli.Context) error {
	var (
		iface = c.String("iface")
	)

	var nv4 *net.IPNet
	var nv6 *net.IPNet
	var gw4 net.IP
	var gw6 net.IP

	for _, ip := range c.StringSlice("ip") {
		i, ipnet, err := net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("invalid cidr(%s): %s", ip, err)
		}

		ipnet.IP = i
		if ipnet.IP.To4() == nil {
			//ipv6
			if nv6 != nil {
				return fmt.Errorf("only one ipv6 range is supported")
			}
			nv6 = ipnet
		} else {
			//ipv4
			if nv4 != nil {
				return fmt.Errorf("only one ipv4 range is supported")
			}
			nv4 = ipnet
		}
	}

	for _, s := range c.StringSlice("gw") {
		gw := net.ParseIP(s)
		if gw == nil {
			return fmt.Errorf("invalid gw '%s'", s)
		}
		if gw.To4() == nil {
			//ipv6
			if gw6 != nil {
				return fmt.Errorf("only one gw ipv6 is supported")
			}
			gw6 = gw
		} else {
			//ipv4
			if gw4 != nil {
				return fmt.Errorf("only one gw ipv4 is supported")
			}
			gw4 = gw
		}
	}

	node := c.Args().First()

	if err := db.SetPublicIface(modules.StrIdentifier(node), &types.PubIface{
		Master: iface,
		IPv4:   nv4,
		IPv6:   nv6,
		GW4:    gw4,
		GW6:    gw6,
	}); err != nil {
		return err
	}
	fmt.Printf("public interface configured on node %s\n", node)
	return nil
}

func selectExit(c *cli.Context) error {
	node := c.Args().First()

	if err := db.SelectExitNode(modules.StrIdentifier(node)); err != nil {
		return err
	}
	fmt.Printf("Node %s marked as exit node\n", node)
	return nil
}
