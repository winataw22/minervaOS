package schema

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
)

var (
	dateRe  = regexp.MustCompile(`^(?:(\d{2,4})/)?(\d{2})/(\d{2,4})(?:\s+(\d{1,2})(am|pm)?:(\d{1,2}))?$`)
	boolMap = map[string]bool{
		"true":  true,
		"yes":   true,
		"y":     true,
		"1":     true,
		"false": false,
		"no":    false,
		"n":     false,
		"0":     false,
	}
)

// Numeric type. this type is tricky so we just going to handle it as string
// for now.
type Numeric string

// Float64 returns parsed float64 value
func (n Numeric) Float64() (float64, error) {
	return strconv.ParseFloat(string(n), 64)
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
	var in string
	if err := json.Unmarshal(bytes, &in); err != nil {
		return err
	}

	m := dateRe.FindStringSubmatch(in)
	if m == nil {
		return fmt.Errorf("invalid date string %s", in)
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

	fmt.Println(year, month, day, hour, min)

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
	return []byte(d.Format("02/01/2006 15:04")), nil
}

// IPRange type
type IPRange struct{ net.IPNet }

// UnmarshalText loads IPRange from string
func (i *IPRange) UnmarshalText(text []byte) error {
	_, net, err := net.ParseCIDR(string(text))
	if err != nil {
		return err
	}

	i.IPNet = *net
	return nil
}

// MarshalText dumps iprange as a string
func (i *IPRange) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

func (i IPRange) String() string {
	return i.IPNet.String()
}
