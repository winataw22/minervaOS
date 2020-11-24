package ndmz

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg/network/dhcp"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
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

// DualStack implement DMZ interface using dual stack ipv4/ipv6
type DualStack struct {
	nodeID     string
	ipv6Master string
}

// NewDualStack creates a new DMZ DualStack
func NewDualStack(nodeID string) *DualStack {
	return &DualStack{
		nodeID: nodeID,
	}
}

//Create create the NDMZ network namespace and configure its default routes and addresses
func (d *DualStack) Create(ctx context.Context) error {
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

	master, err := FindIPv6Master()
	if err != nil {
		return err
	}

	if master == "" {
		return errors.Wrap(err, "cannot find a valid physical interface to use as master for ndmz npub6")
	}
	d.ipv6Master = master

	if err := createPubIface6(DMZPub6, master, d.nodeID, netNS); err != nil {
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

		if err := waitIP4(); err != nil {
			return err
		}

		if err := waitIP6(); err != nil {
			return err
		}
		return nil
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
func (d *DualStack) Delete() error {
	netNS, err := namespace.GetByName(NetNSNDMZ)
	if err == nil {
		if err := namespace.Delete(netNS); err != nil {
			return errors.Wrap(err, "failed to delete ndmz network namespace")
		}
	}

	return nil
}

// AttachNR links a network resource to the NDMZ
func (d *DualStack) AttachNR(networkID string, nr *nr.NetResource, ipamLeaseDir string) error {
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
func (d *DualStack) SetIP6PublicIface(subnet net.IPNet) error {
	return configureYggdrasil(subnet)
}

// IP6PublicIface implements DMZ interface
func (d *DualStack) IP6PublicIface() string {
	return d.ipv6Master
}

// waitIP4 waits to receives some IPv4 from DHCP
// it returns the pid of the dhcp process or an error
func waitIP4() error {
	// run DHCP to interface public in ndmz
	probe := dhcp.NewProbe()

	if err := probe.Start(DMZPub4); err != nil {
		return err
	}
	defer probe.Stop()

	link, err := netlink.LinkByName(DMZPub4)
	if err != nil {
		return err
	}

	cTimeout := time.After(time.Second * 30)
	for {
		select {
		case <-cTimeout:
			return errors.Errorf("public interface in ndmz did not received an IP. make sure DHCP is working")
		default:
			hasGW, _, err := ifaceutil.HasDefaultGW(link, netlink.FAMILY_V4)
			if err != nil {
				return err
			}
			if !hasGW {
				time.Sleep(time.Second)
				continue
			}
			return nil
		}
	}
}

func waitIP6() error {
	if err := ifaceutil.SetLoUp(); err != nil {
		return errors.Wrapf(err, "ndmz: couldn't bring lo up in ndmz namespace")
	}
	// first, disable forwarding, so we can get an IPv6 deft route on public from an RA
	if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "0"); err != nil {
		return errors.Wrapf(err, "ndmz: failed to disable ipv6 forwarding in ndmz namespace")
	}
	// also, set kernel parameter that public always accepts an ra even when forwarding
	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.accept_ra", DMZPub6), "2"); err != nil {
		return errors.Wrapf(err, "ndmz: failed to accept_ra=2 in ndmz namespace")
	}
	// the more, also accept defaultrouter (if isp doesn't have fe80::1 on his deft gw)
	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.accept_ra_defrtr", DMZPub6), "1"); err != nil {
		return errors.Wrapf(err, "ndmz: failed to enable enable_defrtr=1 in ndmz namespace")
	}
	// ipv4InterfaceArpProxySysctlTemple sets proxy_arp by default, not sure if that's a good idea
	// but we disable only here because the rest works.
	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv4.conf.%s.proxy_arp", DMZPub6), "0"); err != nil {
		return errors.Wrapf(err, "ndmz: couldn't disable proxy-arp on %s in ndmz namespace", DMZPub6)
	}

	var routes []netlink.Route
	getRoutes := func() (err error) {
		log.Info().Msg("wait for slaac to give ipv6")
		// check if in the mean time SLAAC gave us an IPv6 deft gw, save it, and reapply after enabling forwarding
		checkipv6 := net.ParseIP("2606:4700:4700::1111")
		routes, err = netlink.RouteGet(checkipv6)
		if err != nil {
			return errors.Wrapf(err, "ndmz: failed to get the IPv6 routes in ndmz")
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 122 * time.Second // default RA from router is every 60 secs
	if err := backoff.Retry(getRoutes, bo); err != nil {
		return err
	}

	if len(routes) == 1 {
		if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
			return errors.Wrapf(err, "ndmz: failed to enable ipv6 forwarding in ndmz namespace")
		}
	}
	return nil
}
