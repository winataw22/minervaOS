package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg"
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
	raidSizeDivisor = map[pkg.RaidProfile]uint64{
		pkg.Single: 1,
		pkg.Raid1:  2,
		pkg.Raid10: 2,
	}
)

// btrfs is the filesystem implementation for btrfs
type btrfs struct {
	devices DeviceManager
	utils   BtrfsUtil
}

func newBtrfs(manager DeviceManager, exec executer) *btrfs {
	return &btrfs{devices: manager, utils: newUtils(exec)}
}

// NewBtrfs creates a new filesystem that implements btrfs
func NewBtrfs(manager DeviceManager) Filesystem {
	return newBtrfs(manager, executerFunc(run))
}

func (b *btrfs) btrfs(ctx context.Context, args ...string) ([]byte, error) {
	return b.utils.run(ctx, "btrfs", args...)
}

func (b *btrfs) Create(ctx context.Context, name string, policy pkg.RaidProfile, devices ...*Device) (Pool, error) {
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
	if _, err := b.utils.run(ctx, "mkfs.btrfs", args...); err != nil {
		return nil, err
	}

	// update cached devices
	for _, dev := range devices {
		dev.Label = name
		dev.Filesystem = BtrfsFSType
	}

	return newBtrfsPool(name, devices, &b.utils), nil
}

func (b *btrfs) List(ctx context.Context, filter Filter) ([]Pool, error) {
	if filter == nil {
		filter = All
	}
	var pools []Pool
	available, err := b.utils.List(ctx, "", false)
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

		pool := newBtrfsPool(fs.Label, devices, &b.utils)

		if !filter(pool) {
			continue
		}

		if len(devices) == 0 {
			// since this should not be able to happen consider it an error
			return nil, fmt.Errorf("pool %v has no corresponding devices on the system", fs.Label)
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

type btrfsPool struct {
	name    string
	devices []*Device
	utils   *BtrfsUtil
}

func newBtrfsPool(name string, devices []*Device, utils *BtrfsUtil) *btrfsPool {
	return &btrfsPool{
		name:    name,
		devices: devices,
		utils:   utils,
	}
}

// Mounted checks if the pool is mounted
// It doesn't check the default mount location of the pool
// but instead check if any of the pool devices is mounted
// under any location
func (p *btrfsPool) Mounted() (string, bool) {
	ctx := context.Background()
	list, _ := p.utils.List(ctx, p.Name(), true)
	if len(list) != 1 {
		return "", false
	}

	return p.mounted(&list[0])
}

func (p *btrfsPool) mounted(fs *Btrfs) (string, bool) {
	for _, device := range fs.Devices {
		if target, ok := GetMountTarget(device.Path); ok {
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

// Mount mounts the pool in it's default mount location under /mnt/name
func (p *btrfsPool) Mount() (string, error) {
	ctx := context.Background()
	list, _ := p.utils.List(ctx, p.name, false)
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

	return mnt, p.utils.QGroupEnable(ctx, mnt)
}

func (p *btrfsPool) UnMount() error {
	mnt, ok := p.Mounted()
	if !ok {
		return nil
	}

	return syscall.Unmount(mnt, syscall.MNT_DETACH)
}

func (p *btrfsPool) addDevice(device *Device, root string) error {
	ctx := context.Background()

	if err := p.utils.DeviceAdd(ctx, device.Path, root); err != nil {
		return err
	}

	// update cached device
	device.Label = p.name
	device.Filesystem = BtrfsFSType

	p.devices = append(p.devices, device)

	return nil
}

func (p *btrfsPool) AddDevice(device *Device) error {
	mnt, ok := p.Mounted()
	if !ok {
		return ErrDeviceNotMounted
	}

	return p.addDevice(device, mnt)
}

func (p *btrfsPool) removeDevice(device *Device, root string) error {
	ctx := context.Background()

	if err := p.utils.DeviceRemove(ctx, device.Path, root); err != nil {
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

func (p *btrfsPool) RemoveDevice(device *Device) error {
	mnt, ok := p.Mounted()
	if !ok {
		return ErrDeviceNotMounted
	}

	return p.removeDevice(device, mnt)
}

func (p *btrfsPool) Volumes() ([]Volume, error) {
	mnt, ok := p.Mounted()
	if !ok {
		return nil, ErrDeviceNotMounted
	}

	var volumes []Volume

	subs, err := p.utils.SubvolumeList(context.Background(), mnt)
	if err != nil {
		return nil, err
	}

	for _, sub := range subs {
		volumes = append(volumes, newBtrfsVolume(
			filepath.Join(mnt, sub.Path),
			p.utils,
		))
	}

	return volumes, nil
}

func (p *btrfsPool) addVolume(root string) (*btrfsVolume, error) {
	if err := p.utils.SubvolumeAdd(context.Background(), root); err != nil {
		return nil, err
	}

	return newBtrfsVolume(root, p.utils), nil
}

func (p *btrfsPool) AddVolume(name string) (Volume, error) {
	mnt, ok := p.Mounted()
	if !ok {
		return nil, ErrDeviceNotMounted
	}

	root := filepath.Join(mnt, name)
	return p.addVolume(root)
}

func (p *btrfsPool) removeVolume(root string) error {
	return p.utils.SubvolumeRemove(context.Background(), root)
}

func (p *btrfsPool) RemoveVolume(name string) error {
	mnt, ok := p.Mounted()
	if !ok {
		return ErrDeviceNotMounted
	}

	root := filepath.Join(mnt, name)
	return p.removeVolume(root)
}

// Size return the pool size
func (p *btrfsPool) Usage() (usage Usage, err error) {
	mnt, ok := p.Mounted()
	if !ok {
		return usage, ErrDeviceNotMounted
	}

	du, err := p.utils.GetDiskUsage(context.Background(), mnt)
	if err != nil {
		return Usage{}, err
	}

	fsi, err := p.utils.List(context.Background(), p.name, true)
	if err != nil {
		return Usage{}, err
	}

	if len(fsi) == 0 {
		return Usage{}, fmt.Errorf("could not find total size of pool %v", p.name)
	}

	var totalSize uint64
	for _, dev := range fsi[0].Devices {
		log.Debug().Int64("size", dev.Size).Str("device", dev.Path).Msg("pool usage")
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
func (p *btrfsPool) Type() pkg.DeviceType {
	// We only create heterogenous pools for now
	return p.devices[0].DiskType
}

// Reserved is reserved size of the devices in bytes
func (p *btrfsPool) Reserved() (uint64, error) {

	volumes, err := p.Volumes()
	if err != nil {
		return 0, err
	}

	var total uint64
	for _, volume := range volumes {
		usage, err := volume.Usage()
		if err != nil {
			return 0, err
		}
		total += usage.Size
	}

	return total, nil
}

type btrfsVolume struct {
	path  string
	utils *BtrfsUtil
}

func newBtrfsVolume(path string, utils *BtrfsUtil) *btrfsVolume {
	return &btrfsVolume{
		path:  path,
		utils: utils,
	}
}

func (v *btrfsVolume) Path() string {
	return v.path
}

func (v *btrfsVolume) Volumes() ([]Volume, error) {
	var volumes []Volume

	subs, err := v.utils.SubvolumeList(context.Background(), v.Path())
	if err != nil {
		return nil, err
	}

	for _, sub := range subs {
		volumes = append(volumes, newBtrfsVolume(filepath.Join(v.Path(), sub.Path), v.utils))
	}

	return volumes, nil
}

func (v *btrfsVolume) AddVolume(name string) (Volume, error) {
	mnt := filepath.Join(v.Path(), name)
	if err := v.utils.SubvolumeAdd(context.Background(), mnt); err != nil {
		return nil, err
	}

	return newBtrfsVolume(mnt, v.utils), nil
}

func (v *btrfsVolume) RemoveVolume(name string) error {
	mnt := filepath.Join(v.Path(), name)
	return v.utils.SubvolumeRemove(context.Background(), mnt)
}

// Usage return the volume usage
func (v *btrfsVolume) Usage() (usage Usage, err error) {
	ctx := context.Background()
	info, err := v.utils.SubvolumeInfo(ctx, v.Path())
	if err != nil {
		return usage, err
	}

	groups, err := v.utils.QGroupList(ctx, v.Path())
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
func (v *btrfsVolume) Limit(size uint64) error {
	ctx := context.Background()

	return v.utils.QGroupLimit(ctx, size, v.Path())
}

// Name of the filesystem
func (v *btrfsVolume) Name() string {
	return filepath.Base(v.Path())
}

// FsType of the filesystem
func (v *btrfsVolume) FsType() string {
	return "btrfs"
}
