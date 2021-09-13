package flist

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	fsTypeG8ufs   = "fuse.g8ufs"
	fsTypeOverlay = "overlay"
)

// //nolint
// type filter func(i *mountInfo) bool

// //nolint
// func withParentDir(path string) filter {
// 	path = filepath.Clean(path)
// 	return func(mnt *mountInfo) bool {
// 		base := filepath.Base(mnt.Target)
// 		return path == base
// 	}
// }

type mountInfo struct {
	Target  string `json:"target"`
	Source  string `json:"source"`
	FSType  string `json:"fstype"`
	Options string `json:"options"`
}

type overlayInfo struct {
	LowerDir string
	UpperDir string
	WorkDir  string
}

type g8ufsInfo struct {
	Pid int64
}

func (m *mountInfo) AsOverlay() overlayInfo {
	var info overlayInfo
	for _, part := range strings.Split(m.Options, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "lowerdir":
			info.LowerDir = kv[1]
		case "upperdir":
			info.UpperDir = kv[1]
		case "workdir":
			info.WorkDir = kv[1]
		}
	}

	return info
}

func (m *mountInfo) AsG8ufs() g8ufsInfo {
	var info g8ufsInfo
	if pid, err := strconv.ParseInt(m.Source, 10, 64); err == nil {
		info.Pid = pid
	}
	return info
}

func (f *flistModule) getMount(path string) (info mountInfo, err error) {
	output, err := f.commander.Command("findmnt", "-J", path).Output()
	if err, ok := err.(*exec.ExitError); ok && err != nil {
		if err.ExitCode() == 1 {
			return info, ErrNotMountPoint
		}
	}

	var result struct {
		Filesystems []mountInfo `json:"filesystems"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return info, errors.Wrap(err, "failed to parse findmnt output")
	}

	if len(result.Filesystems) != 1 {
		return info, fmt.Errorf("invalid number of mounts in output")
	}

	return result.Filesystems[0], nil
}

func (f *flistModule) resolve(path string) (g8ufsInfo, error) {
	info, err := f.getMount(path)
	if err != nil {
		return g8ufsInfo{}, err
	}

	if info.FSType == fsTypeG8ufs {
		return info.AsG8ufs(), nil
	} else if info.FSType == fsTypeOverlay {
		overlay := info.AsOverlay()
		return f.resolve(overlay.LowerDir)
	} else {
		return g8ufsInfo{}, fmt.Errorf("invalid mount fs type: '%s'", info.FSType)
	}
}

// //nolint
// func (f *flistModule) mounts(filter ...filter) ([]mountInfo, error) {
// 	output, err := f.commander.Command("findmnt", "-J", "-l").Output()
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to list system mounts")
// 	}

// 	var result struct {
// 		Filesystems []mountInfo `json:"filesystems"`
// 	}

// 	if err := json.Unmarshal(output, &result); err != nil {
// 		return nil, errors.Wrap(err, "failed to parse findmnt output")
// 	}

// 	if len(filter) == 0 {
// 		return result.Filesystems, nil
// 	}

// 	mounts := result.Filesystems[:0]
// next:
// 	for _, mnt := range result.Filesystems {
// 		for _, f := range filter {
// 			if !f(&mnt) {
// 				continue next
// 			}
// 		}
// 		mounts = append(mounts, mnt)
// 	}

// 	return mounts, nil
// }
