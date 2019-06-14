package zinit

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type InitService struct {
	Exec    string
	Oneshot bool
	After   []string
}

func AddService(name string, service InitService) error {
	b, err := yaml.Marshal(service)
	if err != nil {
		return err
	}
	path := filepath.Join("/etc/zinit", fmt.Sprintf("%s.yaml", name))
	return ioutil.WriteFile(path, b, 0660)
}
