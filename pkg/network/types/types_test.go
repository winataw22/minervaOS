package types

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/schema"
)

func TestParseIPNet(t *testing.T) {
	parser := func(t *testing.T, in string) IPNet {
		//note in is surrounded by "" because it's json
		var str string
		if err := json.Unmarshal([]byte(in), &str); err != nil {
			t.Fatal(err)
		}

		if len(str) == 0 {
			return IPNet{}
		}

		ip, ipNet, err := net.ParseCIDR(str)
		if err != nil {
			t.Fatal(err)
		}
		ipNet.IP = ip
		return IPNet{*ipNet}
	}

	cases := []struct {
		Input  string
		Output func(*testing.T, string) IPNet
	}{
		{`"192.168.1.0/24"`, parser},
		{`"2001:db8::/32"`, parser},
		{`""`, parser},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			var d IPNet
			err := json.Unmarshal([]byte(c.Input), &d)
			if ok := assert.NoError(t, err); !ok {
				t.Fatal()
			}

			if ok := assert.Equal(t, c.Output(t, c.Input), d); !ok {
				t.Error()
			}
		})
	}
}

func TestDumpIPNet(t *testing.T) {
	mustParse := func(in string) IPNet {
		_, ipNet, err := net.ParseCIDR(in)
		if err != nil {
			panic(err)
		}
		return IPNet{*ipNet}
	}

	cases := []struct {
		Input  IPNet
		Output string
	}{
		{IPNet{}, `""`},
		{mustParse("192.168.1.0/24"), `"192.168.1.0/24"`},
		{mustParse("2001:db8::/32"), `"2001:db8::/32"`},
	}

	for _, c := range cases {
		t.Run(c.Output, func(t *testing.T) {
			out, err := json.Marshal(c.Input)
			if ok := assert.NoError(t, err); !ok {
				t.Fatal()
			}

			if ok := assert.Equal(t, c.Output, string(out)); !ok {
				t.Error()
			}
		})
	}
}

func TestNewNodeFromSchema(t *testing.T) {
	type args struct {
		node directory.TfgridNode2
	}
	tests := []struct {
		name string
		args args
		want *Node
	}{
		{
			name: "full",
			args: args{
				node: directory.TfgridNode2{
					NodeID: "node_id",
					FarmID: 1,
					Ifaces: []directory.TfgridNodeIface1{
						{
							Name: "eth0",
							Addrs: []schema.IPRange{
								schema.MustParseIPRange("192.168.0.10/24"),
							},
							Gateway: []net.IP{
								net.ParseIP("192.168.0.1"),
							},
						},
					},
					PublicConfig: &directory.TfgridNodePublicIface1{
						Master: "eth1",
						Type:   directory.TfgridNodePublicIface1TypeMacvlan,
						Ipv4:   schema.MustParseIPRange("185.69.166.245/24"),
						Gw4:    net.ParseIP("185.69.166.1"),
						Ipv6:   schema.MustParseIPRange("2a02:1802:5e:0:1000:0:ff:1/64"),
						Gw6:    net.ParseIP("2a02:1802:5e::1"),
					},
					WGPorts: []uint{1, 2, 3},
				},
			},
			want: &Node{
				NodeID: "node_id",
				FarmID: 1,
				Ifaces: []*IfaceInfo{
					{
						Name: "eth0",
						Addrs: []IPNet{
							{
								net.IPNet{
									IP:   net.ParseIP("192.168.0.10"),
									Mask: net.CIDRMask(24, 32),
								},
							},
						},
						Gateway: []net.IP{
							net.ParseIP("192.168.0.1"),
						},
					},
				},
				PublicConfig: &PubIface{
					Master: "eth1",
					Type:   MacVlanIface,
					IPv4:   MustParseIPNet("185.69.166.245/24"),
					GW4:    net.ParseIP("185.69.166.1"),
					IPv6:   MustParseIPNet("2a02:1802:5e:0:1000:0:ff:1/64"),
					GW6:    net.ParseIP("2a02:1802:5e::1"),
				},
				WGPorts: []uint{1, 2, 3},
			},
		},
		{
			name: "no-public",
			args: args{
				node: directory.TfgridNode2{
					NodeID: "node_id",
					FarmID: 1,
					Ifaces: []directory.TfgridNodeIface1{
						{
							Name: "eth0",
							Addrs: []schema.IPRange{
								schema.MustParseIPRange("192.168.0.10/24"),
							},
							Gateway: []net.IP{
								net.ParseIP("192.168.0.1"),
							},
						},
					},
					PublicConfig: nil,
					WGPorts:      []uint{1, 2, 3},
				},
			},
			want: &Node{
				NodeID: "node_id",
				FarmID: 1,
				Ifaces: []*IfaceInfo{
					{
						Name: "eth0",
						Addrs: []IPNet{
							{
								net.IPNet{
									IP:   net.ParseIP("192.168.0.10"),
									Mask: net.CIDRMask(24, 32),
								},
							},
						},
						Gateway: []net.IP{
							net.ParseIP("192.168.0.1"),
						},
					},
				},
				PublicConfig: nil,
				WGPorts:      []uint{1, 2, 3},
			},
		},
		{
			name: "empty-ifaces",
			args: args{
				node: directory.TfgridNode2{
					NodeID: "node_id",
					FarmID: 1,
					Ifaces: []directory.TfgridNodeIface1{},
					PublicConfig: &directory.TfgridNodePublicIface1{
						Master: "eth1",
						Type:   directory.TfgridNodePublicIface1TypeMacvlan,
						Ipv4:   schema.MustParseIPRange("185.69.166.245/24"),
						Gw4:    net.ParseIP("185.69.166.1"),
						Ipv6:   schema.MustParseIPRange("2a02:1802:5e:0:1000:0:ff:1/64"),
						Gw6:    net.ParseIP("2a02:1802:5e::1"),
					},
					WGPorts: []uint{1, 2, 3},
				},
			},
			want: &Node{
				NodeID: "node_id",
				FarmID: 1,
				Ifaces: []*IfaceInfo{},
				PublicConfig: &PubIface{
					Master: "eth1",
					Type:   MacVlanIface,
					IPv4:   MustParseIPNet("185.69.166.245/24"),
					GW4:    net.ParseIP("185.69.166.1"),
					IPv6:   MustParseIPNet("2a02:1802:5e:0:1000:0:ff:1/64"),
					GW6:    net.ParseIP("2a02:1802:5e::1"),
				},
				WGPorts: []uint{1, 2, 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewNodeFromSchema(tt.args.node))
		})
	}
}
