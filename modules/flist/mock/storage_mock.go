package mock

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/threefoldtech/zosv2/modules"
)

// StorageMock is a mock object of the storage modules
type StorageMock struct {
	subvolumes map[string]string
}

// CreateFilesystem implements the modules.StorageModules interfaces
func (s *StorageMock) CreateFilesystem(name string, size uint64, poolType modules.DeviceType) (string, error) {
	path, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	if s.subvolumes == nil {
		s.subvolumes = make(map[string]string, 1)
	}
	s.subvolumes[name] = path

	return name, nil
}

// ReleaseFilesystem implements the modules.StorageModules interfaces
func (s *StorageMock) ReleaseFilesystem(name string) error {
	path, ok := s.subvolumes[name]
	if !ok {
		return nil
	}
	return os.RemoveAll(path)
}

// Path implements the modules.StorageModules interfaces
func (s *StorageMock) Path(name string) (path string, err error) {
	path, ok := s.subvolumes[name]
	if !ok {
		return "", fmt.Errorf("subvolume %s not found", name)
	}
	return path, nil
}
