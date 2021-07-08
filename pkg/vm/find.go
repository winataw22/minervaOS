package vm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Process struct
type Process struct {
	Pid  int
	Args []string
}

// GetParam gets a value for a parm passed to process cmdline
func (p *Process) GetParam(arg string) ([]string, bool) {
	s := -1
	for i := 0; i < len(p.Args); i++ {
		if p.Args[i] == arg {
			s = i
			break
		}
	}
	if s == -1 {
		// not found
		return nil, false
	}
	s++
	var params []string
	for ; s < len(p.Args); s++ {
		param := p.Args[s]
		if strings.HasPrefix(param, "-") {
			break
		}
		if len(param) == 0 {
			continue
		}
		params = append(params, param)
	}

	return params, true
}

// FindAll finds all running cloud-hypervisor processes
func FindAll() (map[string]Process, error) {
	const (
		proc   = "/proc"
		search = "cloud-hypervisor"
		idFlag = "--log-file"
	)

	found := make(map[string]Process)
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
		args := make([]string, 0, len(parts))
		for _, p := range parts {
			args = append(args, string(p))
		}

		ps := Process{Pid: pid, Args: args}
		values, ok := ps.GetParam(idFlag)
		if !ok || len(values) == 0 {
			// could not find the --log-file flag!
			return nil
		}
		id := filepath.Base(values[0])
		found[id] = ps

		return nil
	})

	return found, err
}

// Find find CH process by vm name
func Find(name string) (Process, error) {
	machines, err := FindAll()
	if err != nil {
		return Process{}, err
	}

	ps, ok := machines[name]
	if !ok {
		return Process{}, fmt.Errorf("vm '%s' not found", name)
	}

	return ps, nil
}
