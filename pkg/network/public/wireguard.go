package public

import (
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/network/wireguard"
	"github.com/vishvananda/netlink"
)

// NewWireguard creates a new wireguard instance in public namespace
// if exits
func NewWireguard(name string) (wg *wireguard.Wireguard, err error) {
	f := func(host ns.NetNS) error {
		wg, err = wireguard.New(name)
		if err != nil {
			return err
		}

		if host == nil {
			return nil
		}

		// we need to move it to the host
		return netlink.LinkSetNsFd(wg, int(host.Fd()))
	}

	ns := getPublicNamespace()
	if ns != nil {
		defer ns.Close()
		if err := ns.Do(f); err != nil {
			return nil, err
		}
		// load it from host ns
		return wireguard.GetByName(wg.Attrs().Name)
	}

	return wg, f(nil)
}
