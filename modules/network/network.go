package network

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/vishvananda/netlink"
)

const (
	defaultBridge = "zos"
)

// Bootstrap creates the default bridge of 0-OS
// it then walk over all pluggued network interfaces and attaches them to the bridge
// one by one and try to get an IP. Bootstrap stops as soon as one of the interface receives and ip with a
// default route
func Bootstrap() error {

	log.Info().Msg("Create default bridge")
	bridge, err := CreateBridge(defaultBridge)
	if err != nil {
		log.Error().Err(err).Msgf("failed to create bridge %s", defaultBridge)
		return err
	}

	links, err := interfaces()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return err
	}

	var defaultGW *netlink.Device

	for _, device := range filterDevices(links) {
		log.Info().Str("interface", device.Name).Msg("probe interface")
		// TODO: support noautonic kernel params
		if device.Name == "lo" {
			continue
		}

		if !isVirtEth(device.Name) {
			if !isPlugged(device.Name) {
				log.Info().Str("interface", device.Name).Msg("interface is not plugged in, skipping")
				continue
			}
		}

		// TODO: see if we need to set the if down
		if err := netlink.LinkSetUp(device); err != nil {
			log.Info().Str("interface", device.Name).Msg("failed to bring interface up")
			continue
		}

		log.Info().Str("interface", device.Name).Msg("attach interface to default bridge")
		if err := BridgeAttachNic(device, bridge); err != nil {
			log.Warn().Err(err).
				Str("device", device.Name).
				Str("bridge", bridge.Name).
				Msg("fail to attach device to bridge")
			continue
		}

		log.Info().Str("interface", device.Name).Msg("start dhcp probing")
		valid, err := dhcpProbe(bridge.Name)
		if err != nil {
			log.Warn().Err(err).Str("device", device.Name).Msg("dhcp probing unexpected error")
			continue
		}

		if valid {
			defaultGW = device
			break
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

// DefaultBridgeName return the name of the default bridge
// created by the network bootstrap
func DefaultBridgeName() string {
	return defaultBridge
}
