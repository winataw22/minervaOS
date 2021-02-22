package flist

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

var (
	templ = template.Must(template.New("script").Parse(`
echo $PPID > {{.pid}}
echo "mount ready" > {{.log}}
`))
)

// StorageMock is a mock object of the storage modules
type StorageMock struct {
	mock.Mock
}

// CreateFilesystem create filesystem mock
func (s *StorageMock) CreateFilesystem(name string, size uint64, poolType pkg.DeviceType) (pkg.Filesystem, error) {
	args := s.Called(name, size, poolType)
	return pkg.Filesystem{
		Path: args.String(0),
	}, args.Error(1)
}

// ReleaseFilesystem releases filesystem mock
func (s *StorageMock) ReleaseFilesystem(name string) error {
	args := s.Called(name)
	return args.Error(0)
}

// ListFilesystems list filesystem mock
func (s *StorageMock) ListFilesystems() ([]pkg.Filesystem, error) {
	args := s.Called()
	return nil, args.Error(1)
}

// Path implements the pkg.StorageModules interfaces
func (s *StorageMock) Path(name string) (pkg.Filesystem, error) {
	args := s.Called(name)
	return pkg.Filesystem{
		Path: args.String(0),
	}, args.Error(1)
}

func (s *StorageMock) CanAllocate(name string, size uint64) (bool, error) {
	args := s.Called(name, size)
	return args.Bool(0), args.Error(1)
}

// GetCacheFS return the special filesystem used by 0-OS to store internal state and flist cache
func (s *StorageMock) GetCacheFS() (pkg.Filesystem, error) {
	return pkg.Filesystem{}, nil
}

// GetVdiskFS return the filesystem used to store the vdisk file for the VM module
func (s *StorageMock) GetVdiskFS() (pkg.Filesystem, error) {
	return pkg.Filesystem{}, nil
}

type testCommander struct {
	*testing.T
	m map[string]string
}

func (t *testCommander) args(args ...string) (map[string]string, []string) {
	var lk string
	v := make(map[string]string)
	var r []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			lk = strings.TrimPrefix(arg, "-")
			v[lk] = ""
			continue
		}

		//if no -, then it must be a value.
		//so if lk is set, update, otherwise append to r
		if len(lk) > 0 {
			v[lk] = arg
			lk = ""
		} else {
			r = append(r, arg)
		}
	}

	t.m = v
	return v, r
}

func (t *testCommander) Command(name string, args ...string) *exec.Cmd {
	if name != "g8ufs" {
		t.Fatal("invalid command name, expected 'g8ufs'")
	}

	m, _ := t.args(args...)
	var script bytes.Buffer
	err := templ.Execute(&script, map[string]string{
		"pid": m["pid"],
		"log": m["log"],
	})

	require.NoError(t.T, err)

	return exec.Command("sh", "-c", script.String())
}

func TestCommander(t *testing.T) {
	cmder := testCommander{T: t}

	m, r := cmder.args("-pid", "pid-file", "-log", "log-file", "remaining")

	require.Equal(t, []string{"remaining"}, r)
	require.Equal(t, map[string]string{
		"pid": "pid-file",
		"log": "log-file",
	}, m)
}

/**
* TestMount tests that the module actually call the underlying modules an binaries
* with the correct arguments. It's mocked in a way that does not actually mount
* an flist, but emulate the call of the g8ufs binary.
* The method also validate that storage module is called as expected
 */
func TestMountUnmount(t *testing.T) {
	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root, err := ioutil.TempDir("", "flist_root")
	require.NoError(t, err)

	defer os.RemoveAll(root)

	flister := newFlister(root, strg, cmder)

	strg.On("Path", mock.Anything).Return("/my/backend", nil)

	strg.On("CreateFilesystem", mock.Anything, uint64(256*mib), zos.SSDDevice).
		Return("/my/backend", nil)

	mnt, err := flister.Mount("https://hub.grid.tf/thabet/redis.flist", "", pkg.DefaultMountOptions)
	require.NoError(t, err)

	// Trick flister into thinking that 0-fs has exited
	os.Remove(cmder.m["pid"])
	strg.On("ReleaseFilesystem", filepath.Base(mnt)).Return(nil)

	err = flister.Umount(mnt)
	require.NoError(t, err)
}

func TestIsolation(t *testing.T) {
	require := require.New(t)

	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root, err := ioutil.TempDir("", "flist_root")
	require.NoError(err)

	defer os.RemoveAll(root)

	flister := newFlister(root, strg, cmder)

	strg.On("Path", mock.Anything).Return("/my/backend", nil)

	strg.On("CreateFilesystem", mock.Anything, uint64(256*mib), zos.SSDDevice).
		Return("/my/backend", nil)

	path1, err := flister.Mount("https://hub.grid.tf/thabet/redis.flist", "", pkg.DefaultMountOptions)
	require.NoError(err)
	args1 := cmder.m

	path2, err := flister.Mount("https://hub.grid.tf/thabet/redis.flist", "", pkg.DefaultMountOptions)
	require.NoError(err)
	args2 := cmder.m

	require.NotEqual(path1, path2)
	require.NotEqual(args1, args2)

	// the 2 mounts since they are exactly the same flist
	// should have same meta, and of course same cache
	// but a different backend and pid
	require.Equal(args1["cache"], args2["cache"])
	require.Equal(args1["meta"], args2["meta"])
	//TODO: the backend url is return by the storage mock, this is why this
	// is failing.
	// require.NotEqual(args1["backend"], args2["backend"])
	require.NotEqual(args1["pid"], args2["pid"])
}

func TestDownloadFlist(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root, err := ioutil.TempDir("", "flist_root")
	require.NoError(err)
	defer os.RemoveAll(root)

	x := newFlister(root, strg, cmder)

	f, ok := x.(*flistModule)
	require.True(ok)
	path1, err := f.downloadFlist("https://hub.grid.tf/thabet/redis.flist")
	require.NoError(err)

	info1, err := os.Stat(path1)
	require.NoError(err)

	path2, err := f.downloadFlist("https://hub.grid.tf/thabet/redis.flist")
	require.NoError(err)

	assert.Equal(path1, path2)

	// mod time should be the same, this proof the second download
	// didn't actually re-wrote the file a second time
	info2, err := os.Stat(path2)
	require.NoError(err)
	assert.Equal(info1.ModTime(), info2.ModTime())

	// now corrupt the flist
	err = os.Truncate(path1, 512)
	require.NoError(err)

	path3, err := f.downloadFlist("https://hub.grid.tf/thabet/redis.flist")
	require.NoError(err)

	info3, err := os.Stat(path3)
	require.NoError(err)
	assert.NotEqual(info2.ModTime(), info3.ModTime())
}

func TestWaitPIDFileExists(t *testing.T) {
	require := require.New(t)
	const testFile = "/tmp/wait.exists.test"
	os.Remove(testFile)

	out := make(chan error)
	go func(out chan<- error) {
		out <- waitPidFile(2*time.Second, testFile, true)
	}(out)

	os.Create(testFile)
	err := <-out
	require.NoError(err)
}

func TestWaitPIDFileDeleted(t *testing.T) {
	require := require.New(t)
	const testFile = "/tmp/wait.deleted.test"
	os.Create(testFile)

	out := make(chan error)
	go func(out chan<- error) {
		out <- waitPidFile(2*time.Second, testFile, false)
	}(out)

	os.Remove(testFile)
	err := <-out
	require.NoError(err)
}

func TestWaitPIDFileTimeout(t *testing.T) {
	require := require.New(t)
	const testFile = "/tmp/wait.deleted.test"
	os.Create(testFile)

	out := make(chan error)
	go func(out chan<- error) {
		out <- waitPidFile(1*time.Second, testFile, false)
	}(out)

	err := <-out
	require.Equal(context.DeadlineExceeded, err)
}

func Test_forceStop(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

	cmd := exec.CommandContext(ctx, "sleep", "50")
	err := cmd.Start()
	require.NoError(t, err)

	go func() {
		cmd.Wait()
	}()

	go func() {
		<-ctx.Done()
		t.Error("didn't stop the process in time")
	}()

	err = forceStop(cmd.Process.Pid)
	assert.NoError(t, err)
}
