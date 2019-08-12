package backend

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const lastPort = "last_reserved"

type fsStore struct {
	*FileLock
	root string
}

// NewFSStore creates a backend for port manager that stores
// the allocated port in a filesystem
func NewFSStore(root string) (Store, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	lk, err := NewFileLock(root)
	if err != nil {
		return nil, err
	}

	return &fsStore{
		FileLock: lk,
		root:     root,
	}, nil
}

func nsPath(root, ns string, port string) string {
	return filepath.Join(root, ns, port)
}

func (s *fsStore) Reserve(ns string, port int) (bool, error) {
	fname := nsPath(s.root, ns, strconv.Itoa(port))

	if err := os.MkdirAll(filepath.Dir(fname), 0755); err != nil {
		return false, err
	}

	f, err := os.OpenFile(fname, os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0660)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return false, err
	}

	// store the reserved port in lastPort file
	lastPortFile := nsPath(s.root, ns, lastPort)
	err = ioutil.WriteFile(lastPortFile, []byte(strconv.Itoa(port)), 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *fsStore) Release(ns string, port int) error {
	fname := nsPath(s.root, ns, strconv.Itoa(port))

	return os.Remove(fname)
}

func (s *fsStore) LastReserved(ns string) (int, error) {
	lastPortFile := nsPath(s.root, ns, lastPort)
	data, err := ioutil.ReadFile(lastPortFile)
	if os.IsNotExist(err) {
		return -1, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func (s *fsStore) GetByNS(ns string) ([]int, error) {
	var ports []int
	dir := filepath.Join(s.root, ns)

	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// walk through all ips in this network to get the ones which belong to a specific ID
	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		p, err := strconv.Atoi(info.Name())
		if err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}

	return ports, nil
}
