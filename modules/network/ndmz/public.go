package ndmz

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/vishvananda/netlink"
)

func getPublicIface() (string, error) {
	var ifaceName string

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
			return "", err
		}

		master, err := netlink.LinkByIndex(ifaceIndex)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get link by index %d", ifaceIndex)
		}
		ifaceName = master.Attrs().Name

	} else {
		// since we are a fully public node
		// get the name of the interface that has the default gateway
		links, err := netlink.LinkList()
		if err != nil {
			return "", errors.Wrap(err, "failed to list interfaces")
		}
		for _, link := range links {
			has, _, err := ifaceutil.HasDefaultGW(link)
			if err != nil {
				return "", errors.Wrapf(err, "failed to inspect default gateway of iface %s", link.Attrs().Name)
			}

			if has {
				ifaceName = link.Attrs().Name
				break
			}
		}
	}

	if ifaceName == "" {
		return "", fmt.Errorf("not interface with default gateway found")
	}

	log.Info().Str("iface", ifaceName).Msg("interface with default gateway found")
	return ifaceName, nil
}
