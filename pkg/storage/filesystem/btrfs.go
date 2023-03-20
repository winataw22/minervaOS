package filesystem

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrDeviceAlreadyMounted indicates that a mounted device is attempted
	// to be mounted again (without MS_BIND flag).
	ErrDeviceAlreadyMounted = fmt.Errorf("device is already mounted")
	// ErrDeviceNotMounted is returned when an action is performed on a device
	// which requires the device to be mounted, while it is not.
	ErrDeviceNotMounted = fmt.Errorf("device is not mounted")
)

type btrfsPool struct {
	device DeviceInfo
	utils  BtrfsUtil
	name   string
}

// NewBtrfsPool creates a btrfs pool associated with device.
// if device does not have a filesystem one is created
func NewBtrfsPool(device DeviceInfo) (Pool, error) {
	return newBtrfsPool(device, executerFunc(run))
}

func newBtrfsPool(device DeviceInfo, exe executer) (Pool, error) {
	pool := &btrfsPool{
		device: device,
		utils:  newUtils(exe),
	}

	return pool, pool.prepare()
}

func (p *btrfsPool) ID() int {
	return 0
}

func (p *btrfsPool) Device() DeviceInfo {
	return p.device
}

func (p *btrfsPool) prepare() error {
	p.name = p.device.Label
	if p.device.Used() {
		// device already have filesystem
		return nil
	}

	ctx := context.Background()

	// otherwise format
	if err := p.format(ctx); err != nil {
		return err
	}
	// make sure kernel knows about this
	return Partprobe(ctx)
}

func (p *btrfsPool) format(ctx context.Context) error {
	name := uuid.New().String()
	p.name = name

	args := []string{
		"-L", name,
		p.device.Path,
	}

	if _, err := p.utils.run(ctx, "mkfs.btrfs", args...); err != nil {
		return errors.Wrapf(err, "failed to format device '%s'", p.device.Path)
	}

	return nil
}

// Mounted checks if the pool is mounted
// It doesn't check the default mount location of the pool
// but instead check if any of the pool devices is mounted
// under any location
func (p *btrfsPool) Mounted() (string, error) {
	ctx := context.TODO()
	mnt, err := p.device.Mountpoint(ctx)
	if err != nil {
		return "", err
	}

	if len(mnt) != 0 {
		return mnt, nil
	}

	return "", ErrDeviceNotMounted
}

func (p *btrfsPool) Name() string {
	return p.name
}

func (p *btrfsPool) Path() string {
	return filepath.Join("/mnt", p.name)
}

// Limit on a pool is not supported yet
func (p *btrfsPool) Limit(size uint64) error {
	return fmt.Errorf("not implemented")
}

// FsType of the filesystem of this volume
func (p *btrfsPool) FsType() string {
	return "btrfs"
}

// Mount mounts the pool in it's default mount location under /mnt/name
func (p *btrfsPool) Mount() (string, error) {
	ctx := context.TODO()
	mnt, err := p.Mounted()
	if err == nil {
		return mnt, nil
	} else if !errors.Is(err, ErrDeviceNotMounted) {
		return "", errors.Wrap(err, "failed to check device mount status")
	}

	// device is not mounted
	mnt = p.Path()
	if err := os.MkdirAll(mnt, 0755); err != nil {
		return "", err
	}

	if err := syscall.Mount(p.device.Path, mnt, "btrfs", 0, ""); err != nil {
		return "", err
	}

	if err := p.utils.QGroupEnable(ctx, mnt); err != nil {
		return "", fmt.Errorf("failed to enable qgroup: %w", err)
	}

	return mnt, p.maintenance()
}

func (p *btrfsPool) UnMount() error {
	mnt, err := p.Mounted()
	if errors.Is(err, ErrDeviceNotMounted) {
		return nil
	} else if err != nil {
		return err
	}

	return syscall.Unmount(mnt, syscall.MNT_DETACH)
}

func (p *btrfsPool) Volumes() ([]Volume, error) {
	mnt, err := p.Mounted()
	if err != nil {
		return nil, err
	}

	var volumes []Volume

	subs, err := p.utils.SubvolumeList(context.Background(), mnt)
	if err != nil {
		return nil, err
	}

	for _, sub := range subs {
		volumes = append(volumes, newBtrfsVolume(
			sub.ID,
			filepath.Join(mnt, sub.Path),
			p.utils,
		))
	}

	return volumes, nil
}

func (p *btrfsPool) addVolume(root string) (Volume, error) {
	ctx := context.Background()
	if err := p.utils.SubvolumeAdd(ctx, root); err != nil {
		return nil, err
	}

	volume, err := p.utils.SubvolumeInfo(ctx, root)
	if err != nil {
		return nil, err
	}

	return newBtrfsVolume(volume.ID, root, p.utils), nil
}

func (p *btrfsPool) AddVolume(name string) (Volume, error) {
	mnt, err := p.Mounted()
	if err != nil {
		return nil, err
	}

	root := filepath.Join(mnt, name)
	return p.addVolume(root)
}

func (p *btrfsPool) removeVolume(root string) error {
	ctx := context.Background()

	info, err := p.utils.SubvolumeInfo(ctx, root)
	if err != nil {
		return err
	}

	if err := p.utils.SubvolumeRemove(ctx, root); err != nil {
		return err
	}

	qgroupID := fmt.Sprintf("0/%d", info.ID)
	if err := p.utils.QGroupDestroy(ctx, qgroupID, p.Path()); err != nil {
		return errors.Wrapf(err, "failed to delete qgroup %s", qgroupID)
	}

	return nil
}

func (p *btrfsPool) RemoveVolume(name string) error {
	mnt, err := p.Mounted()
	if err != nil {
		return err
	}

	root := filepath.Join(mnt, name)
	return p.removeVolume(root)
}

// Size return the pool size
func (p *btrfsPool) Usage() (usage Usage, err error) {
	mnt, err := p.Mounted()
	if err != nil {
		return usage, err
	}

	volumes, err := p.Volumes()

	if err != nil {
		return usage, errors.Wrapf(err, "failed to list pool '%s' volumes", mnt)
	}

	usage.Size = p.device.Size

	for _, volume := range volumes {
		vol, err := volume.Usage()
		if err != nil {
			return Usage{}, errors.Wrapf(err, "failed to calculate volume '%s' usage", volume.Path())
		}

		usage.Used += vol.Used
		usage.Excl += vol.Excl
	}

	return
}

func (p *btrfsPool) maintenance() error {
	// this method cleans up all the unused
	// qgroups that could exists on a filesystem

	volumes, err := p.Volumes()
	if err != nil {
		return err
	}
	subVolsIDs := map[string]struct{}{}
	for _, volume := range volumes {
		// use the 0/X notation to match the qgroup IDs format
		subVolsIDs[fmt.Sprintf("0/%d", volume.ID())] = struct{}{}
	}

	ctx := context.Background()
	qgroups, err := p.utils.QGroupList(ctx, p.Path())
	if err != nil {
		return err
	}

	for qgroupID := range qgroups {
		// for all qgroup that doesn't have an linked
		// volume, delete the qgroup
		_, ok := subVolsIDs[qgroupID]
		if !ok {
			log.Debug().Str("id", qgroupID).Msg("destroy qgroup")
			if err := p.utils.QGroupDestroy(ctx, qgroupID, p.Path()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *btrfsPool) Shutdown() error {
	cmd := exec.Command("hdparm", "-y", p.device.Path)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to shutdown device '%s'", p.device.Path)
	}

	return nil
}

type btrfsVolume struct {
	id    int
	path  string
	utils BtrfsUtil
}

func newBtrfsVolume(ID int, path string, utils BtrfsUtil) Volume {
	vol := btrfsVolume{
		id:    ID,
		path:  path,
		utils: utils,
	}

	return &vol
}

func (v *btrfsVolume) ID() int {
	return v.id
}

func (v *btrfsVolume) Path() string {
	return v.path
}

// Name of the filesystem
func (v *btrfsVolume) Name() string {
	return filepath.Base(v.Path())
}

// FsType of the filesystem
func (v *btrfsVolume) FsType() string {
	return "btrfs"
}

// Usage return the volume usage
func (v *btrfsVolume) Usage() (usage Usage, err error) {
	ctx := context.TODO()
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

	// used is basically amount of space reserved for this
	// volume. We assume that's total usage of the volume
	used := group.MaxRfer

	if used == 0 {
		// in case no limit is set on the subvolume, we assume
		// it's size is the size of the files on that volumes
		// or a special case when the volume is a zdb volume
		used, err = volumeUsage(v.Path())
		if err != nil {
			return usage, errors.Wrap(err, "failed to get subvolume usage")
		}
	}

	return Usage{Used: used, Size: group.MaxRfer, Excl: group.Excl}, nil
}

// Limit size of volume, setting size to 0 means unlimited
func (v *btrfsVolume) Limit(size uint64) error {
	ctx := context.Background()

	return v.utils.QGroupLimit(ctx, size, v.Path())
}
