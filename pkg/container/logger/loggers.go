package logger

import (
	"io"

	"github.com/containerd/containerd/cio"
)

// Loggers keeps stdout and stderr backend list
type Loggers struct {
	stdouts []io.Writer
	stderrs []io.Writer
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

// Log create the containers logs redirector
func (c *Loggers) Log() cio.Creator {
	mwo := io.MultiWriter(c.stdouts...)
	mwe := io.MultiWriter(c.stderrs...)

	return cio.NewCreator(cio.WithStreams(nil, mwo, mwe))
}
