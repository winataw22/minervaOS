package filesystem

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/threefoldtech/zosv2/modules"
)

var (
	_ Filesystem = (*btrfs)(nil)

	// ErrDeviceAlreadyMounted indicates that a mounted device is attempted
	// to be mounted again (without MS_BIND flag).
	ErrDeviceAlreadyMounted = fmt.Errorf("device is already mounted")
	// ErrDeviceNotMounted is returned when an action is performed on a device
	// which requires the device to be mounted, while it is not.
	ErrDeviceNotMounted = fmt.Errorf("device is not mounted")
)

var (
	// divisors for the total usable size of a filesystem
	// an efficiency multiplier would probably make slightly more sense,
	// but this way we don't have to cast uints to floats later
	raidSizeDivisor = map[modules.RaidProfile]uint64{
		modules.Single: 1,
		modules.Raid1:  2,
		modules.Raid10: 2,
	}
)

// btrfs is the filesystem implementation for btrfs
type btrfs struct {
	devices DeviceManager
}

// NewBtrfs creates a new filesystem that implements btrfs
func NewBtrfs(manager DeviceManager) Filesystem {
	return &btrfs{manager}
}

func (b *btrfs) btrfs(ctx context.Context, args ...string) ([]byte, error) {
	return run(ctx, "btrfs", args...)
}

func (b *btrfs) Create(ctx context.Context, name string, devices DeviceCache, policy modules.RaidProfile) (Pool, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return nil, fmt.Errorf("invalid name")
	}

	block, err := b.devices.ByLabel(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(block) != 0 {
		return nil, fmt.Errorf("unique name is required")
	}

	paths := []string{}
	for _, device := range devices {
		if device.Used() {
			return nil, fmt.Errorf("device '%v' is already used", device.Path)
		}

		paths = append(paths, device.Path)
	}

	args := []string{
		"-L", name,
		"-d", string(policy),
		"-m", string(policy),
	}

	args = append(args, paths...)
	if _, err := run(ctx, "mkfs.btrfs", args...); err != nil {
		return nil, err
	}

	// update cached devices
	for _, dev := range devices {
		dev.Label = name
		dev.Filesystem = BtrfsFSType
	}

	return newBtrfsPool(name, devices), nil
}

func (b *btrfs) List(ctx context.Context) ([]Pool, error) {
	var pools []Pool
	available, err := BtrfsList(ctx, "", false)
	if err != nil {
		return nil, err
	}

	for _, fs := range available {
		if len(fs.Label) == 0 {
			// we only assume labeled devices are managed
			continue
		}

		devices, err := b.devices.ByLabel(ctx, fs.Label)
		if err != nil {
			return nil, err
		}

		if len(devices) == 0 {
			// since this should not be able to happen consider it an error
			return nil, fmt.Errorf("pool %v has no corresponding devices on the system", fs.Label)
		}

		pools = append(pools, newBtrfsPool(fs.Label, devices))
	}

	return pools, nil
}

type btrfsPool struct {
	name    string
	devices DeviceCache
}

func newBtrfsPool(name string, devices DeviceCache) *btrfsPool {
	return &btrfsPool{
		name:    name,
		devices: devices,
	}
}

// Mounted checks if the pool is mounted
// It doesn't check the default mount location of the pool
// but instead check if any of the pool devices is mounted
// under any location
func (p *btrfsPool) Mounted() (string, bool) {
	ctx := context.Background()
	list, _ := BtrfsList(ctx, p.Name(), true)
	if len(list) != 1 {
		return "", false
	}

	return p.mounted(&list[0])
}

func (p *btrfsPool) mounted(fs *Btrfs) (string, bool) {
	for _, device := range fs.Devices {
		if target, ok := getMountTarget(device.Path); ok {
			return target, true
		}
	}

	return "", false
}

func (p *btrfsPool) Name() string {
	return p.name
}

func (p *btrfsPool) Path() string {
	return filepath.Join("/mnt", p.name)
}

func (p *btrfsPool) enableQuota(mnt string) error {
	_, err := run(context.Background(), "btrfs", "quota", "enable", mnt)
	return err
}

// Mount mounts the pool in it's default mount location under /mnt/name
func (p *btrfsPool) Mount() (string, error) {
	ctx := context.Background()
	list, _ := BtrfsList(ctx, p.name, false)
	if len(list) != 1 {
		return "", fmt.Errorf("unknown pool '%s'", p.name)
	}

	fs := list[0]

	if mnt, mounted := p.mounted(&fs); mounted {
		return mnt, nil
	}

	mnt := p.Path()
	if err := os.MkdirAll(mnt, 0755); err != nil {
		return "", err
	}

	if err := syscall.Mount(fs.Devices[0].Path, mnt, "btrfs", 0, ""); err != nil {
		return "", err
	}

	return mnt, p.enableQuota(mnt)
}

func (p *btrfsPool) UnMount() error {
	mnt, ok := p.Mounted()
	if !ok {
		return nil
	}

	return syscall.Unmount(mnt, syscall.MNT_DETACH)
}

func (p *btrfsPool) AddDevice(device *Device) error {
	mnt, ok := p.Mounted()
	if !ok {
		return ErrDeviceNotMounted
	}
	ctx := context.Background()

	if _, err := run(ctx, "btrfs", "device", "add", device.Path, mnt); err != nil {
		return err
	}

	p.devices = append(p.devices, device)

	// update cached device
	device.Label = p.name
	device.Filesystem = BtrfsFSType

	return nil
}

func (p *btrfsPool) RemoveDevice(device *Device) error {
	mnt, ok := p.Mounted()
	if !ok {
		return ErrDeviceNotMounted
	}
	ctx := context.Background()

	if _, err := run(ctx, "btrfs", "device", "remove", device.Path, mnt); err != nil {
		return err
	}

	for idx, d := range p.devices {
		if d.Path == device.Path {
			// remove device from list
			p.devices = append(p.devices[:idx], p.devices[idx+1:]...)
		}
	}

	// update cached device
	device.Filesystem = ""
	device.Label = ""

	return nil
}

func (p *btrfsPool) Volumes() ([]Volume, error) {
	mnt, ok := p.Mounted()
	if !ok {
		return nil, ErrDeviceNotMounted
	}

	var volumes []Volume

	subs, err := BtrfsSubvolumeList(context.Background(), mnt)
	if err != nil {
		return nil, err
	}

	for _, sub := range subs {
		volumes = append(volumes, btrfsVolume(filepath.Join(mnt, sub.Path)))
	}

	return volumes, nil
}

func (p *btrfsPool) AddVolume(name string) (Volume, error) {
	mnt, ok := p.Mounted()
	if !ok {
		return nil, ErrDeviceNotMounted
	}

	mnt = filepath.Join(mnt, name)
	if _, err := run(context.Background(), "btrfs", "subvolume", "create", mnt); err != nil {
		return nil, err
	}

	return btrfsVolume(mnt), nil
}

func (p *btrfsPool) RemoveVolume(name string) error {
	mnt, ok := p.Mounted()
	if !ok {
		return ErrDeviceNotMounted
	}

	mnt = path.Join(mnt, name)
	_, err := run(context.Background(), "btrfs", "subvolume", "delete", mnt)

	return err
}

// Size return the pool size
func (p *btrfsPool) Usage() (usage Usage, err error) {
	mnt, ok := p.Mounted()
	if !ok {
		return usage, ErrDeviceNotMounted
	}

	du, err := BtrfsGetDiskUsage(context.Background(), mnt)
	if err != nil {
		return Usage{}, err
	}

	fsi, err := BtrfsList(context.Background(), p.name, true)
	if err != nil {
		return Usage{}, err
	}

	if len(fsi) == 0 {
		return Usage{}, fmt.Errorf("could not find total size of pool %v", p.name)
	}

	var totalSize uint64
	for _, dev := range fsi[0].Devices {
		totalSize += uint64(dev.Size)
	}

	return Usage{Size: totalSize / raidSizeDivisor[du.Data.Profile], Used: uint64(fsi[0].Used)}, nil
}

// Limit on a pool is not supported yet
func (p *btrfsPool) Limit(size uint64) error {
	return fmt.Errorf("not implemented")
}

// FsType of the filesystem of this volume
func (p *btrfsPool) FsType() string {
	return "btrfs"
}

// Type of the physical storage used for this pool
func (p *btrfsPool) Type() modules.DeviceType {
	// We only create heterogenous pools for now
	return p.devices[0].DiskType
}

type btrfsVolume string

func (v btrfsVolume) Path() string {
	return string(v)
}

func (v btrfsVolume) Volumes() ([]Volume, error) {
	var volumes []Volume

	subs, err := BtrfsSubvolumeList(context.Background(), string(v))
	if err != nil {
		return nil, err
	}

	for _, sub := range subs {
		volumes = append(volumes, btrfsVolume(filepath.Join(string(v), sub.Path)))
	}

	return volumes, nil
}

func (v btrfsVolume) AddVolume(name string) (Volume, error) {
	mnt := path.Join(string(v), name)
	if _, err := run(context.Background(), "btrfs", "subvolume", "create", mnt); err != nil {
		return nil, err
	}

	return btrfsVolume(mnt), nil
}

func (v btrfsVolume) RemoveVolume(name string) error {
	mnt := path.Join(string(v), name)
	_, err := run(context.Background(), "btrfs", "subvolume", "remove", mnt)

	return err
}

// Usage return the volume usage
func (v btrfsVolume) Usage() (usage Usage, err error) {
	ctx := context.Background()
	info, err := BtrfsSubvolumeInfo(ctx, string(v))
	if err != nil {
		return usage, err
	}

	groups, err := BtrfsQGroupList(ctx, string(v))
	if err != nil {
		return usage, err
	}

	group, ok := groups[fmt.Sprintf("0/%d", info.ID)]
	if !ok {
		// no qgroup associated with the subvolume id! means no limit, but we also
		// cannot read the usage.
		return
	}

	// otherwise, we return the size as maxrefer and usage as the rfer of the
	// associated group
	// todo: size should be the size of the pool, if maxrfer is 0
	return Usage{Used: group.Rfer, Size: group.MaxRfer}, nil
}

// Limit size of volume, setting size to 0 means unlimited
func (v btrfsVolume) Limit(size uint64) error {
	ctx := context.Background()

	limit := "none"
	if size > 0 {
		limit = fmt.Sprint(size)
	}
	_, err := run(ctx, "btrfs", "qgroup", "limit", limit, string(v))

	return err
}

// Name of the filesystem
func (v btrfsVolume) Name() string {
	_, name := filepath.Split(string(v))
	return name
}

// FsType of the filesystem
func (v btrfsVolume) FsType() string {
	return "btrfs"
}
