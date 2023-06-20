package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const (
	sysDeviceBase = "/sys/bus/pci/devices"
	vfioPCIModue  = "vfio-pci"
)

var (
	modules = []string{"vfio", "vfio-pci", "vfio_iommu_type1"}
)

func (m *Manager) initGPUVfioModules() error {
	for _, mod := range modules {
		if err := exec.Command("modprobe", mod).Run(); err != nil {
			return errors.Wrapf(err, "failed to probe module: %s", mod)
		}
	}

	// also set unsafe interrupts
	if err := os.WriteFile("/sys/module/vfio_iommu_type1/parameters/allow_unsafe_interrupts", []byte{'1'}, 0644); err != nil {
		return errors.Wrapf(err, "failed to set allow_unsafe_interrupts for vfio")
	}

	return nil
}

// unbindBootVga is a helper method to disconnect the boot vga if needed
func (m *Manager) unbindBootVga() error {
	const vtConsole = "/sys/class/vtconsole"
	vts, err := os.ReadDir(vtConsole)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to list VTs")
	}
	for _, vt := range vts {
		if err := os.WriteFile(filepath.Join(vtConsole, vt.Name(), "bind"), []byte("0"), 0644); err != nil {
			// log or return ?
			return errors.Wrapf(err, "failed to unbind vt '%s'", vt.Name())
		}
	}

	if err := os.WriteFile("/sys/bus/platform/drivers/efi-framebuffer/unbind", []byte("efi-framebuffer.0"), 0644); err != nil {
		log.Warn().Err(err).Msg("failed to disable frame-buffer")
	}

	return nil
}

// this function will make sure ALL gpus are bind to the right driver
func (m *Manager) initGPUs() error {
	if err := m.initGPUVfioModules(); err != nil {
		return err
	}

	gpus, err := capacity.ListPCI(capacity.GPU)
	if err != nil {
		return errors.Wrap(err, "failed to list system GPUs")
	}

	for _, gpu := range gpus {
		bootVga, err := gpu.Flag("boot_vga")
		if err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to read GPU '%s' boot_vga flag", gpu.Slot)
		}

		if bootVga > 0 {
			if err := m.unbindBootVga(); err != nil {
				log.Warn().Err(err).Msg("error while unbinding boot vga")
			}
		}

		devices, err := capacity.IoMMUGroup(gpu, capacity.Not(capacity.PCIBridge))
		if err != nil {
			return errors.Wrapf(err, "failed to list devices in iommu group for '%s'", gpu.Slot)
		}

		for _, pci := range devices {
			device := filepath.Join(sysDeviceBase, pci.Slot)
			driver := filepath.Join(device, "driver")
			ln, err := os.Readlink(driver)
			if err != nil && !os.IsNotExist(err) {
				return errors.Wrap(err, "failed to check device driver")
			}

			driverName := filepath.Base(ln)
			//note: Base return `.` if path is empty string
			if driverName == vfioPCIModue {
				// correct driver is bind to the device
				continue
			} else if driverName != "." {
				// another driver is bind to this device!
				// this should not happen but we need to be sure
				// let's unbind

				if err := os.WriteFile(filepath.Join(driver, "unbind"), []byte(pci.Slot), 0600); err != nil {
					return errors.Wrapf(err, "failed to unbind gpu '%s' from driver '%s'", pci.ShortID(), driverName)
				}
			}

			// we then need to do an override
			if err := os.WriteFile(filepath.Join(device, "driver_override"), []byte(vfioPCIModue), 0644); err != nil {
				return errors.Wrapf(err, "failed to override the device '%s' driver", pci.Slot)
			}

			if err := os.WriteFile("/sys/bus/pci/drivers_probe", []byte(pci.Slot), 0200); err != nil {
				return errors.Wrapf(err, "failed to bind device '%s' to vfio", pci.Slot)
			}
		}
	}

	return nil
}

// expandGPUs expands the set of provided GPUs with all devices in the IoMMU group.
// It's required that all devices in an iommu group to be passed together to a VM
// hence we need that for each GPU in the list add all the devices from each device
// IOMMU group
func (m *Manager) expandGPUs(gpus []zos.GPU) ([]capacity.PCI, error) {
	all, err := capacity.ListPCI(capacity.GPU)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list available GPUs")
	}

	allMap := make(map[string]capacity.PCI)
	for _, device := range all {
		allMap[device.ShortID()] = device
	}

	var devices []capacity.PCI
	for _, gpu := range gpus {
		device, ok := allMap[string(gpu)]
		if !ok {
			return nil, fmt.Errorf("unknown GPU id '%s'", gpu)
		}

		sub, err := capacity.IoMMUGroup(device, capacity.Not(capacity.PCIBridge))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list all devices belonging to '%s'", device.Slot)
		}

		devices = append(devices, sub...)
	}

	return devices, nil
}
