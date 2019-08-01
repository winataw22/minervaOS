package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/flist"
	"github.com/threefoldtech/zosv2/modules/flist/mock"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

var (
	rpcTest = flag.Bool("rpc", false, "run RPC tests")
)

func TestMain(m *testing.M) {
	flag.Parse()

	os.Exit(m.Run())
}

func testFlistModule(t *testing.T) (modules.Flister, func()) {
	root, err := ioutil.TempDir("", "flist_root")
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(root)
	}
	return flist.New(root, &mock.StorageMock{}), cleanup
}

func testPrepareRPC(t *testing.T) (modules.Flister, func()) {
	const redisAddr = "tcp://localhost:6379"

	f, cleanup := testFlistModule(t)

	server, err := zbus.NewRedisServer("flist", redisAddr, 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	server.Register(zbus.ObjectID{Name: "flist", Version: "0.0.1"}, f)
	go func() {
		err := server.Run(context.Background())
		require.NoError(t, err)
	}()

	client, err := zbus.NewRedisClient(redisAddr)
	require.NoError(t, err)

	flist := stubs.NewFlisterStub(client)
	return flist, cleanup
}

func TestRPCMountUmount(t *testing.T) {
	// only run these test when rpc flag is passed
	if !*rpcTest {
		t.SkipNow()
	}

	require := require.New(t)
	assert := assert.New(t)

	flist, cleanup := testPrepareRPC(t)
	defer cleanup()

	path, err := flist.Mount("https://hub.grid.tf/zaibon/ubuntu_bionic.flist", "")
	require.NoError(err)

	// check file are accessible
	assert.NotEqual("", path)
	infos, err := ioutil.ReadDir(path)
	require.NoError(err)
	assert.True(len(infos) > 0)

	// check pid file exists
	dir, name := filepath.Split(path)
	dir, _ = filepath.Split(dir[:len(dir)-1])
	pidPath := filepath.Join(dir, "pid", name) + ".pid"
	_, err = os.Stat(pidPath)
	assert.NoError(err)

	err = flist.Umount(path)
	assert.NoError(err)

	// check pid file is gone after unmount
	_, err = os.Stat(pidPath)
	assert.Error(err)
	assert.True(os.IsNotExist(err))
}
