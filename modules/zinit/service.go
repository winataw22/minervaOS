package zinit

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// InitService represent a Zinit service file
type InitService struct {
	Exec    string
	Oneshot bool
	After   []string
}

// AddService write the service into a file in the expected location for Zinit
// you usually want to call Monitor(name) after adding a service
func AddService(name string, service InitService) error {
	b, err := yaml.Marshal(service)
	if err != nil {
		return err
	}
	path := filepath.Join("/etc/zinit", fmt.Sprintf("%s.yaml", name))
	return ioutil.WriteFile(path, b, 0660)
}
