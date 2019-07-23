package network

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"

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

		// TODO: see if we need to set the if down
		if err := netlink.LinkSetUp(device); err != nil {
			log.Info().Str("interface", device.Name).Msg("failed to bring interface up")
			continue
		}

		if !ifaceutil.IsVirtEth(device.Name) && !ifaceutil.IsPluggedTimeout(device.Name, time.Second*5) {
			log.Info().Str("interface", device.Name).Msg("interface is not plugged in, skipping")
			continue
		}

		log.Info().Str("interface", device.Name).Msg("attach interface to default bridge")
		if err := bridge.AttachNic(device, br); err != nil {
			log.Warn().Err(err).
				Str("device", device.Name).
				Str("bridge", br.Name).
				Msg("fail to attach device to bridge")
			continue
		}

		log.Info().Str("interface", device.Name).Msg("start dhcp probing")
		valid, err := dhcpProbe(br.Name)
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
