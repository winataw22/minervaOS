package primitives

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// PublicIP structure
type PublicIP struct {
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IPNet `json:"ip"`
}

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

func (p *Provisioner) publicIPProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.publicIPProvisionImpl(ctx, reservation)
}

func (p *Provisioner) publicIPProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result PublicIPResult, err error) {
	config := PublicIP{}

	network := stubs.NewNetworkerStub(p.zbus)

	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return PublicIPResult{}, errors.Wrap(err, "failed to decode reservation schema")
	}

	pubIP6Base, err := network.GetPublicIPv6Subnet()
	if err != nil {
		return PublicIPResult{}, errors.Wrap(err, "could not look up ipv6 prefix")
	}

	tapName := fmt.Sprintf("p-%s", reservation.ID) // TODO: clean this up, needs to come form networkd
	fName := filterName(reservation.ID)
	mac := ifaceutil.HardwareAddrFromInputBytes(config.IP.IP.To4())

	predictedIPv6, err := predictedSlaac(pubIP6Base.IP, mac.String())
	if err != nil {
		return PublicIPResult{}, errors.Wrap(err, "could not look up ipv6 prefix")
	}

	err = setupFilters(ctx, fName, tapName, config.IP.IP.To4().String(), predictedIPv6, mac.String())
	return PublicIPResult{
		ID: reservation.ID,
		IP: config.IP.IP.String(),
	}, err
}

func (p *Provisioner) publicIPDecomission(ctx context.Context, reservation *provision.Reservation) error {
	// Disconnect the public interface from the network if one exists
	network := stubs.NewNetworkerStub(p.zbus)
	fName := filterName(reservation.ID)
	if err := teardownFilters(ctx, fName); err != nil {
		log.Error().Err(err).Msg("could not remove filter rules")
	}
	return network.DisconnectPubTap(reservation.ID)
}

func filterName(reservationID string) string {
	return fmt.Sprintf("r-%s", reservationID)
}

func setupFilters(ctx context.Context, fName string, iface string, ip string, ipv6 string, mac string) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c",
		fmt.Sprintf(`# add vm
# add a chain for the vm public interface in arp and bridge
nft 'add chain arp filter %[1]s'
nft 'add chain bridge filter %[1]s'

# make nft jump to vm chain
nft 'add rule arp filter input iifname "%[2]s" jump %[1]s'
nft 'add rule bridge filter forward iifname "%[2]s" jump %[1]s'

# arp rule for vm 
nft 'add rule arp filter %[1]s arp operation reply arp saddr ip . arp saddr ether != { %[3]s . %[4]s } drop'

# filter on L2 fowarding of non-matching ip/mac, drop RA,dhcpv6,dhcp
nft 'add rule bridge filter %[1]s ip saddr . ether saddr != { %[3]s . %[4]s } counter drop'
nft 'add rule bridge filter %[1]s ip6 saddr . ether saddr != { %[5]s . %[4]s } counter drop'
nft 'add rule bridge filter %[1]s icmpv6 type nd-router-advert drop'
nft 'add rule bridge filter %[1]s ip6 version 6 udp sport 547 drop'
nft 'add rule bridge filter %[1]s ip version 4 udp sport 67 drop'`, fName, iface, ip, mac, ipv6))

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "could not setup firewall rules for public ip")
	}
	return nil
}

func teardownFilters(ctx context.Context, fName string) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c",
		fmt.Sprintf(`# in bridge table
nft 'flush chain bridge filter %[1]s'
# jump to chain rule
a=$( nft -a list table bridge filter | awk '/jump %[1]s/{ print $NF}' )
nft 'delete rule bridge filter forward handle '${a}
# chain itself
a=$( nft -a list table bridge filter | awk '/chain %[1]s/{ print $NF}' )
nft 'delete chain bridge filter handle '${a}

# in arp table
nft 'flush chain arp filter %[1]s'
# jump to chain rule 
a=$( nft -a list table arp filter | awk '/jump %[1]s/{ print $NF}' )
nft 'delete rule arp filter input handle '${a}
# chain itself
a=$( nft -a list table arp filter | awk '/chain %[1]s/{ print $NF}' )
nft 'delete chain arp filter handle '${a}`, fName))

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "could not setup firewall rules for public ip")
	}
	return nil
}

// modified version of: https://github.com/MalteJ/docker/blob/f09b7897d2a54f35a0b26f7cbe750b3c9383a553/daemon/networkdriver/bridge/driver.go#L585
func predictedSlaac(base net.IP, mac string) (string, error) {
	// TODO: get pub ipv6 prefix
	hx := strings.Replace(mac, ":", "", -1)
	hw, err := hex.DecodeString(hx)
	if err != nil {
		return "", errors.New("Could not parse MAC address " + mac)
	}

	hw[0] ^= 0x2

	base[8] = hw[0]
	base[9] = hw[1]
	base[10] = hw[2]
	base[11] = 0xFF
	base[12] = 0xFE
	base[13] = hw[3]
	base[14] = hw[4]
	base[15] = hw[5]

	return base.String(), nil

}
