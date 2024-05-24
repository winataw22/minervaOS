package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
)

func (g *ZosAPI) adminInterfacesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	// list all interfaces on node
	type Interface struct {
		IPs []string `json:"ips"`
		Mac string   `json:"mac"`
	}

	interfaces, err := g.networkerStub.Interfaces(ctx, "", "")
	if err != nil {
		return nil, err
	}
	output := make(map[string]Interface)
	for name, inf := range interfaces.Interfaces {
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

func (g *ZosAPI) adminGetPublicNICHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.networkerStub.GetPublicExitDevice(ctx)
}

func (g *ZosAPI) adminSetPublicNICHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var iface string
	if err := json.Unmarshal(payload, &iface); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting string: %w", err)
	}
	return nil, g.networkerStub.SetPublicExitDevice(ctx, iface)
}
