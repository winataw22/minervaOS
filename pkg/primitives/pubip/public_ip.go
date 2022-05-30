package pubip

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	_ provision.Manager = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.publicIPProvisionImpl(ctx, wl)
}

func (p *Manager) getAssignedPublicIP(ctx context.Context, wl *gridtypes.WorkloadWithID) (ip gridtypes.IPNet, gw net.IP, err error) {
	//Okay, this implementation is tricky but will be simplified once we store the reserved IP
	//on the contract.

	// Okay, so first (and easiest path) is that the Ip was already
	// assigned, hence we can simply use it again. this is usually
	// the case if the node is rerunning the same workload deployment for
	// some reason.
	if !wl.Result.IsNil() && wl.Result.State == gridtypes.StateOk {
		var result zos.PublicIPResult
		if err := wl.Result.Unmarshal(&result); err != nil {
			return ip, gw, errors.Wrap(err, "failed to load public ip result")
		}

		return result.IP, result.Gateway, nil
	}
	// otherwise we do the following:

	// - We need to get the contract and the farm object this node belongs to
	deployment, err := provision.GetDeployment(ctx)
	if err != nil {
		return ip, gw, errors.Wrap(err, "failed to get deployment")
	}
	contract := provision.GetContract(ctx)

	// - now we find out ALL ips belonging to this contract
	reserved := contract.PublicIPs

	// make sure we have enough reserved IPs
	// we process both ipv4 type and ip type to be backward compatible
	ipWorkloads := deployment.ByType(zos.PublicIPv4Type, zos.PublicIPType)
	reservedCount := 0
	for _, wl := range ipWorkloads {
		config, err := p.getPublicIPData(ctx, wl)
		if err != nil {
			return gridtypes.IPNet{}, nil, err
		}
		if config.V4 {
			reservedCount += 1
		}
	}

	if reservedCount > len(reserved) {
		return ip, gw, fmt.Errorf("required %d ips while contract has %d ip reserved", len(ipWorkloads), len(reserved))
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

		used, err := GetPubIPConfig(ipWl)
		if err != nil {
			return ip, gw, err
		}

		usedIPs[used.IP.String()] = struct{}{}
	}

	// otherwise we go over the list of IPs and take the first free one
	for _, reservedIP := range reserved {
		if _, ok := usedIPs[reservedIP.IP]; !ok {
			// free ip. we can just take it
			ip, err = gridtypes.ParseIPNet(reservedIP.IP)
			if err != nil {
				return ip, gw, fmt.Errorf("found a malformed ip address in contract object '%s'", ip.IP)
			}
			gw = net.ParseIP(reservedIP.Gateway)
			if gw == nil {
				return ip, gw, fmt.Errorf("found a malformed gateway address in farm object '%s'", reservedIP.Gateway)
			}

			return ip, gw, nil
		}
	}

	return ip, gw, fmt.Errorf("could not allocate public IP address to workload")
}

func (p *Manager) getPublicIPData(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.PublicIP, err error) {
	switch wl.Type {
	case zos.PublicIPv4Type:
		// backword compatibility with older ipv4 type
		result.V4 = true
	case zos.PublicIPType:
		err = json.Unmarshal(wl.Data, &result)
	default:
		return result, fmt.Errorf("invalid workload type expecting (%s or %s) got '%s'", zos.PublicIPv4Type, zos.PublicIPType, wl.Type)
	}

	return
}

func (p *Manager) publicIPProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.PublicIPResult, err error) {
	config, err := p.getPublicIPData(ctx, wl)
	if err != nil {
		return zos.PublicIPResult{}, err
	}

	tapName := wl.ID.Unique("pub")
	network := stubs.NewNetworkerStub(p.zbus)
	fName := filterName(tapName)

	if network.PubIPFilterExists(ctx, fName) {
		return result, provision.ErrNoActionNeeded
	}

	var ipv6 gridtypes.IPNet
	var ipv4 gridtypes.IPNet
	var gw net.IP

	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(tapName))
	if config.V6 {
		pubIP6Base, err := network.GetPublicIPv6Subnet(ctx)
		if err != nil {
			return result, errors.Wrap(err, "could not look up ipv6 prefix")
		}

		ipv6, err = predictedSlaac(pubIP6Base, mac.String())
		if err != nil {
			return zos.PublicIPResult{}, errors.Wrap(err, "could not compute ipv6 valu")
		}
	}

	if config.V4 {
		ipv4, gw, err = p.getAssignedPublicIP(ctx, wl)
		if err != nil {
			return zos.PublicIPResult{}, err
		}
	}

	result.IP = ipv4
	result.IPv6 = ipv6
	result.Gateway = gw

	ifName := fmt.Sprintf("p-%s", tapName) // TODO: clean this up, needs to come form networkd
	err = network.SetupPubIPFilter(ctx, fName, ifName, ipv4.IP, ipv6.IP, mac.String())

	return
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	// Disconnect the public interface from the network if one exists
	network := stubs.NewNetworkerStub(p.zbus)
	tapName := wl.ID.Unique("pub")
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
func predictedSlaac(base net.IPNet, mac string) (gridtypes.IPNet, error) {
	// TODO: get pub ipv6 prefix
	hx := strings.Replace(mac, ":", "", -1)
	hw, err := hex.DecodeString(hx)
	if err != nil {
		return gridtypes.IPNet{}, errors.New("Could not parse MAC address " + mac)
	}

	hw[0] ^= 0x2

	base.IP[8] = hw[0]
	base.IP[9] = hw[1]
	base.IP[10] = hw[2]
	base.IP[11] = 0xFF
	base.IP[12] = 0xFE
	base.IP[13] = hw[3]
	base.IP[14] = hw[4]
	base.IP[15] = hw[5]

	return gridtypes.IPNet{IPNet: base}, nil

}

// GetPubIPConfig get the public ip, and the gateway from the workload
func GetPubIPConfig(wl *gridtypes.WorkloadWithID) (result zos.PublicIPResult, err error) {
	if wl.Type != zos.PublicIPv4Type && wl.Type != zos.PublicIPType {
		return result, fmt.Errorf("workload for public IP is of wrong type")
	}

	if wl.Result.State != gridtypes.StateOk {
		return result, fmt.Errorf("public ip workload is not okay")
	}

	if err := wl.Result.Unmarshal(&result); err != nil {
		return result, errors.Wrap(err, "failed to load ip result")
	}

	return result, nil
}
