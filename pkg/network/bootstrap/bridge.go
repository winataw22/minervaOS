package bootstrap

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/vishvananda/netlink"
)

// DefaultBridgeValid validates default bridge exists and of correct type
func DefaultBridgeValid() error {
	link, err := netlink.LinkByName(types.DefaultBridge)
	if err != nil {
		return err
	}

	if link.Type() != "bridge" {
		return fmt.Errorf("invalid default bridge type (%s) expecting (bridge)", link.Type())
	}

	return nil
}

// CreateDefaultBridge creates the default bridge of the node that will received
// the management interface
func CreateDefaultBridge(name string) (*netlink.Bridge, error) {
	log.Info().Msg("Create default bridge")
	br, err := bridge.New(name)
	if err != nil {
		return nil, err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", name), "0"); err != nil {
		return nil, errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
	}
	return br, nil
}
