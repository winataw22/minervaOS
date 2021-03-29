package vm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

var errNoMoreFirecracker = fmt.Errorf("not more fc processes")

func findAllFC() (map[string]int, error) {
	const (
		proc   = "/proc"
		search = "/firecracker"
		idFlag = "--id"
	)

	found := make(map[string]int)
	err := filepath.Walk(proc, func(path string, info os.FileInfo, _ error) error {
		if path == proc {
			// assend into /proc
			return nil
		}

		dir, name := filepath.Split(path)

		if filepath.Clean(dir) != proc {
			// this to make sure we only scan first level
			return filepath.SkipDir
		}

		pid, err := strconv.Atoi(name)
		if err != nil {
			//not a number
			return nil //continue scan
		}
		cmd, err := ioutil.ReadFile(filepath.Join(path, "cmdline"))
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
		}

		parts := bytes.Split(cmd, []byte{0})
		if string(parts[0]) != search {
			return nil
		}

		// a firecracker instance, now find id
		for i, part := range parts {
			if string(part) == idFlag {
				// a hit
				if i == len(parts)-1 {
					// --id some how is last element of the array
					// so avoid a panic by skipping this
					return nil
				}
				id := parts[i+1]
				found[string(id)] = pid
				// this is to stop the scan.
				return nil
			}
		}

		return nil
	})

	return found, err
}

//findFC find legacy fc procsses
func findFC(name string) (int, error) {
	machines, err := findAllFC()
	if err != nil {
		return 0, err
	}

	pid, ok := machines[name]
	if !ok {
		return 0, fmt.Errorf("vm '%s' not found", name)
	}

	return pid, nil
}

// LegacyMonitor has code to clean up legacy machines death
type LegacyMonitor struct {
	root string
}

// Monitor start vms  monitoring
func (m *LegacyMonitor) Monitor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-time.After(monitorEvery):
				err := m.monitor(ctx)
				if err == errNoMoreFirecracker {
					return
				} else if err != nil {
					log.Error().Err(err).Msg("failed to run monitoring")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *LegacyMonitor) machineRoot(id string) string {
	return filepath.Join(m.root, "firecracker", id)
}

func (m *LegacyMonitor) cleanFsFirecracker(id string) error {
	root := filepath.Join(m.machineRoot(id), "root")

	files, err := ioutil.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}

		// we try to unmount every file in the directory
		// because it's faster than trying to find exactly
		// what files are mounted under this location.
		path := filepath.Join(root, entry.Name())
		err := syscall.Unmount(
			path,
			syscall.MNT_DETACH,
		)

		if err != nil {
			log.Warn().Err(err).Str("file", path).Msg("failed to unmount")
		}
	}

	return os.RemoveAll(m.machineRoot(id))
}

func (m *LegacyMonitor) monitor(ctx context.Context) error {

	running, err := findAllFC()
	if err != nil {
		return err
	}

	// list all machines available under `{root}/firecracker`
	root := filepath.Join(m.root, "firecracker")
	items, err := ioutil.ReadDir(root)
	if os.IsNotExist(err) || len(items) == 0 {
		for _, pid := range running {
			syscall.Kill(pid, syscall.SIGKILL)
		}
		return errNoMoreFirecracker
	} else if err != nil {
		return err
	}

	for _, item := range items {
		if !item.IsDir() {
			continue
		}

		id := item.Name()

		if err := m.monitorID(ctx, running, id); err != nil {
			log.Err(err).Str("id", id).Msg("failed to monitor machine")
		}

	}
	return nil
}

func (m *LegacyMonitor) monitorID(ctx context.Context, running map[string]int, id string) error {
	log := log.With().Str("id", id).Logger()

	if _, ok := running[id]; ok {
		return nil
	}

	log.Debug().Msg("a legacy firecracker machine died. can't be rebooted")

	return m.cleanFsFirecracker(id)
}
