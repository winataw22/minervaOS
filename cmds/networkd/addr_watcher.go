package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/vishvananda/netlink"
)

type WatchedLinks struct {
	linkNames map[string]struct{}
	dir       client.Directory
	nodeID    pkg.Identifier
}

func NewWatchedLinks(linkNames []string, nodeID pkg.Identifier, dir client.Directory) WatchedLinks {
	names := make(map[string]struct{}, len(linkNames))

	for _, n := range linkNames {
		names[n] = struct{}{}
	}

	return WatchedLinks{
		linkNames: names,
		dir:       dir,
		nodeID:    nodeID,
	}
}

func (w WatchedLinks) callBack(update netlink.AddrUpdate) error {
	link, err := netlink.LinkByIndex(update.LinkIndex)
	if err != nil {
		return nil
	}

	// skip link that are not watched
	if _, ok := w.linkNames[link.Attrs().Name]; !ok {
		return nil
	}

	log.Info().Msg("send network interfaces update to BCDB")

	ifaces, err := getLocalInterfaces()
	if err != nil {
		return err
	}

	return publishIfaces(ifaces, w.nodeID, w.dir)
}

func (w WatchedLinks) Forever(ctx context.Context) error {
	ch := make(chan netlink.AddrUpdate)
	done := make(chan struct{})
	defer close(done)

	log.Info().Msg("start netlink addr subscription")
	if err := netlink.AddrSubscribe(ch, done); err != nil {
		return err
	}

	nextAllowed := time.Now()

	for {
		select {

		case <-ctx.Done():
			return nil

		case update, ok := <-ch:
			if !ok {
				log.Error().Msg("netlink address subscription exited")
				return fmt.Errorf("netlink closed the subscription channel")
			}

			now := time.Now()
			if now.After(nextAllowed) {
				log.Debug().Msgf("addr update received %+v", update)

				if err := w.callBack(update); err != nil {
					log.Error().Err(err).Msg("addr watcher: error during callback")
				}
				nextAllowed = now.Add(time.Minute * 10)
			}
		}
	}
}

func getLocalInterfaces() ([]types.IfaceInfo, error) {
	var output []types.IfaceInfo

	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return nil, err
	}

	for _, link := range ifaceutil.LinkFilter(links, []string{"device", "bridge"}) {
		// a NIC of which the MII has no handshake detected, doesn't matter if it's up or down, so we bring them up,
		// in case there is some IPv6 RA on that link.

		if err := netlink.LinkSetUp(link); err != nil {
			log.Info().Str("interface", link.Attrs().Name).Msg("failed to bring interface up")
			continue
		}

		if !ifaceutil.IsVirtEth(link.Attrs().Name) && !ifaceutil.IsPluggedTimeout(link.Attrs().Name, time.Second*5) {
			log.Info().Str("interface", link.Attrs().Name).Msg("interface is not plugged in, skipping")
			continue
		}

		_, gw, err := ifaceutil.HasDefaultGW(link, netlink.FAMILY_ALL)
		if err != nil {
			return nil, err
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return nil, err
		}

		info := types.IfaceInfo{
			Name:       link.Attrs().Name,
			Addrs:      make([]types.IPNet, len(addrs)),
			MacAddress: schema.MacAddress{link.Attrs().HardwareAddr},
		}
		for i, addr := range addrs {
			info.Addrs[i] = types.NewIPNet(addr.IPNet)
		}

		if gw != nil {
			info.Gateway = append(info.Gateway, gw)
		}

		output = append(output, info)
	}

	return output, err
}

func publishIfaces(ifaces []types.IfaceInfo, id pkg.Identifier, db client.Directory) error {
	f := func() error {
		log.Info().Msg("try to publish interfaces to TNoDB")
		var input []directory.Iface
		for _, inf := range ifaces {
			input = append(input, inf.ToSchema())
		}
		return db.NodeSetInterfaces(id.Identity(), input)
	}
	errHandler := func(err error, _ time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to publish the node interaces")
		}
	}

	if err := backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler); err != nil {
		return err
	}

	return nil
}
