package vm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

func findAll() (map[string]int, error) {
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

func find(name string) (int, error) {
	machines, err := findAll()
	if err != nil {
		return 0, err
	}

	pid, ok := machines[name]
	if !ok {
		return 0, fmt.Errorf("vm '%s' not found", name)
	}

	return pid, nil
}
