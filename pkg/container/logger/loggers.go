package logger

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/cio"
)

// Loggers keeps stdout and stderr backend list
type Loggers struct {
	stdouts []io.Writer
	stderrs []io.Writer
}

// Serialize dumps logs array into a json file
func Serialize(path string, logs []Logs) error {
	data, err := json.Marshal(logs)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

// Deserialize reads json from disks and returns []Logs
func Deserialize(path string) ([]Logs, error) {
	logs := []Logs{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return logs, err
	}

	if err := json.Unmarshal(data, &logs); err != nil {
		return logs, err
	}

	return logs, nil
}

// NewLoggers initialize empty lists
func NewLoggers() *Loggers {
	return &Loggers{
		stdouts: []io.Writer{},
		stderrs: []io.Writer{},
	}
}

// Add adds a defined backend on the list
func (c *Loggers) Add(stdout io.Writer, stderr io.Writer) {
	c.stdouts = append(c.stdouts, stdout)
	c.stderrs = append(c.stderrs, stderr)
}

// Stdouts returns list of stdout availables
func (c *Loggers) Stdouts() []io.Writer {
	return c.stdouts
}

// Stderrs returns list of stderr availables
func (c *Loggers) Stderrs() []io.Writer {
	return c.stderrs
}

// Log create the containers logs redirector
func (c *Loggers) Log() cio.Creator {
	mwo := io.MultiWriter(c.stdouts...)
	mwe := io.MultiWriter(c.stderrs...)

	return cio.NewCreator(cio.WithStreams(nil, mwo, mwe))
}
