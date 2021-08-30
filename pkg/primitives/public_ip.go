package primitives

import (
	"context"
	"encoding/hex"
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

func (p *Primitives) getAssignedPublicIP(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.PublicIPResult, err error) {
	//Okay, this implementation is tricky but will be simplified once we store the reserved IP
	//on the contract.

	// Okay, so first (and easiest path) is that the Ip was already
	// assigned, hence we can simply use it again. this is usually
	// the case if the node is rerunning the same workload deployment for
	// some reason.
	if !wl.Result.IsNil() && wl.Result.State == gridtypes.StateOk {
		if err := wl.Result.Unmarshal(&result); err != nil {
			return result, errors.Wrap(err, "failed to load public ip result")
		}

		return result, nil
	}
	// otherwise we do the following:

	// - We need to get the contract and the farm object this node belongs to
	deployment := provision.GetDeployment(ctx)
	contract := provision.GetContract(ctx)

	// - now we find out ALL ips belonging to this contract
	reserved := contract.PublicIPs

	// make sure we have enough reserved IPs
	ipWorkloads := deployment.ByType(zos.PublicIPType)
	if len(ipWorkloads) > len(reserved) {
		return result, fmt.Errorf("required %d ips while contract has %d ip reserved", len(ipWorkloads), len(reserved))
	}

	usedIPs := make(map[string]struct{})

	for _, ipWl := range ipWorkloads {
		if wl.Name == ipWl.Name {
			// we don't need this.
			continue
		}

		if ipWl.Result.IsNil() || ipWl.Result.State != gridtypes.StateOk {
			continue
		}
		var used zos.PublicIPResult
		if err := ipWl.Result.Unmarshal(&used); err != nil {
			return result, errors.Wrap(err, "failed to ")
		}

		usedIPs[used.IP.String()] = struct{}{}
	}

	// otherwise we go over the list of IPs and take the first free one
	for _, ip := range reserved {
		if _, ok := usedIPs[ip.IP]; !ok {
			// free ip. we can just take it
			ipNet, err := gridtypes.ParseIPNet(ip.IP)
			if err != nil {
				return result, fmt.Errorf("found a mullformed ip address in contract object '%s'", ip.IP)
			}
			gw := net.ParseIP(ip.Gateway)
			if gw == nil {
				return result, fmt.Errorf("found a mullformed gateway address in farm object '%s'", ip.Gateway)
			}

			return zos.PublicIPResult{IP: ipNet, Gateway: gw}, nil
		}
	}

	return result, fmt.Errorf("could not allocate public IP address to workload")
}

func (p *Primitives) publicIPProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.PublicIPResult, err error) {
	tapName := tapNameFromName(wl.ID, "pub")
	network := stubs.NewNetworkerStub(p.zbus)
	fName := filterName(tapName)

	if network.PubIPFilterExists(ctx, fName) {
		return result, provision.ErrDidNotChange
	}

	pubIP6Base, err := network.GetPublicIPv6Subnet(ctx)
	if err != nil {
		return zos.PublicIPResult{}, errors.Wrap(err, "could not look up ipv6 prefix")
	}
	ifName := fmt.Sprintf("p-%s", tapName) // TODO: clean this up, needs to come form networkd
	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(tapName))

	predictedIPv6, err := predictedSlaac(pubIP6Base.IP, mac.String())
	if err != nil {
		return zos.PublicIPResult{}, errors.Wrap(err, "could not look up ipv6 prefix")
	}

	result, err = p.getAssignedPublicIP(ctx, wl)
	if err != nil {
		return zos.PublicIPResult{}, err
	}

	err = network.SetupPubIPFilter(ctx, fName, ifName, result.IP.IP.String(), predictedIPv6, mac.String())
	return
}

func (p *Primitives) publicIPDecomission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	// Disconnect the public interface from the network if one exists
	network := stubs.NewNetworkerStub(p.zbus)
	tapName := tapNameFromName(wl.ID, "pub")
	fName := filterName(tapName)
	if err := network.RemovePubIPFilter(ctx, fName); err != nil {
		log.Error().Err(err).Msg("could not remove filter rules")
	}
	return network.DisconnectPubTap(ctx, tapName)
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
