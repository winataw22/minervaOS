package types

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
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
