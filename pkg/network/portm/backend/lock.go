package backend

import (
	"os"
	"path"

	"github.com/alexflint/go-filemutex"
)

// FileLock wraps os.File to be used as a lock using flock
type FileLock struct {
	*filemutex.FileMutex
}

// NewFileLock opens file/dir at path and returns unlocked FileLock object
func NewFileLock(lockPath string) (*FileLock, error) {
	fi, err := os.Stat(lockPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		lockPath = path.Join(lockPath, "lock")
	}

	f, err := filemutex.New(lockPath)
	if err != nil {
		return nil, err
	}

	return &FileLock{f}, nil
}
