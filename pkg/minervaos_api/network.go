package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (g *ZosAPI) networkListWGPortsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.networkerStub.WireguardPorts(ctx)
}
func (g *ZosAPI) networkPublicConfigGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.networkerStub.GetPublicConfig(ctx)
}
func (g *ZosAPI) networkInterfacesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	results := make(map[string][]net.IP)
	type q struct {
		inf    string
		ns     string
		rename string
	}
	for _, i := range []q{{"zos", "", "zos"}, {"nygg6", "ndmz", "ygg"}} {
		ips, _, err := g.networkerStub.Addrs(ctx, i.inf, i.ns)
		if err != nil {
			return nil, fmt.Errorf("failed to get ips for '%s' interface: %w", i, err)
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
func (g *ZosAPI) networkHasIPv6Handler(ctx context.Context, payload []byte) (interface{}, error) {
	ipData, err := g.networkerStub.GetPublicIPv6Subnet(ctx)
	hasIP := ipData.IP != nil && err == nil
	return hasIP, err
}
func (g *ZosAPI) networkListPublicIPsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.provisionStub.ListPublicIPs(ctx)
}

func (g *ZosAPI) networkListPrivateIPsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		NetworkName gridtypes.Name `json:"network_name"`
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	twin := peer.GetTwinID(ctx)
	return g.provisionStub.ListPrivateIPs(ctx, twin, args.NetworkName)
}
