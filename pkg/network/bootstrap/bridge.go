package bootstrap

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
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

	// check there is a default route for ipv6 on the zos bridge
	hasGW, _, err := ifaceutil.HasDefaultGW(link, netlink.FAMILY_V6)
	if err != nil {
		return err
	}

	// If we do not have a default route, then toggle accept_ra to force slaac to send us the route again
	if !hasGW {
		log.Info().Msg("no default route found, try to turn accept_ra off and on again")

		if _, err := sysctl.Sysctl("net.ipv6.conf.all.accept_ra", "0"); err != nil {
			log.Fatal().Err(err).Msgf("failed to disable accept_ra zos")
		}

		if _, err := sysctl.Sysctl("net.ipv6.conf.all.accept_ra", "1"); err != nil {
			log.Fatal().Err(err).Msgf("failed to enable accept_ra zos")
		}
	}

	// zos doesn't not need to forward anything, disable forwarding on it
	if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "0"); err != nil {
		log.Fatal().Err(err).Msgf("failed to disable ipv6 forwarding on bridge zos")
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

	if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "0"); err != nil {
		return nil, errors.Wrapf(err, "failed to disable ipv6 forwarding")
	}

	return br, nil
}
