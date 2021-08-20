package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Device interface
type Device interface {
	// Path returns path to the device like /dev/sda
	Path() string
	// Name returns name of the device like sda
	Name() string
	// Size device size
	Size() uint64
	// Type returns detected device type (hdd, ssd)
	Type() zos.DeviceType
	// Info is current device information, this should not be cached because
	// it might change over time
	Info() (DeviceInfo, error)
	// ReadTime detected read time of the device
	ReadTime() uint64
}

// DeviceManager is able to list all/specific devices on a system
type DeviceManager interface {
	// Device returns the device at the specified path
	Device(ctx context.Context, device string) (Device, error)
	// Devices finds all devices on a system
	Devices(ctx context.Context) (Devices, error)
	// ByLabel finds all devices with the specified label
	ByLabel(ctx context.Context, label string) (Devices, error)
}

// Devices represents a list of cached in memory devices
type Devices []Device

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

// DeviceInfo contains information about the device
type DeviceInfo struct {
	Path       string `json:"path"`
	Label      string `json:"label"`
	Size       uint64 `json:"size"`
	Mountpoint string `json:"mountpoint"`
	Filesystem FSType `json:"fstype"`
}

// func (i *DeviceInfo) ID() (id string, err error) {
// 	// check if it's a pool using the label format.
// 	// if it has a filesystem then it should have the label in correct format
// 	if _, err = fmt.Sscanf(i.Label, PoolLabelPrefix+"%s", &id); err != nil {
// 		return id, ErrInvalidLabel
// 	}

// 	return
// }

// func (i *DeviceInfo) IsPool() bool {
// 	id, err := i.ID()
// 	if err != nil {
// 		return false
// 	}

// 	return len(id) != 0
// }

// Used assumes that the device is used if it has custom label or fstype or children
func (i *DeviceInfo) Used() bool {
	return len(i.Label) != 0 || len(i.Filesystem) != 0
}

type deviceImpl struct {
	minDevice
	mgr *lsblkDeviceManager
}

func (d *deviceImpl) Info() (DeviceInfo, error) {
	var devices struct {
		BlockDevices []DeviceInfo `json:"blockdevices"`
	}

	if err := d.mgr.lsblk(context.Background(), &devices, d.IPath); err != nil {
		return DeviceInfo{}, err
	}
	if len(devices.BlockDevices) != 1 {
		return DeviceInfo{}, fmt.Errorf("device not found")
	}

	return devices.BlockDevices[0], nil
}

func (d *deviceImpl) Type() zos.DeviceType {
	return d.DiskType
}

func (d *deviceImpl) ReadTime() uint64 {
	return d.RTime
}

type minDevice struct {
	IPath      string         `json:"path"`
	IName      string         `json:"name"`
	ISize      uint64         `json:"size"`
	DiskType   zos.DeviceType `json:"-"`
	RTime      uint64         `json:"-"`
	Subsystems string         `json:"subsystems"`
}

func (m minDevice) toDevice(mgr *lsblkDeviceManager) Device {
	return &deviceImpl{
		minDevice: m,
		mgr:       mgr,
	}
}

func (m *minDevice) Path() string {
	return m.IPath
}

func (m *minDevice) Size() uint64 {
	return m.ISize
}

func (m *minDevice) Name() string {
	return m.IName
}

// lsblkDeviceManager uses the lsblk utility to scann the disk for devices, and
// caches the result.
//
// Found devices are cached, and the cache is only repopulated after the `Scan`
// method is called.
type lsblkDeviceManager struct {
	executer
	cache []minDevice
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

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) (Devices, error) {
	devices, err := l.scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make(Devices, 0, len(devices))
	for _, dev := range devices {
		result = append(result, dev.toDevice(l))
	}

	return result, nil
}

func (l *lsblkDeviceManager) ByLabel(ctx context.Context, label string) (Devices, error) {
	devices, err := l.Devices(ctx)
	if err != nil {
		return nil, err
	}

	var filtered Devices

	for _, device := range devices {
		info, err := device.Info()
		if err != nil {
			return nil, err
		}

		if info.Label == label {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

func (l *lsblkDeviceManager) Device(ctx context.Context, path string) (device Device, err error) {
	devices, err := l.scan(ctx)
	if err != nil {
		return nil, err
	}

	for _, dev := range devices {
		if dev.IPath == path {
			return dev.toDevice(l), nil
		}
	}

	return nil, fmt.Errorf("device not found")

}

func (l *lsblkDeviceManager) lsblk(ctx context.Context, output interface{}, device ...string) error {
	args := []string{
		"--json",
		"--output-all",
		"--bytes",
		"--exclude",
		"1,2,11",
		"--path",
	}

	if len(device) == 1 {
		args = append(args, device[0])
	} else if len(device) > 1 {
		return fmt.Errorf("only one device is supported")
	}

	bytes, err := l.run(ctx, "lsblk", args...)
	if err != nil {
		return err
	}

	// skipping unmarshal when lsblk response is empty
	if len(bytes) == 0 {
		log.Warn().Msg("no disks found on the system")
		return nil
	}

	// parsing lsblk response
	if err := json.Unmarshal(bytes, output); err != nil {
		return err
	}

	return nil
}

func (l *lsblkDeviceManager) raw(ctx context.Context) ([]minDevice, error) {

	var devices struct {
		BlockDevices []minDevice `json:"blockdevices"`
	}

	if err := l.lsblk(ctx, &devices); err != nil {
		return nil, err
	}

	filtered := devices.BlockDevices[:0]
	for _, device := range devices.BlockDevices {
		if device.Subsystems != "block:scsi:usb:pci" {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

// scan the system for disks using the `lsblk` command
func (l *lsblkDeviceManager) scan(ctx context.Context) ([]minDevice, error) {
	if l.cache != nil {
		return l.cache, nil
	}

	devs, err := l.raw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan devices")
	}

	if err := l.setDeviceTypes(devs); err != nil {
		return nil, err
	}

	l.cache = devs
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

func (l *lsblkDeviceManager) setDeviceTypes(devices []minDevice) error {
	for idx := range devices {
		d := &devices[idx]
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		typ, rt, err := l.seektime(ctx, d.IPath)
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
func (l *lsblkDeviceManager) setDeviceType(device *minDevice, typ zos.DeviceType, readTime uint64) {
	device.DiskType = typ
	device.RTime = readTime

}

func (l *lsblkDeviceManager) deviceTypeFromString(typ string) zos.DeviceType {
	switch strings.ToLower(typ) {
	case string(zos.SSDDevice):
		return zos.SSDDevice
	case string(zos.HDDDevice):
		return zos.HDDDevice
	default:
		// if we have an error or unrecognized type, set type to HDD
		return zos.HDDDevice
	}
}

// ByReadTime implements sort.Interface for []Device based on the ReadTime field
type ByReadTime Devices

func (a ByReadTime) Len() int           { return len(a) }
func (a ByReadTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByReadTime) Less(i, j int) bool { return a[i].ReadTime() < a[j].ReadTime() }
