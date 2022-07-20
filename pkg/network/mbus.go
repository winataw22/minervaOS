package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// Network message bus api
type Network struct {
	mgr pkg.Networker
}

// NewNetworkMessageBus creates a new messagebus instance
func NewNetworkMessageBus(router rmb.Router, mgr pkg.Networker) (*Network, error) {
	api := &Network{
		mgr: mgr,
	}

	if err := api.setup(router); err != nil {
		return nil, err
	}

	return api, nil
}

func (n *Network) hasPublicIPv6Handler(ctx context.Context, payload []byte) (interface{}, error) {
	return n.hasPublicIPv6(ctx), nil
}

func (n *Network) setup(router rmb.Router) error {

	// network handlers
	sub := router.Subroute("network")
	sub.WithHandler("list_wg_ports", n.listPorts)
	sub.WithHandler("public_config_get", n.getPublicConfig)
	sub.WithHandler("interfaces", n.listInterfaces)
	sub.WithHandler("has_ipv6", n.hasPublicIPv6Handler)

	admin := sub.Subroute("admin")
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get environment")
	}
	mgr, err := environment.GetSubstrate()
	if err != nil {
		return errors.Wrap(err, "failed to get substrate")
	}
	mw, err := rmb.Authorized(mgr, uint32(env.FarmerID))
	if err != nil {
		return errors.Wrap(err, "failed to initialized admin mw")
	}
	admin.Use(mw)
	admin.WithHandler("interfaces", n.listAllInterfaces)
	admin.WithHandler("set_public_nic", n.setPublicNic)
	admin.WithHandler("get_public_nic", n.getPublicNic)

	return nil
}

func (n *Network) listAllInterfaces(ctx context.Context, _ []byte) (interface{}, error) {
	// list all interfaces on node
	type Interface struct {
		IPs []string `json:"ips"`
		Mac string   `json:"mac"`
	}

	interfaces, err := n.mgr.Interfaces("", "")
	if err != nil {
		return nil, err
	}
	output := make(map[string]Interface)
	for name, inf := range interfaces {
		output[name] = Interface{
			Mac: inf.Mac,
			IPs: func() []string {
				var ips []string
				for _, ip := range inf.IPs {
					ips = append(ips, ip.String())
				}
				return ips
			}(),
		}
	}

	return output, nil
}

func (n *Network) setPublicNic(ctx context.Context, data []byte) (interface{}, error) {
	var iface string
	if err := json.Unmarshal(data, &iface); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting string")
	}

	return nil, n.mgr.SetPublicExitDevice(iface)
}

func (n *Network) getPublicNic(ctx context.Context, _ []byte) (interface{}, error) {
	return n.mgr.GetPublicExitDevice()
}

func (n *Network) listPorts(ctx context.Context, _ []byte) (interface{}, error) {
	return n.mgr.WireguardPorts()
}

func (n *Network) hasPublicIPv6(ctx context.Context) interface{} {
	ipData, err := n.mgr.GetPublicIPv6Subnet()
	return ipData.IP != nil && err == nil
}
func (n *Network) listInterfaces(ctx context.Context, _ []byte) (interface{}, error) {

	results := make(map[string][]net.IP)
	type q struct {
		inf    string
		ns     string
		rename string
	}
	for _, i := range []q{{"zos", "", "zos"}, {"nygg6", "ndmz", "ygg"}} {
		ips, _, err := n.mgr.Addrs(i.inf, i.ns)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get ips for '%s' interface", i)
		}

		results[i.rename] = func() []net.IP {
			list := make([]net.IP, 0, len(ips))
			for _, item := range ips {
				ip := net.IP(item)
				list = append(list, ip)
			}

			return list
		}()
	}

	return results, nil
}

func (n *Network) getPublicConfig(ctx context.Context, _ []byte) (interface{}, error) {
	return n.mgr.GetPublicConfig()
}
