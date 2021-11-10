package tuntap

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/vishvananda/netlink"
)

// CreateTap creates a new tap device with the given name, and sets the master interface
func CreateTap(name string, master string) (*netlink.Tuntap, error) {
	masterIface, err := netlink.LinkByName(master)
	if err != nil {
		return nil, errors.Wrap(err, "failed to look up tap master")
	}

	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         1500,
			Name:        name,
			ParentIndex: masterIface.Attrs().Index,
		},
		Mode: netlink.TUNTAP_MODE_TAP,
	}

	if err = netlink.LinkAdd(tap); err != nil {
		return nil, errors.Wrap(err, "could not add tap device")
	}
	defer func() {
		if err != nil {
			_ = netlink.LinkDel(tap)
		}
	}()

	// Setting the master iface on the link attrs at creation time seems to not work
	// (at least not always), so explicitly set the master again once the iface is added.
	if err = netlink.LinkSetMaster(tap, masterIface); err != nil {
		return nil, errors.Wrap(err, "could not set tap master")
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
	tap, ok = link.(*netlink.Tuntap)
	if !ok {
		// right now, the netlink lib returns a `*GenericLink` for the tap interface,
		// so  assign properties to a blank tap
		gl, ok := link.(*netlink.GenericLink)
		if !ok {
			return nil, fmt.Errorf("link %s should be of type tuntap", name)
		}
		tap = &netlink.Tuntap{LinkAttrs: gl.LinkAttrs, Mode: netlink.TUNTAP_MODE_TAP}
	} else {
		// make sure we have the right interface type
		if tap.Mode != netlink.TUNTAP_MODE_TAP {
			return nil, errors.New("tuntap iface does not have the expected 'tap' mode")
		}
	}

	if err = netlink.SetPromiscOn(tap); err != nil {
		return nil, errors.Wrap(err, "could not bring set promsic on iface")
	}

	if err = netlink.LinkSetUp(tap); err != nil {
		return nil, errors.Wrap(err, "could not bring up tap iface")
	}

	return tap, nil
}
