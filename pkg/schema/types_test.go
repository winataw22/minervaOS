package schema

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParseDate(t *testing.T) {
	year := time.Now().Year()
	cases := []struct {
		Input  string
		Output time.Time
	}{
		{`"01/02/03"`, time.Date(2001, time.Month(2), 3, 0, 0, 0, 0, time.UTC)},
		{`"01/01/2019 9pm:10"`, time.Date(2019, time.Month(1), 1, 21, 10, 0, 0, time.UTC)},
		{`"15/12/03 22:00"`, time.Date(2015, time.Month(12), 3, 22, 0, 0, 0, time.UTC)},
		{`"06/28"`, time.Date(year, time.Month(6), 28, 0, 0, 0, 0, time.UTC)},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			var d Date
			err := json.Unmarshal([]byte(c.Input), &d)
			if ok := assert.NoError(t, err); !ok {
				t.Fatal()
			}

			if ok := assert.Equal(t, c.Output, d.Time); !ok {
				t.Error()
			}
		})
	}
}

func TestParseEmail(t *testing.T) {
	cases := []struct {
		Input  string
		Output Email
	}{
		{`"azmy@gmail.com"`, Email("azmy@gmail.com")},
		{`"a@b"`, Email("a@b")},
		{`""`, Email("")},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			var d Email
			err := json.Unmarshal([]byte(c.Input), &d)
			if ok := assert.NoError(t, err); !ok {
				t.Fatal()
			}

			if ok := assert.Equal(t, c.Output, d); !ok {
				t.Error()
			}
		})
	}

	failed := []string{
		`"gmail.com"`,
		`"a"`,
	}

	for _, c := range failed {
		t.Run(c, func(t *testing.T) {
			var d Email
			err := json.Unmarshal([]byte(c), &d)
			if ok := assert.Error(t, err); !ok {
				t.Fatal()
			}

		})
	}

}

func TestNumericBigInt(t *testing.T) {
	const inputValue = "1344719667586153181419716641724567886890850696275767987106294472017884974410332069524504824747437757"
	var n Numeric
	err := n.UnmarshalJSON([]byte(fmt.Sprintf(`"%s"`, inputValue)))
	if err != nil {
		t.Fatal(err)
	}
	bi, err := n.BigInt()
	if err != nil {
		t.Fatal(err)
	}
	output := bi.String()
	if inputValue != output {
		t.Fatalf("%s != %s", inputValue, output)
	}
}

func TestDateNil(t *testing.T) {
	var d Date

	b, err := json.Marshal(d)
	require.NoError(t, err)

	assert.Equal(t, []byte(`0`), b)

	d = Date{Time: time.Unix(500, 0)}
	b, err = json.Marshal(d)
	require.NoError(t, err)

	assert.Equal(t, []byte(`500`), b)
}

func TestParseIPRange(t *testing.T) {
	parser := func(t *testing.T, in string) IPRange {
		//note in is surrounded by "" because it's json
		var str string
		if err := json.Unmarshal([]byte(in), &str); err != nil {
			t.Fatal(err)
		}

		if len(str) == 0 {
			return IPRange{}
		}

		ip, ipNet, err := net.ParseCIDR(str)
		if err != nil {
			t.Fatal(err)
		}
		ipNet.IP = ip
		return IPRange{*ipNet}
	}

	cases := []struct {
		Input  string
		Output func(*testing.T, string) IPRange
	}{
		{`"192.168.1.0/24"`, parser},
		{`"2001:db8::/32"`, parser},
		{`""`, parser},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			var d IPRange
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

func TestDumpIPRange(t *testing.T) {
	mustParse := func(in string) IPRange {
		_, ipNet, err := net.ParseCIDR(in)
		if err != nil {
			panic(err)
		}
		return IPRange{*ipNet}
	}

	cases := []struct {
		Input  IPRange
		Output string
	}{
		{IPRange{}, `""`},
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

func TestParseMacAddress(t *testing.T) {
	parser := func(t *testing.T, in string) MacAddress {
		//note in is surrounded by "" because it's json
		var str string
		if err := json.Unmarshal([]byte(in), &str); err != nil {
			t.Fatal(err)
		}

		if len(str) == 0 {
			return MacAddress{}
		}

		mac, err := net.ParseMAC(str)
		if err != nil {
			t.Fatal(err)
		}
		return MacAddress{mac}
	}

	cases := []struct {
		Input  string
		Output func(*testing.T, string) MacAddress
	}{
		{`"54:45:46:f6:02:61"`, parser},
		{`"FF:FF:FF:FF:FF:FF"`, parser},
		{`""`, parser},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			var d MacAddress
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

func TestDumpMacaddress(t *testing.T) {
	mustParse := func(in string) MacAddress {
		mac, err := net.ParseMAC(in)
		if err != nil {
			require.NoError(t, err)
		}
		return MacAddress{mac}
	}

	cases := []struct {
		Input  MacAddress
		Output string
	}{
		{MacAddress{}, `""`},
		{mustParse("54:45:46:f6:02:61"), `"54:45:46:f6:02:61"`},
		{mustParse("FF:FF:FF:FF:FF:FF"), `"ff:ff:ff:ff:ff:ff"`},
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
