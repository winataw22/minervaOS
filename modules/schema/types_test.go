package schema

import (
	"encoding/json"
	"testing"
	"time"

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
