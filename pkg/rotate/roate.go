/**
rotate package provides a very simple tool to truncate a given file to 0 after it copies
the last *configurable* part of this file to a new file with suffix .tail

The idea is that services need to have their log files (or redirection) be open in append
mode. So truncation of the log file should be enough.

There is no grantee that some logs will be lost between the copying of the file tail and the
truncation of the file.
*/
package rotate

import (
	"fmt"
	"io"
	"os"
)

const (
	Bytes     Size = 1
	Kilobytes Size = 1024 * Bytes
	Megabytes Size = 1024 * Kilobytes
	Gigabyte  Size = 1024 * Megabytes

	suffix = ".0"
)

var (
	defaultRotator = Rotator{
		maxsize:  20 * Megabytes,
		tailsize: 10 * Megabytes,
		suffix:   suffix,
	}
)

type Size int64

type Option interface {
	apply(cfg *Rotator)
}

type optFn func(cfg *Rotator)

func (fn optFn) apply(cfg *Rotator) {
	fn(cfg)
}

// MaxSize of the file maximum size. If file size is bigger
// than this value, it will be truncated. Default to 20MB
func MaxSize(size Size) Option {
	var fn optFn = func(cfg *Rotator) {
		cfg.maxsize = size
	}

	return fn
}

// TailSize sets the size of the tail chunk that will be kept before
// the truncation. If value is bigger than MaxSize, it will be set to MaxSize.
// default to 10M
func TailSize(size Size) Option {
	var fn optFn = func(cfg *Rotator) {
		cfg.tailsize = size
	}

	return fn
}

// Suffix of the tail before truncation, default to '.0'
func Suffix(suffix string) Option {
	if suffix == "" {
		panic("suffix cannot be empty")
	}

	var fn optFn = func(cfg *Rotator) {
		cfg.suffix = suffix
	}

	return fn
}

type Rotator struct {
	// maxsize is the max file size. If file size exceeds maxsize rotation is applied
	// otherwise file is not touched
	maxsize Size
	// tailsize is size of the chunk to keep with Suffix before truncation of the file.
	// Defaults to MaxSize
	tailsize Size

	// suffix of the tail chunk, default to suffix
	suffix string
}

func NewRotator(opt ...Option) Rotator {
	cfg := defaultRotator
	for _, o := range opt {
		o.apply(&cfg)
	}
	if cfg.tailsize > cfg.maxsize {
		cfg.tailsize = cfg.maxsize
	}

	return cfg
}

func (r *Rotator) Rotate(file string) error {
	fd, err := os.OpenFile(file, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", file, err)
	}

	defer fd.Close()
	stat, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	if stat.Size() <= int64(r.maxsize) {
		return nil
	}

	//otherwise we move seek to the end of the file
	if _, err := fd.Seek(-int64(r.tailsize), 2); err != nil {
		return fmt.Errorf("failed to seek to truncate position: %w", err)
	}

	tail := file + r.suffix
	tailFd, err := os.Create(tail)
	if err != nil {
		return fmt.Errorf("failed to create tail file '%s': %w", tail, err)
	}

	if _, err := io.Copy(tailFd, fd); err != nil {
		return fmt.Errorf("failed to copy log tail: %w", err)
	}

	return fd.Truncate(0)
}
