package capacity

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/capacity/smartctl"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
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

func IsSecureBoot() (bool, error) {
	// check if node is booted via efi
	const (
		efivars    = "/sys/firmware/efi/efivars"
		secureBoot = "/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"
	)

	_, err := os.Stat(efivars)
	if os.IsNotExist(err) {
		// not even booted with uefi
		return false, nil
	}

	if !filesystem.IsMountPoint(efivars) {
		if err := syscall.Mount("none", efivars, "efivarfs", 0, ""); err != nil {
			return false, errors.Wrap(err, "failed to mount efivars")
		}

		defer func() {
			if err := syscall.Unmount(efivars, 0); err != nil {
				log.Error().Err(err).Msg("failed to unmount efivars")
			}
		}()
	}

	bytes, err := ioutil.ReadFile(secureBoot)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to read secure boot status")
	}

	if len(bytes) != 5 {
		return false, errors.Wrap(err, "invalid efivar data for secure boot flag")
	}

	return bytes[4] == 1, nil
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
func (r *ResourceOracle) GetHypervisor() (string, error) {
	out, err := exec.Command("virt-what").CombinedOutput()

	if err != nil {
		return "", errors.Wrap(err, "could not detect if VM or not")
	}

	str := strings.TrimSpace(string(out))
	if len(str) == 0 {
		return "", nil
	}

	lines := strings.Fields(str)
	if len(lines) > 0 {
		return lines[0], nil
	}

	return "", nil
}
