package storage

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

var TestType = gridtypes.WorkloadType("test")

type TestData struct{}

func (t TestData) Valid(getter gridtypes.WorkloadGetter) error {
	return nil
}

func (t TestData) Challenge(w io.Writer) error {
	return nil
}

func (t TestData) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{}, nil
}

func init() {
	gridtypes.RegisterType(TestType, TestData{})
}
func TestStorageAdd(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)

	twin := uint32(1)
	id := uint32(1)
	err = store.Add(gridtypes.Deployment{
		TwinID:       twin,
		DeploymentID: id,
		Metadata:     "meta",
		Description:  "descriptions",
		Workloads: []gridtypes.Workload{
			{
				Name: "volume",
				Type: TestType,
				Data: gridtypes.MustMarshal(TestData{}),
			},
		},
	})

	require.NoError(err)
	stat, err := os.Lstat(filepath.Join(root, fmt.Sprint(twin), fmt.Sprint(id)))
	require.NoError(err)
	require.True(stat.Mode().IsRegular())
}

func TestStorageSet(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)

	twin := uint32(1)
	id := uint32(1)
	deployment := gridtypes.Deployment{
		TwinID:       twin,
		DeploymentID: id,
		Metadata:     "meta",
		Description:  "descriptions",
		Workloads: []gridtypes.Workload{
			{
				Name: "volume",
				Type: TestType,
				Data: gridtypes.MustMarshal(TestData{}),
			},
		},
	}

	err = store.Set(deployment)

	require.Error(err)
	require.True(errors.Is(err, provision.ErrDeploymentNotExists))

	err = store.Add(deployment)
	require.NoError(err)

	err = store.Set(deployment)
	require.NoError(err)
}

func TestStorageGet(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)
	twin := uint32(1)
	id := uint32(1)
	deployment := gridtypes.Deployment{
		TwinID:       twin,
		DeploymentID: id,
		Metadata:     "meta",
		Description:  "descriptions",
		Workloads: []gridtypes.Workload{
			{
				Name: "volume",
				Type: TestType,
				Data: gridtypes.MustMarshal(TestData{}),
			},
		},
	}

	err = store.Add(deployment)
	require.NoError(err)

	loaded, err := store.Get(deployment.TwinID, deployment.DeploymentID)
	require.NoError(err)
	require.Equal(deployment.Description, loaded.Description)
	require.Equal(deployment.Metadata, loaded.Metadata)
	require.Equal(len(deployment.Workloads), len(deployment.Workloads))
}

func TestStorageByTwin(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)

	deployment1 := gridtypes.Deployment{
		TwinID:       1,
		DeploymentID: 1,
		Metadata:     "meta",
		Description:  "descriptions",
		Workloads: []gridtypes.Workload{
			{
				Name: "volume",
				Type: TestType,
				Data: gridtypes.MustMarshal(TestData{}),
			},
		},
	}

	err = store.Add(deployment1)
	require.NoError(err)

	deployment2 := gridtypes.Deployment{
		TwinID:       1,
		DeploymentID: 2,
		Metadata:     "meta",
		Description:  "descriptions",
		Workloads: []gridtypes.Workload{
			{
				Name: "volume",
				Type: TestType,
				Data: gridtypes.MustMarshal(TestData{}),
			},
		},
	}

	err = store.Add(deployment2)
	require.NoError(err)

	deployment3 := gridtypes.Deployment{
		TwinID:       2,
		DeploymentID: 1,
		Metadata:     "meta",
		Description:  "descriptions",
		Workloads: []gridtypes.Workload{
			{
				Name: "volume",
				Type: TestType,
				Data: gridtypes.MustMarshal(TestData{}),
			},
		},
	}

	err = store.Add(deployment3)
	require.NoError(err)

	ids, err := store.ByTwin(1)
	require.NoError(err)
	require.Len(ids, 2)

	ids, err = store.ByTwin(2)
	require.NoError(err)
	require.Len(ids, 1)

}
