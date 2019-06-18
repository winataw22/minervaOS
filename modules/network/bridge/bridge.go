package bridge

import (
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

// New creates a bridge and set it up
func New(name string) (*netlink.Bridge, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	bridge := &netlink.Bridge{
		LinkAttrs: attrs,
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return bridge, err
	}

	var err error
	defer func() {
		if err != nil {
			if err := netlink.LinkDel(bridge); err != nil {
				log.Error().Err(err).Msgf("failed to delete bridge %s", bridge.Name)
			}
		}
	}()
	err = netlink.LinkSetUp(bridge)
	return bridge, nil
}

// Delete remove a bridge
func Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if err == (netlink.LinkNotFoundError{}) {
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
	briges, err := List()
	if err != nil {
		return false
	}
	found := false
	for _, bridge := range briges {
		if bridge.Attrs().Name == name {
			found = true
			break
		}
	}
	return found
}

// AttachNic attaches an interface to a bridge
func AttachNic(link netlink.Link, bridge *netlink.Bridge) error {
	return netlink.LinkSetMaster(link, bridge)
}

// DetachNic detaches an interface to a bridge
func DetachNic(bridge *netlink.Bridge) error {
	return netlink.LinkSetNoMaster(bridge)
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
