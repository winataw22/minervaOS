package capacity

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/capacity/smartctl"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Capacity hold the amount of resource unit of a node
type Capacity struct {
	CRU uint64 `json:"cru"`
	MRU uint64 `json:"mru"`
	SRU uint64 `json:"sru"`
	HRU uint64 `json:"hru"`
}

// ResourceOracle is the structure responsible for capacity tracking
type ResourceOracle struct {
	storage *stubs.StorageModuleStub
}

// NewResourceOracle creates a new ResourceOracle
func NewResourceOracle(s *stubs.StorageModuleStub) *ResourceOracle {
	return &ResourceOracle{storage: s}
}

// Total returns the total amount of resource units of the node
func (r *ResourceOracle) Total() (c gridtypes.Capacity, err error) {

	c.CRU, err = r.cru()
	if err != nil {
		return c, err
	}
	c.MRU, err = r.mru()
	if err != nil {
		return c, err
	}
	c.SRU, err = r.sru()
	if err != nil {
		return c, err
	}
	c.HRU, err = r.hru()
	if err != nil {
		return c, err
	}

	return c, nil
}

// DMI run and parse dmidecode commands
func (r *ResourceOracle) DMI() (*dmi.DMI, error) {
	return dmi.Decode()
}

// Uptime returns the uptime of the node
func (r *ResourceOracle) Uptime() (uint64, error) {
	info, err := host.Info()
	if err != nil {
		return 0, err
	}
	return info.Uptime, nil
}

// Disks contains the hardware information about the disk of a node
type Disks struct {
	Tool        string          `json:"tool"`
	Environment string          `json:"environment"`
	Aggregator  string          `json:"aggregator"`
	Devices     []smartctl.Info `json:"devices"`
}

// Disks list and parse the hardware information using smartctl
func (r *ResourceOracle) Disks() (d Disks, err error) {
	devices, err := smartctl.ListDevices()
	if errors.Is(err, smartctl.ErrEmpty) {
		// TODO: for now we allow to not have the smartctl dump of all the disks
		log.Warn().Err(err).Msg("smartctl did not found any disk on the system")
		return d, nil
	}
	if err != nil {
		return
	}

	var info smartctl.Info
	d.Devices = make([]smartctl.Info, len(devices))

	for i, device := range devices {
		info, err = smartctl.DeviceInfo(device)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get device info for: %s", device.Path)
			continue
		}
		d.Devices[i] = info
		if d.Environment == "" {
			d.Environment = info.Environment
		}
		if d.Tool == "" {
			d.Tool = info.Tool
		}
	}

	d.Aggregator = "0-OS smartctl aggregator"

	return
}

//GetHypervisor gets the name of the hypervisor used on the node
func (r *ResourceOracle) GetHypervisor() ([]string, error) {
	out, err := exec.Command("virt-what").CombinedOutput()

	if err != nil {
		return nil, errors.Wrap(err, "could not detect if VM or not")
	}

	str := strings.TrimSpace(string(out))
	if len(str) == 0 {
		return nil, nil
	}

	lines := strings.Split(str, "\n")
	return lines, nil
}
