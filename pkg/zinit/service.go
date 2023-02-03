package zinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// LogType is an enum type
type LogType string

// UnmarshalYAML implements the  yaml.Unmarshaler interface
func (s *LogType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	if err := unmarshal(&buf); err != nil {
		return err
	}
	*s = LogType(strings.ToLower(buf))
	return nil
}

// All the type of logging supported by zinit
const (
	StdoutLogType LogType = "stdout"
	RingLogType   LogType = "ring"
	NoneLogType   LogType = "none"
)

// InitService represent a Zinit service file
type InitService struct {
	Exec    string            `yaml:"exec,omitempty"`
	Oneshot bool              `yaml:"oneshot,omitempty"`
	Test    string            `yaml:"test,omitempty"`
	After   []string          `yaml:"after,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	Log     LogType           `yaml:"log,omitempty"`
}

// AddService write the service into a file in the expected location for Zinit
// you usually want to call Monitor(name) after adding a service
func AddService(name string, service InitService) error {
	b, err := yaml.Marshal(service)
	if err != nil {
		return err
	}
	path := filepath.Join("/etc/zinit", fmt.Sprintf("%s.yaml", name))
	return os.WriteFile(path, b, 0660)
}

// RemoveService delete the service file from the filesystem
// make sure the service has been stopped and forgot before deleting it
func RemoveService(name string) error {
	path := filepath.Join("/etc/zinit", fmt.Sprintf("%s.yaml", name))
	return os.RemoveAll(path)
}
