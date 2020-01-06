package ndmz

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/threefoldtech/zos/pkg/network/nr"

	"github.com/threefoldtech/zos/pkg/network/dhcp"

	"github.com/threefoldtech/zos/pkg/network/macvlan"

	"github.com/threefoldtech/zos/pkg/network/bridge"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
)

const (
	//BridgeNDMZ is the name of the ipv4 routing bridge in the ndmz namespace
	BridgeNDMZ = "br-ndmz"
	netNSNDMZ  = "ndmz"

	ndmzNsMACDerivationSuffix = "-ndmz"

	publicIfaceName = "public"
)

//Create create the NDMZ network namespace and configure its default routes and addresses
func Create(nodeID pkg.Identifier) error {

	os.RemoveAll("/var/cache/modules/networkd/lease/dmz/")

	netNS, err := namespace.GetByName(netNSNDMZ)
	if err != nil {
		netNS, err = namespace.Create(netNSNDMZ)
		if err != nil {
			return err
		}
	}

	defer netNS.Close()

	if err := netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
			return errors.Wrapf(err, "failed to enable ipv6 forwarding in gateway namespace")
		}
		return ifaceutil.SetLoUp()
	}); err != nil {
		return err
	}

	if err := createRoutingBridge(netNS); err != nil {
		return err
	}

	if err := createPublicIface(netNS); err != nil {
		return err
	}

	// set mac address to something static to make sure we receive the same IP from a DHCP server
	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + ndmzNsMACDerivationSuffix))
	log.Debug().
		Str("mac", mac.String()).
		Msg("set mac on public iface")

	if err = ifaceutil.SetMAC(types.PublicIface, mac, netNS); err != nil {
		return err
	}

	err = netNS.Do(func(_ ns.NetNS) error {
		// run DHCP to interface public in ndmz
		received, err := dhcp.Probe(types.PublicIface)
		if err != nil {
			return err
		}
		if !received {
			return errors.Errorf("public interface in ndmz did not received an IP. make sure dhcp is working")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return applyFirewall()
}

// Delete deletes the NDMZ network namespace
func Delete() error {
	netNS, err := namespace.GetByName(netNSNDMZ)
	if err == nil {
		if err := namespace.Delete(netNS); err != nil {
			return errors.Wrap(err, "failed to delete ndmz network namespace")
		}
	}

	return nil
}

func createPublicIface(netNS ns.NetNS) error {
	if !ifaceutil.Exists(types.PublicIface, netNS) {

		var (
			master netlink.Link
			err    error
		)

		if namespace.Exists(types.PublicNamespace) {
			master, err = getPublicIface()
		} else {
			master, err = netlink.LinkByName("zos")
		}
		if err != nil {
			return err
		}

		_, err = macvlan.Create(types.PublicIface, master.Attrs().Name, netNS)
		return err
	}

	return nil
}

func createRoutingBridge(netNS ns.NetNS) error {
	if !bridge.Exists(BridgeNDMZ) {
		if _, err := bridge.New(BridgeNDMZ); err != nil {
			return err
		}
	}

	const tonrsIface = "tonrs"

	if !ifaceutil.Exists(tonrsIface, netNS) {
		if _, err := macvlan.Create(tonrsIface, BridgeNDMZ, netNS); err != nil {
			return err
		}
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", BridgeNDMZ), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", BridgeNDMZ)
	}

	return netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", tonrsIface), "1"); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on macvlan %s", tonrsIface)
		}

		link, err := netlink.LinkByName(tonrsIface)
		if err != nil {
			return err
		}

		err = netlink.AddrAdd(link, &netlink.Addr{
			IPNet: &net.IPNet{
				IP:   net.ParseIP("100.127.0.1"),
				Mask: net.CIDRMask(16, 32),
			},
		})
		if err != nil && !os.IsExist(err) {
			return err
		}

		return netlink.LinkSetUp(link)
	})
}

func applyFirewall() error {
	buf := bytes.Buffer{}

	if err := fwTmpl.Execute(&buf, nil); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, netNSNDMZ); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

// AttachNR links a network resource to the NDMZ
func AttachNR(networkID string, nr *nr.NetResource) error {
	nrNSName, err := nr.Namespace()
	if err != nil {
		return err
	}

	nrNS, err := namespace.GetByName(nrNSName)
	if err != nil {
		return err
	}

	if !ifaceutil.Exists(publicIfaceName, nrNS) {
		if _, err = macvlan.Create(publicIfaceName, BridgeNDMZ, nrNS); err != nil {
			return err
		}
	}

	return nrNS.Do(func(_ ns.NetNS) error {
		addr, err := allocateIPv4(networkID)
		if err != nil {
			return errors.Wrap(err, "ip allocation for network resource")
		}

		pubIface, err := netlink.LinkByName(publicIfaceName)
		if err != nil {
			return err
		}

		if err := netlink.AddrAdd(pubIface, &netlink.Addr{IPNet: addr}); err != nil && !os.IsExist(err) {
			return err
		}

		if err = netlink.LinkSetUp(pubIface); err != nil {
			return err
		}

		err = netlink.RouteAdd(&netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw:        net.ParseIP("100.127.0.1"),
			LinkIndex: pubIface.Attrs().Index,
		})
		if err != nil && !os.IsExist(err) {
			return err
		}
		return nil
	})
}
