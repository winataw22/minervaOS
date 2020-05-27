package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
)

// DeviceManager is able to list all/specific devices on a system
type DeviceManager interface {
	// Device returns the device at the specified path
	Device(ctx context.Context, device string) (*Device, error)
	// Devices finds all devices on a system
	Devices(ctx context.Context) (DeviceCache, error)
	// ByLabel finds all devices with the specified label
	ByLabel(ctx context.Context, label string) ([]*Device, error)
	// Raw returns the devices as represented in the kernel
	// without using the internal cache, nor flattened
	Raw(ctx context.Context) (DeviceCache, error)
	// Reset returns a "clean" instace of the device manager
	// The implementation must take care to clean any caching
	// or other in-memory states and starting fresh. It's called
	// by some routines to make sure a listing of devices will
	// always return the real state of the system.
	Reset() DeviceManager
}

// DeviceCache represents a list of cached in memory devices
type DeviceCache []Device

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

// Device represents a physical device
type Device struct {
	Type       string         `json:"type"`
	Path       string         `json:"name"`
	Label      string         `json:"label"`
	Filesystem FSType         `json:"fstype"`
	Children   []Device       `json:"children"`
	DiskType   pkg.DeviceType `json:"-"`
	ReadTime   uint64         `json:"-"`
	//HasPartions is different from children, because once the
	//devices are flattend in the device, cache, the children list is
	//zeroed (since all devices are flat), then has partions is set to
	//make sure the device is not altered.
	HasPartions bool `json:"-"`
}

// Used assumes that the device is used if it has custom label or fstype or children
func (d *Device) Used() bool {
	return len(d.Label) != 0 || len(d.Filesystem) != 0 || len(d.Children) > 0 || d.HasPartions
}

// lsblkDeviceManager uses the lsblk utility to scann the disk for devices, and
// caches the result.
//
// Found devices are cached, and the cache is only repopulated after the `Scan`
// method is called.
type lsblkDeviceManager struct {
	executer
	cache DeviceCache
}

// DefaultDeviceManager returns a default device manager implementation
func DefaultDeviceManager(ctx context.Context) DeviceManager {
	return defaultDeviceManager(ctx, executerFunc(run))
}

func defaultDeviceManager(ctx context.Context, exec executer) DeviceManager {
	m := &lsblkDeviceManager{
		executer: exec,
	}

	return m
}

func (l *lsblkDeviceManager) Reset() DeviceManager {
	return &lsblkDeviceManager{executer: l.executer}
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) (DeviceCache, error) {
	devices, err := l.scan(ctx)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

func (l *lsblkDeviceManager) ByLabel(ctx context.Context, label string) ([]*Device, error) {
	devices, err := l.scan(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*Device

	for idx := range devices {
		device := &devices[idx]
		if device.Label == label {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

func (l *lsblkDeviceManager) Device(ctx context.Context, path string) (device *Device, err error) {
	devices, err := l.scan(ctx)
	if err != nil {
		return nil, err
	}

	for idx := range devices {
		if devices[idx].Path == path {
			return &devices[idx], nil
		}
	}

	return nil, fmt.Errorf("device not found")

}

func (l *lsblkDeviceManager) Raw(ctx context.Context) (DeviceCache, error) {
	bytes, err := l.run(ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2,11", "--path")
	if err != nil {
		return nil, err
	}

	var devices struct {
		BlockDevices []Device `json:"blockdevices"`
	}

	// skipping unmarshal when lsblk response is empty
	if len(bytes) == 0 {
		log.Warn().Msg("no disks found on the system")
		return DeviceCache(devices.BlockDevices), nil
	}

	// parsing lsblk response
	if err := json.Unmarshal(bytes, &devices); err != nil {
		return nil, err
	}

	return DeviceCache(devices.BlockDevices), nil
}

// scan the system for disks using the `lsblk` command
func (l *lsblkDeviceManager) scan(ctx context.Context) (DeviceCache, error) {
	if l.cache != nil {
		return l.cache, nil
	}

	devs, err := l.Raw(ctx)
	if err != nil {
		errors.Wrap(err, "failed to scan devices")
	}

	if err := l.setDeviceTypes(devs); err != nil {
		return nil, err
	}

	l.cache = l.flattenDevices(devs)
	return l.cache, nil
}

// seektime uses the seektime binary to try and determine the type of a disk
// This function returns the type of the device, as reported by seektime,
// and the elapsed time in microseconds (also reported by seektime)
func (l *lsblkDeviceManager) seektime(ctx context.Context, path string) (string, uint64, error) {
	bytes, err := l.run(ctx, "seektime", "-j", path)
	if err != nil {
		return "", 0, err
	}

	var seekTime struct {
		Typ  string `json:"type"`
		Time uint64 `json:"elapsed"`
	}

	err = json.Unmarshal(bytes, &seekTime)
	log.Debug().Str("disk", path).Str("type", seekTime.Typ).Uint64("time", seekTime.Time).Msg("seektime")

	return seekTime.Typ, seekTime.Time, err
}

func (l *lsblkDeviceManager) flattenDevices(devices DeviceCache) DeviceCache {
	var list DeviceCache
	for _, device := range devices {
		children := device.Children
		device.Children = nil
		if len(children) > 0 {
			device.HasPartions = true
		}
		list = append(list, device)
		list = append(list, l.flattenDevices(children)...)
	}

	return list
}

func (l *lsblkDeviceManager) setDeviceTypes(devices []Device) error {
	for idx := range devices {
		d := &devices[idx]
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		typ, rt, err := l.seektime(ctx, d.Path)
		if err != nil {
			// don't include errored devices in the result
			log.Error().Msgf("Failed to get disk read time: %v", err)
			return err
		}

		l.setDeviceType(d, l.deviceTypeFromString(typ), rt)
	}

	return nil
}

// setDeviceType recursively sets a device type and read time on a device and
// all of its children
func (l *lsblkDeviceManager) setDeviceType(device *Device, typ pkg.DeviceType, readTime uint64) {
	device.DiskType = typ
	device.ReadTime = readTime

	for idx := range device.Children {
		dev := &device.Children[idx]
		l.setDeviceType(dev, typ, readTime)
	}
}

func (l *lsblkDeviceManager) deviceTypeFromString(typ string) pkg.DeviceType {
	switch strings.ToLower(typ) {
	case string(pkg.SSDDevice):
		return pkg.SSDDevice
	case string(pkg.HDDDevice):
		return pkg.HDDDevice
	default:
		// if we have an error or unrecognized type, set type to HDD
		return pkg.HDDDevice
	}
}

// ByReadTime implements sort.Interface for []Device based on the ReadTime field
type ByReadTime DeviceCache

func (a ByReadTime) Len() int           { return len(a) }
func (a ByReadTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByReadTime) Less(i, j int) bool { return a[i].ReadTime < a[j].ReadTime }
