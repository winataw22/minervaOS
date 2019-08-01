package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
)

// DeviceManager is able to list all/specific devices on a system
type DeviceManager interface {
	// Device returns the device at the specified path
	Device(ctx context.Context, device string) (*Device, error)
	// Devices finds all devices on a system
	Devices(ctx context.Context) (DeviceCache, error)
	// ByLabel finds all devices with the specified label
	ByLabel(ctx context.Context, label string) (DeviceCache, error)
}

// DeviceCache represents a list of cached in memory devices
type DeviceCache []*Device

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

// Device represents a physical device
type Device struct {
	Type       string             `json:"type"`
	Path       string             `json:"name"`
	Label      string             `json:"label"`
	Filesystem FSType             `json:"fstype"`
	Children   []Device           `json:"children"`
	DiskType   modules.DeviceType `json:"-"`
	ReadTime   uint64             `json:"-"`
}

// Used assumes that the device is used if it has custom label or fstype or children
func (d *Device) Used() bool {
	return len(d.Label) != 0 || len(d.Filesystem) != 0 || len(d.Children) > 0
}

// lsblkDeviceManager uses the lsblk utility to scann the disk for devices, and
// caches the result.
//
// Found devices are cached, and the cache is only repopulated after the `Scan`
// method is called.
type lsblkDeviceManager struct {
	devices DeviceCache
}

// DefaultDeviceManager returns a default device manager implementation
func DefaultDeviceManager(ctx context.Context) (DeviceManager, error) {
	m := &lsblkDeviceManager{
		devices: DeviceCache{},
	}

	err := m.scan(ctx)
	return m, err
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) (DeviceCache, error) {
	return l.devices, nil
}

func (l *lsblkDeviceManager) ByLabel(ctx context.Context, label string) (DeviceCache, error) {
	var filtered DeviceCache

	for _, device := range l.devices {
		if device.Label == label {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

func (l *lsblkDeviceManager) Device(ctx context.Context, path string) (device *Device, err error) {
	for idx := range l.devices {
		if l.devices[idx].Path == path {
			return l.devices[idx], nil
		}
	}

	return nil, fmt.Errorf("device not found")

}

// scan the system for disks using the `lsblk` command
func (l *lsblkDeviceManager) scan(ctx context.Context) error {
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

	typedDevs, err := setDeviceTypes(devices.BlockDevices)
	if err != nil {
		return err
	}

	var devs DeviceCache

	for idx := range typedDevs {
		devs = append(devs, &typedDevs[idx])
	}

	l.devices = flattenDevices(devs)

	return nil
}

func flattenDevices(devices DeviceCache) DeviceCache {
	list := DeviceCache{}
	for idx := range devices {
		list = append(list, devices[idx])
		if devices[idx].Children != nil {
			childCache := DeviceCache{}
			for jdx := range devices[idx].Children {
				childCache = append(childCache, &devices[idx].Children[jdx])
			}
			list = append(list, flattenDevices(childCache)...)
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
func setDeviceType(device *Device, typ modules.DeviceType, readTime uint64) {
	device.DiskType = typ
	device.ReadTime = readTime

	for _, dev := range device.Children {
		setDeviceType(&dev, typ, readTime)
	}
}

func deviceTypeFromString(typ string) modules.DeviceType {
	switch typ {
	case string(modules.SSDDevice):
		return modules.SSDDevice
	case string(modules.HDDDevice):
		return modules.HDDDevice
	default:
		// if we have an error or unrecognized type, set type to HDD
		return modules.HDDDevice
	}
}

// ByReadTime implements sort.Interface for []Device based on the ReadTime field
type ByReadTime DeviceCache

func (a ByReadTime) Len() int           { return len(a) }
func (a ByReadTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByReadTime) Less(i, j int) bool { return a[i].ReadTime < a[j].ReadTime }
