package primitives

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (p *Primitives) publicIPProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.publicIPProvisionImpl(ctx, wl)
}

func (p *Primitives) publicIPProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.PublicIPResult, err error) {
	config := zos.PublicIP{}

	network := stubs.NewNetworkerStub(p.zbus)
	fName := filterName(wl.ID.String())

	if network.PubIPFilterExists(ctx, fName) {
		return result, provision.ErrDidNotChange
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return zos.PublicIPResult{}, errors.Wrap(err, "failed to decode reservation schema")
	}

	pubIP6Base, err := network.GetPublicIPv6Subnet(ctx)
	if err != nil {
		return zos.PublicIPResult{}, errors.Wrap(err, "could not look up ipv6 prefix")
	}

	tapName := fmt.Sprintf("p-%s", wl.ID.String()) // TODO: clean this up, needs to come form networkd
	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(wl.ID.String()))

	predictedIPv6, err := predictedSlaac(pubIP6Base.IP, mac.String())
	if err != nil {
		return zos.PublicIPResult{}, errors.Wrap(err, "could not look up ipv6 prefix")
	}

	result.IP = config.IP
	err = network.SetupPubIPFilter(ctx, fName, tapName, config.IP.IP.To4().String(), predictedIPv6, mac.String())
	return
}

func (p *Primitives) publicIPDecomission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	// Disconnect the public interface from the network if one exists
	network := stubs.NewNetworkerStub(p.zbus)
	fName := filterName(wl.ID.String())
	if err := network.RemovePubIPFilter(ctx, fName); err != nil {
		log.Error().Err(err).Msg("could not remove filter rules")
	}
	return network.DisconnectPubTap(ctx, wl.ID.String())
}

func filterName(reservationID string) string {
	return fmt.Sprintf("r-%s", reservationID)
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
