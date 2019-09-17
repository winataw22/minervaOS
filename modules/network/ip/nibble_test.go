package ip

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

func TestNewNibble(t *testing.T) {
	type args struct {
		prefix  *net.IPNet
		allocNr int8
	}
	tests := []struct {
		name    string
		args    args
		want    *Nibble
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/64"),
				allocNr: 1,
			},
			want: &Nibble{
				nibble:  []byte{0xff, 0x02},
				allocNr: 1,
			},
			wantErr: false,
		},
		{
			name: "wrong-size-smaller",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/44"),
				allocNr: 1,
			},
			wantErr: true,
		},
		{
			name: "wrong-size-bigger",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/72"),
				allocNr: 1,
			},
			wantErr: true,
		},
		{
			name: "prefix nil",
			args: args{
				prefix:  nil,
				allocNr: 1,
			},
			wantErr: true,
		},
		{
			name: "alloc negative",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/64"),
				allocNr: -1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNibble(tt.args.prefix, tt.args.allocNr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestHex(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.Hex()
	assert.Equal(t, "ff02", actual)
}
func TestWGName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGName()
	assert.Equal(t, "wg-ff02-0", actual)
}
func TestWireguardPort(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WireguardPort()
	assert.Equal(t, uint16(65282), actual)
}
func TestBridgeName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.BridgeName()
	assert.Equal(t, "br-ff02-0", actual)
}
func TestNamespaceName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.NamespaceName()
	assert.Equal(t, "net-ff02-0", actual)
}

func TestNRLocalName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.NRLocalName()
	assert.Equal(t, "veth-ff02-0", actual)
}
func TestEPPubName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.EPPubName()
	assert.Equal(t, "pub-ff02-0", actual)
}
func TestEPPubLL(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.EPPubLL()
	assert.Equal(t, "fe80::ff02:1/64", actual.String())
}

func TestNRLocalIP4(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:fe02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.NRLocalIP4()
	assert.Equal(t, "10.254.2.1/24", actual.String())
}
func TestWGAllowedIP4(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:fe02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGAllowedIP4()
	assert.Equal(t, "10.255.254.2/16", actual.String())
}

func TestWGAllowedIP6(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:fe02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGAllowedIP6()
	assert.Equal(t, "fe80::fe02/128", actual.String())
}

func TestWGLL(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGLL()
	assert.Equal(t, "fe80::ff02/64", actual.String())
}

func TestRouteIPv6Exit(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv6Exit()
	assert.Equal(t, &netlink.Route{
		Dst: mustParseCIDR("::/0"),
		Gw:  net.ParseIP(fmt.Sprintf("fe80::ff02")),
	}, actual)
}
func TestRouteIPv4Exit(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv4Exit()
	assert.Equal(t, &netlink.Route{
		Dst: mustParseCIDR("10.255.2.0/24"),
		Gw:  net.ParseIP(fmt.Sprintf("10.255.255.02")),
	}, actual)
}
func TestRouteIPv4DefaultExit(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv4DefaultExit()
	assert.Equal(t, &netlink.Route{
		Dst: mustParseCIDR("0.0.0.0/0"),
		Gw:  net.ParseIP(fmt.Sprintf("10.255.255.02")),
	}, actual)
}

func TestRouteIPv4DefaultContainer(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv4DefaultContainer()
	assert.Equal(t, &netlink.Route{
		Dst: mustParseCIDR("0.0.0.0/0"),
		Gw:  net.ParseIP(fmt.Sprintf("10.255.2.1")),
	}, actual)
}

func TestEPToGWName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.EPToGWName()
	assert.Equal(t, "to-ff02-0", actual)
}

func TestGWPubName(t *testing.T) {
	actual := GWPubName(1, 0)
	assert.Equal(t, "pub-1-0", actual)
}

func TestGWtoEPName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWtoEPName()
	assert.Equal(t, "to-ff02-0", actual)
}

func TestGWtoEPLL(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWtoEPLL()
	assert.Equal(t, mustParseCIDR("fe80::1:ff02/64"), actual)
}

func TestGWPubLL(t *testing.T) {
	actual := GWPubLL(1)
	assert.Equal(t, mustParseCIDR("fe80::1000:0:0:1/64").IP, actual.IP)
	actual = GWPubLL(15)
	assert.Equal(t, mustParseCIDR("fe80::f000:0:0:1/64").IP, actual.IP)
}

func TestGWPubIP6(t *testing.T) {
	prefixZero := net.ParseIP("2a02:1802:5e::")
	actual := GWPubIP6(prefixZero, 1)
	assert.Equal(t, mustParseCIDR("2a02:1802:5e:0:1000::1/64").IP, actual.IP)
}

func TestExitNodeRange(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/64")

	rand.Seed(1)
	rnd := uint16(rand.Int63n(4096))

	actual := ExitNodeRange(prefix, 1, rnd)
	assert.Equal(t, mustParseCIDR("2a02:1802:5e:1d52::/64"), actual)

	actual = ExitNodeRange(prefix, 15, rnd)
	assert.Equal(t, mustParseCIDR("2a02:1802:5e:fd52::/64"), actual)
}

func mustParseCIDR(cidr string) *net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return ipnet
}
