package gridtypes

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Name is a type for reservation names
type Name string

// IsEmpty indicates if name is not set
func (n Name) IsEmpty() bool {
	return len(string(n)) == 0
}

// Unit defines a capacity unit in "bytes"
// Any "value" of type Unit must be in bytes only
// hence use the Unit mutliplies below to set the
// write value
type Unit uint64

const (
	// Kilobyte unit multiplier
	Kilobyte Unit = 1024
	// Megabyte unit multiplier
	Megabyte Unit = 1024 * Kilobyte
	// Gigabyte unit multiplier
	Gigabyte Unit = 1024 * Megabyte
	// Terabyte unit multiplier
	Terabyte Unit = 1024 * Gigabyte
)

// Max return max of u, and v
func Max(u, v Unit) Unit {
	if u > v {
		return u
	}

	return v
}

// Min return min of u, and v
func Min(u, v Unit) Unit {
	if u < v {
		return u
	}

	return v
}

// Timestamp type
type Timestamp int64

// Time gets time from timestamp
func (t *Timestamp) Time() time.Time {
	return time.Unix(int64(*t), 0)
}

// UnmarshalJSON supports multiple formats
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var u int64
	if err := json.Unmarshal(data, &u); err == nil {
		*t = Timestamp(u)
		return nil
	}

	// else we try time
	var v time.Time
	if err := json.Unmarshal(data, &v); err == nil {
		*t = Timestamp(v.Unix())
		return nil
	}

	return fmt.Errorf("unknown timestamp format, expecting a timestamp or an ISO-8601 date")
}

// IPNet type
type IPNet struct{ net.IPNet }

// NewIPNet creates a new IPNet from net.IPNet
func NewIPNet(n net.IPNet) IPNet {
	return IPNet{IPNet: n}
}

// ParseIPNet parse iprange
func ParseIPNet(txt string) (r IPNet, err error) {
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

// MustParseIPNet prases iprange, panics if invalid
func MustParseIPNet(txt string) IPNet {
	r, err := ParseIPNet(txt)
	if err != nil {
		panic(err)
	}
	return r
}

// UnmarshalText loads IPRange from string
func (i *IPNet) UnmarshalText(text []byte) error {
	v, err := ParseIPNet(string(text))
	if err != nil {
		return err
	}

	i.IPNet = v.IPNet
	return nil
}

// MarshalJSON dumps iprange as a string
func (i IPNet) MarshalJSON() ([]byte, error) {
	if len(i.IPNet.IP) == 0 {
		return []byte(`""`), nil
	}
	v := fmt.Sprint("\"", i.String(), "\"")
	return []byte(v), nil
}

func (i IPNet) String() string {
	return i.IPNet.String()
}

// Nil returns true if IPNet is not set
func (i *IPNet) Nil() bool {
	return i.IP == nil && i.Mask == nil
}
