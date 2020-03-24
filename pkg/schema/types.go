package schema

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	"math/big"
)

var (
	dateRe = regexp.MustCompile(`^(?:(\d{2,4})/)?(\d{2})/(\d{2,4})(?:\s+(\d{1,2})(am|pm)?:(\d{1,2}))?$`)
)

// Numeric type. this type is tricky so we just going to handle it as string
// for now.
type Numeric string

// Float64 returns parsed float64 value
func (n Numeric) Float64() (float64, error) {
	return strconv.ParseFloat(string(n), 64)
}

// BigInt returns parsed big.Int value
func (n Numeric) BigInt() (*big.Int, error) {
	bi := new(big.Int)
	_, ok := bi.SetString(string(n), 10)
	if !ok {
		return nil, fmt.Errorf("failed to return '%s' as a big Int", n)
	}
	return bi, nil
}

// UnmarshalJSON method
func (n *Numeric) UnmarshalJSON(in []byte) error {
	var v interface{}
	if err := json.Unmarshal(in, &v); err != nil {
		return err
	}

	*n = Numeric(fmt.Sprint(v))

	return nil
}

// Date a jumpscale date wrapper
type Date struct{ time.Time }

// UnmarshalJSON method
func (d *Date) UnmarshalJSON(bytes []byte) error {
	var inI interface{}
	if err := json.Unmarshal(bytes, &inI); err != nil {
		return err
	}

	var in string
	switch v := inI.(type) {
	case int64:
		d.Time = time.Unix(v, 0).UTC()
		return nil
	case float64:
		d.Time = time.Unix(int64(v), 0).UTC()
		return nil
	case string:
		in = v
	default:
		return fmt.Errorf("unknown date format: %T(%s)", v, string(bytes))
	}

	if len(in) == 0 {
		//null date
		d.Time = time.Time{}
		return nil
	}

	m := dateRe.FindStringSubmatch(in)
	if m == nil {
		return fmt.Errorf("invalid date string '%s'", in)
	}

	first := m[1]
	month := m[2]
	last := m[3]

	hour := m[4]
	ampm := m[5]
	min := m[6]

	var year string
	var day string

	if first == "" {
		year = fmt.Sprint(time.Now().Year())
		day = last
	} else if len(first) == 4 && len(last) == 4 {
		return fmt.Errorf("invalid date format ambiguous year: %s", in)
	} else if len(last) == 4 {
		year = last
		day = first
	} else {
		// both ar 2 or first is 4 and last is 2
		year = first
		day = last
	}

	if hour == "" {
		hour = "0"
	}
	if min == "" {
		min = "0"
	}

	var values []int
	for _, str := range []string{year, month, day, hour, min} {
		value, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s' in date", str)
		}
		values = append(values, value)
	}

	if values[0] < 100 {
		values[0] += 2000
	}

	if ampm == "pm" {
		values[3] += 12
	}

	d.Time = time.Date(values[0], time.Month(values[1]), values[2], values[3], values[4], 0, 0, time.UTC)

	return nil
}

// MarshalJSON formats a text
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte(`0`), nil
	}
	return []byte(fmt.Sprintf(`%d`, d.Unix())), nil
}

// String implements stringer interface
func (d Date) String() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, d.Format("02/01/2006 15:04"))), nil
}

// IPRange type
type IPRange struct{ net.IPNet }

// ParseIPRange parse iprange
func ParseIPRange(txt string) (r IPRange, err error) {
	if len(txt) == 0 {
		//empty ip net value
		return r, nil
	}
	//fmt.Println("parsing: ", string(text))
	ip, net, err := net.ParseCIDR(txt)
	if err != nil {
		return r, err
	}

	net.IP = ip
	r.IPNet = *net
	return
}

// MustParseIPRange prases iprange, panics if invalid
func MustParseIPRange(txt string) IPRange {
	r, err := ParseIPRange(txt)
	if err != nil {
		panic(err)
	}
	return r
}

// UnmarshalText loads IPRange from string
func (i *IPRange) UnmarshalText(text []byte) error {
	v, err := ParseIPRange(string(text))
	if err != nil {
		return err
	}

	i.IPNet = v.IPNet
	return nil
}

// MarshalJSON dumps iprange as a string
func (i IPRange) MarshalJSON() ([]byte, error) {
	if len(i.IPNet.IP) == 0 {
		return []byte(`""`), nil
	}
	v := fmt.Sprint("\"", i.String(), "\"")
	return []byte(v), nil
}

func (i IPRange) String() string {
	return i.IPNet.String()
}

// Email type
type Email string

var (
	emailP = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// UnmarshalText loads IPRange from string
func (e *Email) UnmarshalText(text []byte) error {
	v := string(text)
	if v == "" {
		return nil
	}

	if !emailP.MatchString(v) {
		return fmt.Errorf("invalid email address: %s", v)
	}

	*e = Email(v)
	return nil
}

// ID object id
type ID int64

// MacAddress type
type MacAddress struct{ net.HardwareAddr }

// MarshalText marshals MacAddress type to a string
func (mac MacAddress) MarshalText() ([]byte, error) {
	if mac.HardwareAddr == nil {
		return nil, nil
	} else if mac.HardwareAddr.String() == "" {
		return nil, nil
	}
	return []byte(mac.HardwareAddr.String()), nil
}

// UnmarshalText loads a macaddress from a string
func (mac *MacAddress) UnmarshalText(addr []byte) error {
	if len(addr) == 0 {
		return nil
	}
	addr, err := net.ParseMAC(string(addr))
	if err != nil {
		return err
	}
	mac.HardwareAddr = addr
	return nil
}
