package network

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

// NodeIDv1 returns the node ID as it was calculated in 0-OS v1
func NodeIDv1() (string, error) {
	zos, err := netlink.LinkByName(types.DefaultBridge)
	if err != nil {
		return "", err
	}

	links, err := netlink.LinkList()
	if err != nil {
		return "", err
	}

	// find the physical interface attached to the default bridge
	for _, l := range links {
		if l.Attrs().MasterIndex == zos.Attrs().Index {
			return convertMac(l.Attrs().HardwareAddr.String()), nil
		}
	}
	return "", fmt.Errorf("not physical interface attached to default bridge found")
}

func convertMac(mac string) string {
	return strings.Replace(mac, ":", "", -1)
}
