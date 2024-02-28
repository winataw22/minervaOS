package pkg

import (
	"io"
	"io/fs"
	"os"
	"syscall"

	"github.com/stretchr/testify/mock"
)

// instance of default filesystem
var DefaultSystemOS = &defaultFileSystem{}

// SystemOS interface to mock actual OS operations.
type SystemOS interface {
	Create(string) (io.ReadCloser, error)
	MkdirAll(string, fs.FileMode) error
	Mkdir(string, fs.FileMode) error
	Stat(string) (fs.FileInfo, error)
	RemoveAll(string) error
	IsNotExist(error) bool
}

type defaultFileSystem struct{}

func (dfs *defaultFileSystem) Create(name string) (io.ReadCloser, error) {
	return os.Create(name)
}

func (dfs *defaultFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (dfs *defaultFileSystem) Mkdir(name string, perm fs.FileMode) error {
	return os.Mkdir(name, perm)
}

func (dfs *defaultFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (dfs *defaultFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (dfs *defaultFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

var DefaultSysCall = &defaultSysCall{}

type SystemCall interface {
	Mount(string, string, string, uintptr, string) error
}

type defaultSysCall struct{}

func (dfs *defaultSysCall) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return syscall.Mount(source, target, fstype, flags, data)
}

// SystemOSMock used to mock the filesystem object
type SystemOSMock struct {
	mock.Mock
}

func (os *SystemOSMock) Create(path string) (io.ReadCloser, error) {
	args := os.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (os *SystemOSMock) MkdirAll(path string, perm fs.FileMode) error {
	args := os.Called(path, perm)
	return args.Error(0)
}

func (os *SystemOSMock) Mkdir(path string, perm fs.FileMode) error {
	args := os.Called(path, perm)
	return args.Error(0)
}

func (os *SystemOSMock) Stat(path string) (fs.FileInfo, error) {
	args := os.Called(path)
	return args.Get(0).(fs.FileInfo), args.Error(1)
}

func (os *SystemOSMock) IsNotExist(err error) bool {
	args := os.Called(err)
	return args.Bool(0)
}

func (os *SystemOSMock) RemoveAll(path string) error {
	args := os.Called(path)
	return args.Error(0)
}

func (os *SystemOSMock) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	args := os.Called(source, target, fstype, flags, data)
	return args.Error(0)
}
