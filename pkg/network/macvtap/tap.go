package macvtap

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/vishvananda/netlink"
)

// CreateMACvTap creates a new macvtap device with the given name, and sets the master interface
func CreateMACvTap(name string, master string, hw net.HardwareAddr) (*netlink.Macvtap, error) {
	masterIface, err := netlink.LinkByName(master)
	if err != nil {
		return nil, errors.Wrap(err, "failed to look up tap master")
	}

	lan := netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         1500,
			Name:        name,
			ParentIndex: masterIface.Attrs().Index,
			TxQLen:      500,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	tap := &netlink.Macvtap{Macvlan: lan}
	if err = netlink.LinkAdd(tap); err != nil {
		return nil, errors.Wrap(err, "could not add tap device")
	}

	defer func() {
		if err != nil {
			_ = netlink.LinkDel(tap)
		}
	}()

	if err = netlink.LinkSetHardwareAddr(tap, hw); err != nil {
		return nil, errors.Wrap(err, "could not set hw address")
	}

	if err := options.Set(name, options.IPv6Disable(true)); err != nil {
		return nil, errors.Wrap(err, "failed to disable ipv6 on interface host side")
	}

	// Re-fetch tap to get all properties/attributes
	var link netlink.Link
	link, err = netlink.LinkByName(name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to refetch tap interface %q", name)
	}

	var ok bool
	tap, ok = link.(*netlink.Macvtap)
	if !ok {
		// right now, the netlink lib returns a `*GenericLink` for the tap interface,
		// so  assign properties to a blank tap
		gl, ok := link.(*netlink.GenericLink)
		if !ok {
			return nil, fmt.Errorf("link %s should be of type macvtap", name)
		}

		tap = &netlink.Macvtap{
			Macvlan: netlink.Macvlan{
				LinkAttrs: gl.LinkAttrs,
				Mode:      netlink.MACVLAN_MODE_BRIDGE,
			},
		}
	} else {
		// make sure we have the right interface type
		if tap.Mode != netlink.MACVLAN_MODE_BRIDGE {
			return nil, errors.New("tuntap iface does not have the expected 'bridge' mode")
		}
	}

	if err = netlink.LinkSetUp(tap); err != nil {
		return nil, errors.Wrap(err, "could not bring up macvtap iface")
	}

	if err = netlink.SetPromiscOn(tap); err != nil {
		return nil, errors.Wrap(err, "could not bring set promsic on iface")
	}

	return tap, nil
}
