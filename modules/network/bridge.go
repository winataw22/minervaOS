package network

import (
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

func CreateBridge(name string) (*netlink.Bridge, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	bridge := &netlink.Bridge{
		LinkAttrs: attrs,
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return bridge, err
	}

	var err error
	defer func() {
		if err != nil {
			if err := netlink.LinkDel(bridge); err != nil {
				log.Error().Err(err).Msgf("failed to delete bridge %s", bridge.Name)
			}
		}
	}()
	err = netlink.LinkSetUp(bridge)
	return bridge, err
}

func ListBridges() ([]*netlink.Bridge, error) {
	links, err := interfaces()
	if err != nil {
		return nil, err
	}
	return filterBridge(links), nil
}

func BridgeAttachNic(device *netlink.Device, bridge *netlink.Bridge) error {
	return netlink.LinkSetMaster(device, bridge)
}

func BridgeDetachNic(bridge *netlink.Bridge) error {
	return netlink.LinkSetNoMaster(bridge)
}
