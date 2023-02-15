package bridge

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/vishvananda/netlink"
)

// New creates a bridge and set it up
func New(name string) (*netlink.Bridge, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	attrs.MTU = 1500
	bridge := &netlink.Bridge{LinkAttrs: attrs}

	if err := netlink.LinkAdd(bridge); err != nil && !os.IsExist(err) {
		return bridge, err
	}

	if err := netlink.LinkSetUp(bridge); err != nil {
		return nil, err
	}

	// get the bridge object from the kernel now
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup %q: %v", name, err)
	}

	newBr, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%q already exists but is not a bridge", name)
	}

	return newBr, nil
}

// Get a bridge by name
func Get(name string) (*netlink.Bridge, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, errors.Wrapf(err, "bridge %s not found", name)
	}

	if link.Type() != "bridge" {
		return nil, fmt.Errorf("device '%s' is not a bridge", name)
	}

	return link.(*netlink.Bridge), nil
}

// Delete remove a bridge
func Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return nil
		}
		return err
	}
	return netlink.LinkDel(link)
}

// List list all the bridge interfaces
func List() ([]*netlink.Bridge, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}
	return filterBridge(links), nil
}

// Exists check if a bridge named name already exists
func Exists(name string) bool {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return false
	}
	_, ok := link.(*netlink.Bridge)
	return ok
}

func vethName(from, to string) string {
	name := fmt.Sprintf("%sto%s", from, to)
	if len(name) > 13 {
		return name[:13]
	}

	return name
}

// Attach attaches any link to a bridge, based on the link type
// can be directly plugged or crossed over with a veth pair
// if name is provided, the name will be used in case of veth pair instead of
// a generated name
func Attach(link netlink.Link, bridge *netlink.Bridge, name ...string) error {
	if link.Type() == "device" {
		return AttachNic(link, bridge)
	} else if link.Type() == "bridge" {
		linkBr := link.(*netlink.Bridge)
		n := vethName(link.Attrs().Name, bridge.Name)
		if len(name) > 0 {
			n = name[0]
		}
		//we need to create an veth pair to wire 2 bridges.
		veth, err := ifaceutil.MakeVethPair(n, bridge.Name, 1500)
		if err != nil {
			return err
		}

		return AttachNic(veth, linkBr)
	}

	return fmt.Errorf("unsupported link type '%s'", link.Type())
}

// AttachNic attaches an interface to a bridge
func AttachNic(link netlink.Link, bridge *netlink.Bridge) error {
	// Jan said this was fine
	if err := netlink.LinkSetUp(link); err != nil {
		return errors.Wrap(err, "could not set veth peer up")
	}
	// disable ipv6 on slave

	if err := options.Set(link.Attrs().Name, options.IPv6Disable(true)); err != nil {
		return errors.Wrap(err, "failed to disable ipv6 on link interface")
	}
	return netlink.LinkSetMaster(link, bridge)
}

// List all nics attached to a bridge
func ListNics(bridge *netlink.Bridge, physical bool) ([]netlink.Link, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	filtered := links[:0]

	for _, link := range links {
		if link.Attrs().MasterIndex != bridge.Index {
			continue
		}
		if physical && link.Type() != "device" {
			continue
		}
		filtered = append(filtered, link)
	}

	return filtered, nil
}

// AttachNicWithMac attaches an interface to a bridge and sets
// the MAC of the bridge to the same of the NIC
func AttachNicWithMac(link netlink.Link, bridge *netlink.Bridge) error {
	hwaddr := link.Attrs().HardwareAddr

	err := netlink.LinkSetHardwareAddr(bridge, hwaddr)
	if err != nil {
		return err
	}
	return netlink.LinkSetMaster(link, bridge)
}

// DetachNic detaches an interface from a bridge
func DetachNic(link netlink.Link) error {
	return netlink.LinkSetNoMaster(link)
}

func filterBridge(links []netlink.Link) []*netlink.Bridge {
	bridges := []*netlink.Bridge{}

	for _, link := range links {
		if link.Type() == "bridge" {
			bridge, ok := link.(*netlink.Bridge)
			if ok {
				bridges = append(bridges, bridge)
			}
		}
	}
	return bridges
}
