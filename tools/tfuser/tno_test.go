package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/threefoldtech/zos/pkg/network/types"
)

func Test_selectPublicIP(t *testing.T) {
	type args struct {
		node *types.Node
	}
	tests := []struct {
		name    string
		args    args
		want    net.IP
		wantErr bool
	}{
		{
			name: "has public config",
			args: args{
				node: &types.Node{
					PublicConfig: &types.PubIface{
						IPv6: &net.IPNet{
							IP:   net.ParseIP("a02:1802:5e:0:ec4:7aff:fe30:82f9"),
							Mask: net.CIDRMask(64, 128),
						},
					},
				},
			},
			want:    net.ParseIP("a02:1802:5e:0:ec4:7aff:fe30:82f9"),
			wantErr: false,
		},
		{
			name: "has public IP",
			args: args{
				node: &types.Node{
					Ifaces: []*types.IfaceInfo{
						{
							Addrs: []*net.IPNet{
								{
									IP:   net.ParseIP("a02:1802:5e:0:ec4:7aff:fe30:82f9"),
									Mask: net.CIDRMask(64, 128),
								},
							},
						},
					},
				},
			},
			want:    net.ParseIP("a02:1802:5e:0:ec4:7aff:fe30:82f9"),
			wantErr: false,
		},
		{
			name: "has link local",
			args: args{
				node: &types.Node{
					Ifaces: []*types.IfaceInfo{
						{
							Addrs: []*net.IPNet{
								{
									IP:   net.ParseIP("fe80::ec4:7aff:fe30:82f9"),
									Mask: net.CIDRMask(64, 128),
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no public IP",
			args: args{
				node: &types.Node{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectPublicIP(tt.args.node)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
