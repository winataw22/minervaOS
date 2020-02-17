package bridge

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
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

// AttachNic attaches an interface to a bridge
func AttachNic(link netlink.Link, bridge *netlink.Bridge) error {
	return netlink.LinkSetMaster(link, bridge)
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
