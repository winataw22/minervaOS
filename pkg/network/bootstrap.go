package network

import (
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg/network/dhcp"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/vishvananda/netlink"
)

// DefaultBridge is the name of the default bridge created
// by the bootstrap of networkd
const DefaultBridge = "zos"

// Bootstrap creates the default bridge of 0-OS
// it then walk over all plugged network interfaces and attaches them to the bridge
// one by one and try to get an IP. Bootstrap stops as soon as one of the interface receives and ip with a
// default route
func Bootstrap() error {

	log.Info().Msg("Create default bridge")
	br, err := bridge.New(DefaultBridge)
	if err != nil {
		log.Error().Err(err).Msgf("failed to create bridge %s", DefaultBridge)
		return err
	}

	if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", DefaultBridge), "0"); err != nil {
		return errors.Wrapf(err, "failed to disable ip6 on bridge %s", DefaultBridge)
	}

	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return err
	}

	var defaultGW *netlink.Device

	for _, link := range ifaceutil.LinkFilter(links, []string{"device"}) {
		device, ok := link.(*netlink.Device)
		if !ok {
			continue
		}
		log.Info().Str("interface", device.Name).Msg("probe interface")
		// TODO: support noautonic kernel params
		if device.Name == "lo" {
			continue
		}

		if addresses, err := netlink.AddrList(device, netlink.FAMILY_ALL); err == nil {
			for _, address := range addresses {
				if err := netlink.AddrDel(device, &address); err != nil {
					log.Error().
						Err(err).
						Str("address", address.String()).
						Str("interface", device.Name).
						Msg("failed to remove assigned address")
				}
			}
		}

		if err := netlink.LinkSetUp(device); err != nil {
			log.Info().Str("interface", device.Name).Msg("failed to bring interface up")
			continue
		}

		if !ifaceutil.IsVirtEth(device.Name) && !ifaceutil.IsPluggedTimeout(device.Name, time.Second*5) {
			log.Info().Str("interface", device.Name).Msg("interface is not plugged in, skipping")
			continue
		}

		log.Info().Str("interface", device.Name).Msg("attach interface to default bridge")
		if err := bridge.AttachNicWithMac(device, br); err != nil {
			log.Warn().Err(err).
				Str("device", device.Name).
				Str("bridge", br.Name).
				Msg("fail to attach device to bridge")
			continue
		}

		log.Info().Str("interface", device.Name).Msg("start dhcp probing")
		valid, err := dhcp.Probe(br.Name, netlink.FAMILY_ALL)
		if err != nil {
			log.Warn().Err(err).Str("device", device.Name).Msg("dhcp probing unexpected error")
			if err := bridge.DetachNic(br); err != nil {
				log.Warn().Err(err).Str("device", device.Name).Msg("error detaching device from default bridge")
			}
			continue
		}

		if valid {
			defaultGW = device
			break
		} else {
			if err := bridge.DetachNic(br); err != nil {
				log.Warn().Err(err).Str("device", device.Name).Msg("error detaching device from default bridge")
			}
		}
	}

	if defaultGW == nil {
		err = fmt.Errorf("no interface with default gateway found")
		log.Error().Err(err).Msg("cannot configure network")
		return err
	}

	log.Info().Str("device", defaultGW.Name).Msg("default gateway found")
	return nil
}

// DefaultBridgeValid validates default bridge exists and of correct type
func DefaultBridgeValid() error {
	link, err := netlink.LinkByName(DefaultBridge)
	if err != nil {
		return err
	}

	if link.Type() != "bridge" {
		return fmt.Errorf("invalid default bridge type (%s) expecting (bridge)", link.Type())
	}

	return nil
}
