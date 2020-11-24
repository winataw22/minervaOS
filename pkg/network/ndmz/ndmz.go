package ndmz

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/threefoldtech/zos/pkg/network/yggdrasil"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
	"github.com/threefoldtech/zos/pkg/network/nr"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const (
	//BridgeNDMZ is the name of the ipv4 routing bridge in the ndmz namespace
	BridgeNDMZ = "br-ndmz"
	//NetNSNDMZ name of the dmz namespace
	NetNSNDMZ = "ndmz"

	ndmzNsMACDerivationSuffix6 = "-ndmz6"
	ndmzNsMACDerivationSuffix4 = "-ndmz4"

	// DMZPub4 ipv4 public interface
	DMZPub4 = "npub4"
	// DMZPub6 ipv6 public interface
	DMZPub6 = "npub6"

	//nrPubIface is the name of the public interface in a network resource
	nrPubIface = "public"
)

// DMZ is an interface used to create an DMZ network namespace
type DMZ interface {
	// create the ndmz network namespace and all requires network interfaces
	Create(ctx context.Context) error
	// delete the ndmz network namespace and clean up all network interfaces
	Delete() error
	// link a network resource from a user network to ndmz
	AttachNR(networkID string, nr *nr.NetResource, ipamLeaseDir string) error
	// Return the interface used by ndmz to router public ipv6 traffic
	IP6PublicIface() string
	//configure an address on the public IPv6 interface
	SetIP6PublicIface(net.IPNet) error
}

// FindIPv6Master finds which interface to use as master for NDMZ npub6 interface
func FindIPv6Master() (master string, err error) {

	if namespace.Exists(types.PublicNamespace) {
		pubNS, err := namespace.GetByName(types.PublicNamespace)
		if err != nil {
			return "", err
		}
		defer pubNS.Close()

		parent, err := ifaceutil.ParentIface(types.PublicIface, pubNS)
		if err != nil {
			return "", err
		}
		return parent.Attrs().Name, nil
	}

	master, err = ifaceutil.HostIPV6Iface()
	if err != nil {
		return "", errors.Wrap(err, "failed to find a valid network interface to use as parent for ndmz public interface")
	}

	return master, nil
}

func createPubIface6(name, master, nodeID string, netNS ns.NetNS) error {
	if !ifaceutil.Exists(name, netNS) {
		if _, err := macvlan.Create(name, master, netNS); err != nil {
			return err
		}
	}

	return netNS.Do(func(_ ns.NetNS) error {
		// set mac address to something static to make sure we receive the same IP from a DHCP server
		mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID + ndmzNsMACDerivationSuffix6))
		log.Debug().
			Str("mac", mac.String()).
			Str("interface", name).
			Msg("set mac on ipv6 ndmz public iface")

		return ifaceutil.SetMAC(name, mac, nil)
	})
}

func createPubIface4(name, nodeID string, netNS ns.NetNS) error {
	if !ifaceutil.Exists(name, netNS) {
		if _, err := macvlan.Create(name, types.DefaultBridge, netNS); err != nil {
			return err
		}
	}

	return netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "1"); err != nil {
			return errors.Wrapf(err, "failed to disable ip6 on %s", name)
		}
		// set mac address to something static to make sure we receive the same IP from a DHCP server
		mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID + ndmzNsMACDerivationSuffix4))
		log.Debug().
			Str("mac", mac.String()).
			Str("interface", name).
			Msg("set mac on ipv4 ndmz public iface")

		return ifaceutil.SetMAC(name, mac, nil)
	})
}

func createRoutingBridge(name string, netNS ns.NetNS) error {
	if !bridge.Exists(name) {
		if _, err := bridge.New(name); err != nil {
			return errors.Wrapf(err, "couldn't create bridge %s", name)
		}
	}

	const tonrsIface = "tonrs"

	if !ifaceutil.Exists(tonrsIface, netNS) {
		if _, err := macvlan.Create(tonrsIface, name, netNS); err != nil {
			return errors.Wrapf(err, "ndmz: couldn't create %s", tonrsIface)
		}
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "1"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
	}

	return netNS.Do(func(_ ns.NetNS) error {

		link, err := netlink.LinkByName(tonrsIface)
		if err != nil {
			return err
		}
		if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", tonrsIface), "0"); err != nil {
			return errors.Wrapf(err, "failed to enable ip6 on interface %s", tonrsIface)
		}

		addrs := []*netlink.Addr{
			{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("100.127.0.1"),
					Mask: net.CIDRMask(16, 32),
				},
			},
			{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("fe80::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
			{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("fd00::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
		}

		for _, addr := range addrs {
			err = netlink.AddrAdd(link, addr)
			if err != nil && !os.IsExist(err) {
				return err
			}
		}

		return netlink.LinkSetUp(link)
	})
}

func applyFirewall() error {
	buf := bytes.Buffer{}

	data := struct {
		YggPorts string
	}{
		YggPorts: strings.Join([]string{
			strconv.Itoa(yggdrasil.YggListenTCP),
			strconv.Itoa(yggdrasil.YggListenTLS),
			strconv.Itoa(yggdrasil.YggListenLinkLocal),
		}, ","),
	}

	if err := fwTmpl.Execute(&buf, data); err != nil {
		return errors.Wrap(err, "failed to build nft rule set")
	}

	if err := nft.Apply(&buf, NetNSNDMZ); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

func convertIpv4ToIpv6(ip net.IP) net.IP {
	var ipv6 string
	if len(ip) == net.IPv4len {
		ipv6 = fmt.Sprintf("fd00::%02x%02x", ip[2], ip[3])
	} else {
		ipv6 = fmt.Sprintf("fd00::%02x%02x", ip[14], ip[15])
	}
	fmt.Println(ipv6)
	return net.ParseIP(ipv6)
}

func configureYggdrasil(subnetIP net.IPNet) error {
	netns, err := namespace.GetByName(NetNSNDMZ)
	if err != nil {
		return err
	}

	err = netns.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(DMZPub6)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(link, &netlink.Addr{
			IPNet: &subnetIP,
		}); err != nil && !os.IsExist(err) {
			return err
		}
		return nil
	})
	return err
}
