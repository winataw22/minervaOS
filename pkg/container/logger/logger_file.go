package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// FileType defines file logger type name
const FileType = "file"

// File write stdout/stderr to files
type File struct {
	target *os.File
}

// NewFile open file and prepare logs writing
func NewFile(stdout string, stderr string) (io.Writer, io.Writer, error) {
	log.Debug().Str("stdout", stdout).Str("stderr", stderr).Msg("initializing localfile logging")

	fo, err := os.Create(stdout)
	if err != nil {
		return nil, nil, err
	}

	// If stdout and stderr are the same, only one file is open
	fe := fo

	if stdout != stderr {
		fe, err = os.Create(stderr)
		if err != nil {
			return nil, nil, err
		}
	}

	fstdout := &File{
		target: fo,
	}

	fstderr := &File{
		target: fe,
	}

	return fstdout, fstderr, nil
}

// Write forwards write to underlaying layer
func (c *File) Write(data []byte) (int, error) {
	n, err := c.target.Write(data)
	if err != nil {
		log.Error().Err(err).Msg("log file write")
	}

	if n != len(data) {
		log.Error().Int("expected", len(data)).Int("written", n).Msg("log file write not complete")
	}

	return n, err
}
