package zos

import (
	"fmt"
	"io"
	"net/url"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	validLogSchemes = map[string]struct{}{
		"redis": {},
		"ws":    {},
		"wss":   {},
	}
)

type ZLogs struct {
	// ZMachine stream logs for which zmachine
	ZMachine gridtypes.Name `json:"zmachine"`
	// Output url
	Output string `json:"output"`
}

func (z ZLogs) Valid(getter gridtypes.WorkloadGetter) error {
	wl, err := getter.Get(z.ZMachine)
	if err != nil || wl.Type != ZMachineType {
		return fmt.Errorf("no zmachine with name '%s' found", z.ZMachine)
	}

	u, err := url.Parse(z.Output)
	if err != nil {
		return errors.Wrap(err, "invalid url supplied in output")
	}

	if _, ok := validLogSchemes[u.Scheme]; !ok {
		return fmt.Errorf("invalid output schema '%s' not supported", u.Scheme)
	}

	return nil
}

func (z ZLogs) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", z.ZMachine); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", z.Output); err != nil {
		return err
	}

	return nil
}

func (z ZLogs) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{}, nil
}
