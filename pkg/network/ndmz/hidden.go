package ndmz

import (
	"context"
	"net"
	"os"

	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/zinit"

	"github.com/threefoldtech/zos/pkg/network/nr"

	"github.com/threefoldtech/zos/pkg/network/macvlan"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/namespace"
)

// Hidden implement DMZ interface using ipv4 only
type Hidden struct {
	nodeID string
}

// NewHidden creates a new DMZ Hidden
func NewHidden(nodeID string) *Hidden {
	return &Hidden{
		nodeID: nodeID,
	}
}

//Create create the NDMZ network namespace and configure its default routes and addresses
func (d *Hidden) Create(ctx context.Context) error {
	netNS, err := namespace.GetByName(NetNSNDMZ)
	if err != nil {
		netNS, err = namespace.Create(NetNSNDMZ)
		if err != nil {
			return err
		}
	}
	defer netNS.Close()

	if err := createRoutingBridge(BridgeNDMZ, netNS); err != nil {
		return errors.Wrapf(err, "ndmz: createRoutingBride error")
	}

	if err := createPubIface6(DMZPub6, types.DefaultBridge, d.nodeID, netNS); err != nil {
		return errors.Wrapf(err, "ndmz: could not node create pub iface 6")
	}

	if err := createPubIface4(DMZPub4, d.nodeID, netNS); err != nil {
		return errors.Wrapf(err, "ndmz: could not create pub iface 4")
	}

	if err = applyFirewall(); err != nil {
		return err
	}

	err = netNS.Do(func(_ ns.NetNS) error {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
			return errors.Wrapf(err, "failed to enable forwarding in ndmz")
		}

		return waitIP4()
	})
	if err != nil {
		return err
	}

	z, err := zinit.New("")
	if err != nil {
		return err
	}
	dhcpMon := NewDHCPMon(DMZPub4, NetNSNDMZ, z)
	go dhcpMon.Start(ctx)

	return nil
}

// Delete deletes the NDMZ network namespace
func (d *Hidden) Delete() error {
	netNS, err := namespace.GetByName(NetNSNDMZ)
	if err == nil {
		if err := namespace.Delete(netNS); err != nil {
			return errors.Wrap(err, "failed to delete ndmz network namespace")
		}
	}

	return nil
}

// AttachNR links a network resource to the NDMZ
func (d *Hidden) AttachNR(networkID string, nr *nr.NetResource, ipamLeaseDir string) error {
	nrNSName, err := nr.Namespace()
	if err != nil {
		return err
	}

	nrNS, err := namespace.GetByName(nrNSName)
	if err != nil {
		return err
	}

	if !ifaceutil.Exists(nrPubIface, nrNS) {
		if _, err = macvlan.Create(nrPubIface, BridgeNDMZ, nrNS); err != nil {
			return err
		}
	}

	return nrNS.Do(func(_ ns.NetNS) error {
		addr, err := allocateIPv4(networkID, ipamLeaseDir)
		if err != nil {
			return errors.Wrap(err, "ip allocation for network resource")
		}

		pubIface, err := netlink.LinkByName(nrPubIface)
		if err != nil {
			return err
		}

		if err := netlink.AddrAdd(pubIface, &netlink.Addr{IPNet: addr}); err != nil && !os.IsExist(err) {
			return err
		}

		ipv6 := convertIpv4ToIpv6(addr.IP)
		log.Debug().Msgf("ndmz: setting public NR ip to: %s from %s", ipv6.String(), addr.IP.String())

		if err := netlink.AddrAdd(pubIface, &netlink.Addr{IPNet: &net.IPNet{
			IP:   ipv6,
			Mask: net.CIDRMask(64, 128),
		}}); err != nil && !os.IsExist(err) {
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

		err = netlink.RouteAdd(&netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        net.ParseIP("fe80::1"),
			LinkIndex: pubIface.Attrs().Index,
		})
		if err != nil && !os.IsExist(err) {
			return err
		}

		return nil
	})
}

// SetIP6PublicIface implements DMZ interface
func (d *Hidden) SetIP6PublicIface(subnet net.IPNet) error {
	return configureYggdrasil(subnet)
}

// IP6PublicIface implements DMZ interface
func (d *Hidden) IP6PublicIface() string {
	return types.DefaultBridge
}
