package ndmz

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

func getPublicIface() (netlink.Link, error) {
	var iface netlink.Link

	pubNS, err := namespace.GetByName(types.PublicNamespace)

	if err == nil { // there is a public namespace on the node
		var ifaceIndex int

		defer pubNS.Close()
		// get the name of the public interface in the public namespace
		if err := pubNS.Do(func(_ ns.NetNS) error {
			// get the name of the interface connected to the public segment
			public, err := netlink.LinkByName(types.PublicIface)
			if err != nil {
				return errors.Wrap(err, "failed to get public link")
			}

			ifaceIndex = public.Attrs().ParentIndex
			return nil
		}); err != nil {
			return nil, err
		}

		master, err := netlink.LinkByIndex(ifaceIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get link by index %d", ifaceIndex)
		}
		iface = master
	} else {
		zos, err := netlink.LinkByName("zos")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get zos link")
		}
		// find the name of the interface attached to zos bridge
		links, err := netlink.LinkList()
		if err != nil {
			return nil, errors.Wrap(err, "failed to list interfaces")
		}
		for _, link := range links {
			if link.Attrs().MasterIndex == zos.Attrs().Index && link.Type() == "device" {
				iface = link
				break
			}
		}
	}

	if iface == nil {
		return nil, fmt.Errorf("no interface with default gateway found")
	}

	log.Info().Str("iface", iface.Attrs().Name).Msg("interface with default gateway found")
	return iface, nil
}
