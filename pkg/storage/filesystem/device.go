package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// DeviceManager is able to list all/specific devices on a system
type DeviceManager interface {
	// Device returns the device at the specified path
	Device(ctx context.Context, device string) (DeviceInfo, error)
	// Devices finds all devices on a system
	Devices(ctx context.Context) (Devices, error)
	// ByLabel finds all devices with the specified label
	ByLabel(ctx context.Context, label string) (Devices, error)
	// Mountpoint returns mount point of a device
	Mountpoint(ctx context.Context, device string) (string, error)
	// Seektime checks device seektime
	Seektime(ctx context.Context, device string) (zos.DeviceType, error)
}

// Devices represents a list of cached in memory devices
type Devices []DeviceInfo

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

var (
	subvolFindmntOption = regexp.MustCompile(`(^|,)subvol=/($|,)`)
)

// blockDevices lsblk output
type blockDevices struct {
	BlockDevices []DeviceInfo `json:"blockdevices"`
}

// DeviceInfo contains information about the device
type DeviceInfo struct {
	mgr DeviceManager

	Path       string `json:"path"`
	Label      string `json:"label"`
	Size       uint64 `json:"size"`
	Filesystem FSType `json:"fstype"`
	Rota       bool   `json:"rota"`
	Subsystems string `json:"subsystems"`
}

func (i *DeviceInfo) Name() string {
	return filepath.Base(i.Path)
}

// Used assumes that the device is used if it has custom label or fstype or children
func (i *DeviceInfo) Used() bool {
	return len(i.Label) != 0 || len(i.Filesystem) != 0
}

// DetectType returns the device type according to seektime
func (d *DeviceInfo) DetectType() (zos.DeviceType, error) {
	return d.mgr.Seektime(context.Background(), d.Path)
}

func (d *DeviceInfo) Mountpoint(ctx context.Context) (string, error) {
	return d.mgr.Mountpoint(ctx, d.Path)
}

// lsblkDeviceManager uses the lsblk utility to scann the disk for devices, and
// caches the result.
//
// Found devices are cached, and the cache is only repopulated after the `Scan`
// method is called.
type lsblkDeviceManager struct {
	executer
	cache []DeviceInfo
}

// DefaultDeviceManager returns a default device manager implementation
func DefaultDeviceManager() DeviceManager {
	return defaultDeviceManager(executerFunc(run))
}

func defaultDeviceManager(exec executer) DeviceManager {
	m := &lsblkDeviceManager{
		executer: exec,
	}

	return m
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Seektime(ctx context.Context, device string) (zos.DeviceType, error) {
	log.Debug().Str("device", device).Msg("checking seektim for device")
	out, err := l.executer.run(ctx, "seektime", "-j", device)
	if err != nil {
		return "", errors.Wrap(err, "failed to check device seektime")
	}

	var result struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return "", errors.Wrap(err, "failed to load seektime data")
	}

	return zos.DeviceType(strings.ToLower(result.Type)), nil
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) (Devices, error) {
	return l.scan(ctx)
}

func (l *lsblkDeviceManager) ByLabel(ctx context.Context, label string) (Devices, error) {
	devices, err := l.Devices(ctx)
	if err != nil {
		return nil, err
	}

	var filtered Devices

	for _, device := range devices {
		if device.Label == label {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

func (l *lsblkDeviceManager) Device(ctx context.Context, path string) (device DeviceInfo, err error) {
	devices, err := l.scan(ctx)
	if err != nil {
		return device, err
	}

	for _, dev := range devices {
		if dev.Path == path {
			return dev, nil
		}
	}

	return device, fmt.Errorf("device not found")

}

func (l *lsblkDeviceManager) lsblk(ctx context.Context) ([]DeviceInfo, error) {
	var devices blockDevices

	args := []string{
		"--json",
		"-o",
		"PATH,NAME,SIZE,SUBSYSTEMS,FSTYPE,LABEL,ROTA",
		"--bytes",
		"--exclude",
		"1,2,11",
		"--path",
	}

	bytes, err := l.run(ctx, "lsblk", args...)
	if err != nil {
		return nil, err
	}

	// skipping unmarshal when lsblk response is empty
	if len(bytes) == 0 {
		return nil, nil
	}

	// parsing lsblk response
	if err := json.Unmarshal(bytes, &devices); err != nil {
		return nil, err
	}

	for i := range devices.BlockDevices {
		devices.BlockDevices[i].mgr = l
	}

	return devices.BlockDevices, nil
}

func (l *lsblkDeviceManager) Mountpoint(ctx context.Context, device string) (string, error) {
	// to not pollute global namespace with ugly hack types
	var mounts struct {
		Filesystems []struct {
			Source  string
			Target  string
			Options string
		}
	}
	args := []string{
		"-J",
		"-S", device,
	}

	bytes, err := l.run(ctx, "findmnt", args...)
	if err != nil {
		// empty output and exit code 1 in case the device is not found
		return "", nil
	}
	if len(bytes) != 0 {
		if err := json.Unmarshal(bytes, &mounts); err != nil {
			return "", err
		}
	}

	for _, m := range mounts.Filesystems {
		if subvolFindmntOption.MatchString(m.Options) {
			return m.Target, nil
		}
	}

	return "", nil

}

func (l *lsblkDeviceManager) raw(ctx context.Context) ([]DeviceInfo, error) {
	devices, err := l.lsblk(ctx)
	if err != nil {
		return nil, err
	}

	filtered := devices[:0]
	for _, device := range devices {
		if device.Subsystems != "block:scsi:usb:pci" {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

// scan the system for disks using the `lsblk` command
func (l *lsblkDeviceManager) scan(ctx context.Context) ([]DeviceInfo, error) {
	if l.cache != nil {
		return l.cache, nil
	}

	devs, err := l.raw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan devices")
	}

	l.cache = devs
	return l.cache, nil
}
