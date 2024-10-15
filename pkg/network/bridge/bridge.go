package bridge

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/vishvananda/netlink"
)

// New creates a bridge and set it up
func New(name string) (*netlink.Bridge, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	attrs.MTU = 1500
	enable := true
	bridge := &netlink.Bridge{
		LinkAttrs:     attrs,
		VlanFiltering: &enable,
	}

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
func Attach(link netlink.Link, bridge *netlink.Bridge, vlan *uint16, name ...string) error {
	if link.Type() == "device" {
		if vlan != nil {
			log.Warn().Msg("vlan is not supported in dual nic setup")
		}

		return attachNic(link, bridge, nil)
	} else if link.Type() == "bridge" {
		linkBr := link.(*netlink.Bridge)
		n := vethName(link.Attrs().Name, bridge.Name)
		if len(name) > 0 {
			n = name[0]
		}
		//we need to create an veth pair to wire 2 bridges.
		if err := ifaceutil.MakeVethPair(n, bridge.Name, 1500, nil); err != nil {
			return fmt.Errorf("failed to create veth pair from link %q to bridge %q: %w", n, bridge.Name, err)
		}

		veth, err := netlink.LinkByName(n)
		if err != nil {
			return fmt.Errorf("no link found with name %q: %w", n, err)
		}
		return attachNic(veth, linkBr, vlan)
	}

	return fmt.Errorf("unsupported link type '%s'", link.Type())
}

// attachNic attaches an interface to a bridge
func attachNic(link netlink.Link, bridge *netlink.Bridge, vlan *uint16) error {
	// Jan said this was fine
	if err := netlink.LinkSetUp(link); err != nil {
		return errors.Wrap(err, "could not set veth peer up")
	}
	// disable ipv6 on slave

	if err := options.Set(link.Attrs().Name, options.IPv6Disable(true)); err != nil {
		return errors.Wrap(err, "failed to disable ipv6 on link interface")
	}
	if err := netlink.LinkSetMaster(link, bridge); err != nil {
		return errors.Wrapf(err, "failed to attach link %s to bridge %s", link.Attrs().Name, bridge.Name)
	}

	if vlan == nil {
		return nil
	}

	if err := netlink.BridgeVlanDel(link, 1, true, true, false, false); err != nil {
		return errors.Wrapf(err, "failed to delete default vlan tag on device '%s'", link.Attrs().Name)
	}

	if err := netlink.BridgeVlanAdd(link, *vlan, true, true, false, false); err != nil {
		return errors.Wrapf(err, "failed to set vlan on device '%s'", link.Attrs().Name)
	}

	return nil
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
