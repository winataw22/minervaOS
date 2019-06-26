package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/rs/zerolog/log"
)

// DeviceManager is able to list all/specific devices on a system
type DeviceManager interface {
	// Device returns the device at the specified path
	Device(ctx context.Context, device string) (Device, error)
	// Devices finds all devices on a system
	Devices(ctx context.Context) ([]Device, error)
	// ByLabel finds all devices with the specified label
	ByLabel(ctx context.Context, label string) ([]Device, error)
	// Scan the system for devices. This must be called after all actions which
	// make persistent changes on the disk layout.
	Scan(ctx context.Context) error
}

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice = "SSD"
	HDDDevice = "HDD"
)

// Device represents a physical device
type Device struct {
	Type       string     `json:"type"`
	Path       string     `json:"name"`
	Label      string     `json:"label"`
	Filesystem FSType     `json:"fstype"`
	Children   []Device   `json:"children"`
	DiskType   DeviceType `json:"-"`
	ReadTime   uint64     `json:"-"`
}

// Used assumes that the device is used if it has custom label of fstype
func (d *Device) Used() bool {
	return len(d.Label) != 0 || len(d.Filesystem) != 0 || len(d.Children) > 0
}

// lsblkDeviceManager uses the lsblk utility to scann the disk for devices, and
// caches the result.
//
// Found devices are cached, and the cache is only repopulated after the `Scan`
// method is called.
type lsblkDeviceManager struct {
	devices []Device
}

// DefaultDeviceManager returns a default device manager implementation
func DefaultDeviceManager() DeviceManager {
	return &lsblkDeviceManager{
		devices: []Device{},
	}
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) ([]Device, error) {
	return flattenDevices(l.devices), nil
}

func (l *lsblkDeviceManager) ByLabel(ctx context.Context, label string) ([]Device, error) {
	var filtered []Device

	for _, device := range l.devices {
		if device.Label == label {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

func (l *lsblkDeviceManager) Device(ctx context.Context, path string) (device Device, err error) {
	for _, device := range l.devices {
		if device.Path == path {
			return device, nil
		}
	}

	return Device{}, fmt.Errorf("device not found")

}

func (l *lsblkDeviceManager) Scan(ctx context.Context) error {
	bytes, err := run(ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2,11", "--path")
	if err != nil {
		return err
	}

	var devices struct {
		BlockDevices []Device `json:"blockdevices"`
	}

	if err := json.Unmarshal(bytes, &devices); err != nil {
		return err
	}

	l.devices, err = setDeviceTypes(devices.BlockDevices)
	return err
}

func flattenDevices(devices []Device) []Device {
	list := []Device{}
	for _, d := range devices {
		list = append(list, d)
		if d.Children != nil {
			list = append(list, flattenDevices(d.Children)...)
		}
	}
	return list
}

func setDeviceTypes(devices []Device) ([]Device, error) {
	list := []Device{}
	for _, d := range devices {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		typ, rt, err := seektime(ctx, d.Path)
		if err != nil {
			// don't include errored devices in the result
			log.Error().Msgf("Failed to get disk read time: %v", err)
			return nil, err
		}

		setDeviceType(&d, deviceTypeFromString(typ), rt)

		list = append(list, d)
	}

	return list, nil
}

// setDeviceType recursively sets a device type and read time on a device and
// all of its children
func setDeviceType(device *Device, typ DeviceType, readTime uint64) {
	device.DiskType = typ
	device.ReadTime = readTime

	for _, dev := range device.Children {
		setDeviceType(&dev, typ, readTime)
	}
}

func deviceTypeFromString(typ string) DeviceType {
	switch typ {
	case string(SSDDevice):
		return SSDDevice
	case string(HDDDevice):
		return HDDDevice
	default:
		// if we have an error or unrecognized type, set type to HDD
		return HDDDevice
	}
}

// ByReadTime implements sort.Interface for []Device based on the ReadTime field
type ByReadTime []Device

func (a ByReadTime) Len() int           { return len(a) }
func (a ByReadTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByReadTime) Less(i, j int) bool { return a[i].ReadTime < a[j].ReadTime }
