package filesystem

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/threefoldtech/zos/pkg"
)

var (
	reBtrfsFilesystemDf = regexp.MustCompile(`(?m:(\w+),\s(\w+):\s+total=(\d+),\s+used=(\d+))`)
	reBtrfsQgroup       = regexp.MustCompile(`(?m:^(\d+/\d+)\s+(\d+)\s+(\d+)\s+(\d+|none)\s+(\d+|none).*$)`)
)

// Btrfs holds metadata of underlying btrfs filesystem
type Btrfs struct {
	Label        string        `json:"label"`
	UUID         string        `json:"uuid"`
	TotalDevices int           `json:"total_devices"`
	Used         int64         `json:"used"`
	Devices      []BtrfsDevice `json:"devices"`
	Warnings     string        `json:"warnings"`
}

// BtrfsDevice holds metadata about a single device in a btrfs filesystem
type BtrfsDevice struct {
	Missing bool   `json:"missing,omitempty"`
	DevID   int    `json:"dev_id"`
	Size    int64  `json:"size"`
	Used    int64  `json:"used"`
	Path    string `json:"path"`
}

// BtrfsVolume holds metadata about a single subvolume
type BtrfsVolume struct {
	Path       string
	ID         int
	Generation int
	ParentID   int
}

// BtrfsQGroup is parsed btrfs qgroup information
type BtrfsQGroup struct {
	ID      string
	Rfer    uint64
	Excl    uint64
	MaxRfer uint64
	MaxExcl uint64
}

// DiskUsage is parsed information from a btrfs fi df line
type DiskUsage struct {
	Profile pkg.RaidProfile `json:"profile"`
	Total   uint64          `json:"total"`
	Used    uint64          `json:"used"`
}

// BtrfsDiskUsage is parsed information form btrfs fi df
type BtrfsDiskUsage struct {
	Data          DiskUsage `json:"data"`
	System        DiskUsage `json:"system"`
	Metadata      DiskUsage `json:"metadata"`
	GlobalReserve DiskUsage `json:"globalreserve"`
}

// BtrfsUtil utils for btrfs
type BtrfsUtil struct {
	executer
}

// NewUtils create a new BtrfsUtil object
func NewUtils() BtrfsUtil {
	return BtrfsUtil{executerFunc(run)}
}

func newUtils(exec executer) BtrfsUtil {
	return BtrfsUtil{exec}
}

// List lists all availabel btrfs pools
// if label is provided, only get fs of that label, if mounted = True, only return
// mounted filesystems, otherwise any.
func (u *BtrfsUtil) List(ctx context.Context, label string, mounted bool) ([]Btrfs, error) {
	args := []string{
		"filesystem", "show", "--raw",
	}

	if mounted {
		args = append(args, "-m")
	}

	if len(label) != 0 {
		args = append(args, label)
	}

	output, err := u.run(ctx, "btrfs", args...)
	if err != nil {
		return nil, err
	}

	return parseList(string(output))
}

// SubvolumeList list direct subvolumes of this location
func (u *BtrfsUtil) SubvolumeList(ctx context.Context, root string) ([]BtrfsVolume, error) {
	output, err := u.run(ctx, "btrfs", "subvolume", "list", "-o", root)
	if err != nil {
		return nil, err
	}

	return parseSubvolList(string(output))
}

// SubvolumeInfo get info of a subvolume giving its path
func (u *BtrfsUtil) SubvolumeInfo(ctx context.Context, path string) (volume BtrfsVolume, err error) {
	output, err := u.run(context.Background(), "btrfs", "subvolume", "show", path)
	if err != nil {
		return
	}

	volume, err = parseSubvolInfo(string(output))
	volume.Path = path

	return
}

// SubvolumeAdd adds a new subvolume at path
func (u *BtrfsUtil) SubvolumeAdd(ctx context.Context, root string) error {
	_, err := u.run(ctx, "btrfs", "subvolume", "create", root)
	return err
}

// SubvolumeRemove removes a subvolume
func (u *BtrfsUtil) SubvolumeRemove(ctx context.Context, root string) error {
	_, err := u.run(ctx, "btrfs", "subvolume", "delete", root)
	return err
}

// DeviceAdd adds a device to a btrfs pool
func (u *BtrfsUtil) DeviceAdd(ctx context.Context, dev string, root string) error {
	_, err := u.run(ctx, "btrfs", "device", "add", dev, root)
	return err
}

// DeviceRemove removes a device from a btrfs pool
func (u *BtrfsUtil) DeviceRemove(ctx context.Context, dev string, root string) error {
	_, err := u.run(ctx, "btrfs", "device", "remove", dev, root)
	return err
}

// QGroupEnable enable quota
func (u *BtrfsUtil) QGroupEnable(ctx context.Context, root string) error {
	_, err := u.run(ctx, "btrfs", "quota", "enable", root)
	return err
}

// QGroupList list available qgroups
func (u *BtrfsUtil) QGroupList(ctx context.Context, path string) (map[string]BtrfsQGroup, error) {
	output, err := u.run(ctx, "btrfs", "qgroup", "show", "-re", "--raw", path)
	if err != nil {
		return nil, err
	}

	return parseQGroups(string(output)), nil
}

// QGroupLimit limit size on subvol
func (u *BtrfsUtil) QGroupLimit(ctx context.Context, size uint64, path string) error {
	limit := "none"
	if size > 0 {
		limit = fmt.Sprint(size)
	}

	_, err := u.run(ctx, "btrfs", "qgroup", "limit", limit, path)

	return err
}

// QGroupDestroy deletes a qgroup on a subvol
func (u *BtrfsUtil) QGroupDestroy(ctx context.Context, id, path string) error {
	_, err := u.run(ctx, "btrfs", "qgroup", "destroy", id, path)

	return err
}

// GetDiskUsage get btrfs usage
func (u *BtrfsUtil) GetDiskUsage(ctx context.Context, path string) (usage BtrfsDiskUsage, err error) {
	output, err := u.run(ctx, "btrfs", "filesystem", "df", "--raw", path)
	if err != nil {
		return usage, err
	}

	return parseFilesystemDF(string(output))
}

func parseSubvolInfo(output string) (volume BtrfsVolume, err error) {
	values := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		value := strings.TrimSpace(parts[1])
		if value == "-" {
			continue
		}
		values[strings.TrimSpace(parts[0])] = value
	}

	if _, err := fmt.Sscanf(values["Subvolume ID"], "%d", &volume.ID); err != nil {
		return volume, err
	}

	if _, err := fmt.Sscanf(values["Parent ID"], "%d", &volume.ParentID); err != nil {
		return volume, err
	}

	if _, err := fmt.Sscanf(values["Generation"], "%d", &volume.Generation); err != nil {
		return volume, err
	}

	return
}

func parseSubvolList(output string) ([]BtrfsVolume, error) {
	var volumes []BtrfsVolume
	for _, line := range strings.Split(output, "\n") {
		// ID 263 gen 45 top level 261 path root/home
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var volume BtrfsVolume
		if _, err := fmt.Sscanf(line, "ID %d gen %d top level %d path %s",
			&volume.ID, &volume.Generation, &volume.ParentID, &volume.Path); err != nil {
			return nil, err
		}

		volumes = append(volumes, volume)
	}

	return volumes, nil
}

func parseList(output string) ([]Btrfs, error) {
	var fss []Btrfs

	blocks := strings.Split(output, "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		// Ensure that fsLines starts with Label (and collect all warnings into fs.Warnings)
		labelIdx := strings.Index(block, "Label:")
		if labelIdx != 0 {
			block = block[labelIdx:]
		}
		fsLines := strings.Split(block, "\n")
		if len(fsLines) < 3 {
			continue
		}
		fs, err := parseFS(fsLines)

		if err != nil {
			return fss, err
		}
		fss = append(fss, fs)
	}
	return fss, nil
}

func parseFS(lines []string) (Btrfs, error) {
	// first line should be label && uuid
	var label, uuid string
	_, err := fmt.Sscanf(lines[0], `Label: %s uuid: %s`, &label, &uuid)
	if err != nil {
		return Btrfs{}, err
	}
	if label != "none" {
		label = label[1 : len(label)-1]
	}

	// total device & byte used
	var totDevice int
	var used int64
	if _, err := fmt.Sscanf(strings.TrimSpace(lines[1]), "Total devices %d FS bytes used %d", &totDevice, &used); err != nil {
		return Btrfs{}, err
	}
	var validDevsLines []string
	var fsWarnings string
	for _, line := range lines[2:] {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "**") {
			// a warning
			fsWarnings += trimmedLine
		} else {
			validDevsLines = append(validDevsLines, line)
		}
	}
	devs, err := parseDevices(validDevsLines)
	if err != nil {
		return Btrfs{}, err
	}
	return Btrfs{
		Label:        label,
		UUID:         uuid,
		TotalDevices: totDevice,
		Used:         used,
		Devices:      devs,
		Warnings:     fsWarnings,
	}, nil
}

func parseDevices(lines []string) ([]BtrfsDevice, error) {
	var devs []BtrfsDevice
	for _, line := range lines {
		if line == "" {
			continue
		}
		var dev BtrfsDevice
		if _, err := fmt.Sscanf(strings.TrimSpace(line), "devid    %d size %d used %d path %s", &dev.DevID, &dev.Size, &dev.Used, &dev.Path); err == nil {
			devs = append(devs, dev)
		}
	}
	return devs, nil
}

func parseQGroups(output string) map[string]BtrfsQGroup {
	qgroups := make(map[string]BtrfsQGroup)
	for _, line := range reBtrfsQgroup.FindAllStringSubmatch(output, -1) {
		qgroup := BtrfsQGroup{
			ID: line[1],
		}

		qgroup.Rfer, _ = strconv.ParseUint(line[2], 10, 64)
		qgroup.Excl, _ = strconv.ParseUint(line[3], 10, 64)
		if line[4] != "none" {
			qgroup.MaxRfer, _ = strconv.ParseUint(line[4], 10, 64)
		}

		if line[5] != "none" {
			qgroup.MaxExcl, _ = strconv.ParseUint(line[5], 10, 64)
		}

		qgroups[qgroup.ID] = qgroup
	}

	return qgroups
}

func parseFilesystemDF(output string) (usage BtrfsDiskUsage, err error) {
	lines := reBtrfsFilesystemDf.FindAllStringSubmatch(output, -1)
	for _, line := range lines {
		name := line[1]
		var datainfo *DiskUsage
		switch name {
		case "Data":
			datainfo = &usage.Data
		case "System":
			datainfo = &usage.System
		case "Metadata":
			datainfo = &usage.Metadata
		case "GlobalReserve":
			datainfo = &usage.GlobalReserve
		default:
			continue
		}
		datainfo.Profile = pkg.RaidProfile(strings.ToLower(line[2]))
		datainfo.Total, err = strconv.ParseUint(line[3], 10, 64)
		if err != nil {
			return
		}

		datainfo.Used, err = strconv.ParseUint(line[4], 10, 64)
		if err != nil {
			return
		}

	}

	return
}
