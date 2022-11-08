package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	testType1         = gridtypes.WorkloadType("type1")
	testType2         = gridtypes.WorkloadType("type2")
	testSharableType1 = gridtypes.WorkloadType("sharable1")
)

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
	gridtypes.RegisterType(testType1, TestData{})
	gridtypes.RegisterType(testType2, TestData{})
	gridtypes.RegisterSharableType(testSharableType1, TestData{})
}

func TestCreateDeployment(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}
	err = db.Create(dl)
	require.NoError(err)

	err = db.Create(dl)
	require.ErrorIs(err, provision.ErrDeploymentExists)
}

func TestCreateDeploymentWithWorkloads(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
		Workloads: []gridtypes.Workload{
			{
				Type: testType1,
				Name: "vm1",
			},
			{
				Type: testType2,
				Name: "vm2",
			},
		},
	}

	err = db.Create(dl)
	require.NoError(err)

	err = db.Create(dl)
	require.ErrorIs(err, provision.ErrDeploymentExists)

	loaded, err := db.Get(1, 10)
	require.NoError(err)
	require.Len(loaded.Workloads, 2)
}

func TestCreateDeploymentWithSharableWorkloads(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
		Workloads: []gridtypes.Workload{
			{
				Type: testType1,
				Name: "vm1",
			},
			{
				Type: testSharableType1,
				Name: "network",
			},
		},
	}

	err = db.Create(dl)
	require.NoError(err)

	dl.ContractID = 11
	err = db.Create(dl)
	require.ErrorIs(err, provision.ErrDeploymentConflict)

	require.NoError(db.Remove(1, 10, "networkd"))
	err = db.Create(dl)
	require.ErrorIs(err, provision.ErrDeploymentConflict)

}

func TestAddWorkload(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	err = db.Add(1, 10, gridtypes.Workload{Name: "vm1", Type: testType1})
	require.ErrorIs(err, provision.ErrDeploymentNotExists)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}

	err = db.Create(dl)
	require.NoError(err)

	err = db.Add(1, 10, gridtypes.Workload{Name: "vm1", Type: testType1})
	require.NoError(err)

	err = db.Add(1, 10, gridtypes.Workload{Name: "vm1", Type: testType1})
	require.ErrorIs(err, provision.ErrWorkloadExists)
}

func TestRemoveWorkload(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}

	err = db.Create(dl)
	require.NoError(err)

	err = db.Add(1, 10, gridtypes.Workload{Name: "vm1", Type: testType1})
	require.NoError(err)

	err = db.Remove(1, 10, "vm1")
	require.NoError(err)

	err = db.Add(1, 10, gridtypes.Workload{Name: "vm1", Type: testType1})
	require.NoError(err)

}

func TestTransactions(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}

	err = db.Create(dl)
	require.NoError(err)

	_, err = db.Current(1, 10, "vm1")
	require.ErrorIs(err, provision.ErrWorkloadNotExist)

	err = db.Add(1, 10, gridtypes.Workload{Name: "vm1", Type: testType1})
	require.NoError(err)

	wl, err := db.Current(1, 10, "vm1")
	require.NoError(err)
	require.Equal(gridtypes.StateInit, wl.Result.State)

	err = db.Transaction(1, 10, gridtypes.Workload{
		Type: testType1,
		Name: gridtypes.Name("wrong"), // wrong name
		Result: gridtypes.Result{
			Created: gridtypes.Now(),
			State:   gridtypes.StateOk,
		},
	})

	require.ErrorIs(err, provision.ErrWorkloadNotExist)

	err = db.Transaction(1, 10, gridtypes.Workload{
		Type: testType2, // wrong type
		Name: gridtypes.Name("vm1"),
		Result: gridtypes.Result{
			Created: gridtypes.Now(),
			State:   gridtypes.StateOk,
		},
	})

	require.ErrorIs(err, ErrInvalidWorkloadType)

	err = db.Transaction(1, 10, gridtypes.Workload{
		Type: testType1,
		Name: gridtypes.Name("vm1"),
		Result: gridtypes.Result{
			Created: gridtypes.Now(),
			State:   gridtypes.StateOk,
		},
	})

	require.NoError(err)

	wl, err = db.Current(1, 10, "vm1")
	require.NoError(err)
	require.Equal(gridtypes.Name("vm1"), wl.Name)
	require.Equal(testType1, wl.Type)
	require.Equal(gridtypes.StateOk, wl.Result.State)
}

func TestTwins(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}

	err = db.Create(dl)
	require.NoError(err)

	dl.TwinID = 2

	err = db.Create(dl)
	require.NoError(err)

	twins, err := db.Twins()
	require.NoError(err)

	require.Len(twins, 2)
	require.EqualValues(1, twins[0])
	require.EqualValues(2, twins[1])
}

func TestGet(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}

	err = db.Create(dl)
	require.NoError(err)

	require.NoError(db.Add(dl.TwinID, dl.ContractID, gridtypes.Workload{Name: "vm1", Type: testType1}))
	require.NoError(db.Add(dl.TwinID, dl.ContractID, gridtypes.Workload{Name: "vm2", Type: testType2}))

	loaded, err := db.Get(1, 10)
	require.NoError(err)

	require.EqualValues(1, loaded.Version)
	require.EqualValues(1, loaded.TwinID)
	require.EqualValues(10, loaded.ContractID)
	require.EqualValues("description", loaded.Description)
	require.EqualValues("some metadata", loaded.Metadata)
	require.Len(loaded.Workloads, 2)
}

func TestError(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	someError := fmt.Errorf("something is wrong")
	err = db.Error(1, 10, someError)
	require.ErrorIs(err, provision.ErrDeploymentNotExists)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
		Workloads: []gridtypes.Workload{
			{Name: "vm1", Type: testType1},
		},
	}

	err = db.Create(dl)
	require.NoError(err)

	err = db.Error(1, 10, someError)
	require.NoError(err)

	loaded, err := db.Get(1, 10)
	require.NoError(err)
	require.Equal(gridtypes.StateError, loaded.Workloads[0].Result.State)
	require.Equal(someError.Error(), loaded.Workloads[0].Result.Error)
}

func TestMigrate(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
		Workloads: []gridtypes.Workload{
			{
				Name: "vm1",
				Type: testType1,
				Data: json.RawMessage("null"),
				Result: gridtypes.Result{
					Created: gridtypes.Now(),
					State:   gridtypes.StateOk,
					Data:    json.RawMessage("\"hello\""),
				},
			},
			{
				Name: "vm2",
				Type: testType2,
				Data: json.RawMessage("\"input\""),
				Result: gridtypes.Result{
					Created: gridtypes.Now(),
					State:   gridtypes.StateError,
					Data:    json.RawMessage("null"),
					Error:   "some error",
				},
			},
		},
	}

	migration := db.Migration()
	err = migration.Migrate(dl)
	require.NoError(err)

	loaded, err := db.Get(1, 10)
	sort.Slice(loaded.Workloads, func(i, j int) bool {
		return loaded.Workloads[i].Name < loaded.Workloads[j].Name
	})

	require.NoError(err)
	require.EqualValues(dl, loaded)
}

func TestMigrateUnsafe(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	migration := db.Migration()

	require.False(db.unsafe)
	require.True(migration.unsafe.unsafe)
}

func TestDeleteDeployment(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
		Workloads: []gridtypes.Workload{
			{
				Type: testType1,
				Name: "vm1",
			},
			{
				Type: testType2,
				Name: "vm2",
			},
		},
	}

	err = db.Create(dl)
	require.NoError(err)

	err = db.Delete(1, 10)
	require.NoError(err)

	_, err = db.Get(1, 10)
	require.ErrorIs(err, provision.ErrDeploymentNotExists)
	deployments, err := db.ByTwin(1)
	require.NoError(err)
	require.Empty(deployments)

	err = db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(db.u32(1))
		if bucket == nil {
			return nil
		}
		return fmt.Errorf("twin bucket was not deleted")
	})
	require.NoError(err)
}

func TestDeleteDeploymentMultiple(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
		Workloads: []gridtypes.Workload{
			{
				Type: testType1,
				Name: "vm1",
			},
			{
				Type: testType2,
				Name: "vm2",
			},
		},
	}

	err = db.Create(dl)
	require.NoError(err)

	dl.ContractID = 20
	err = db.Create(dl)
	require.NoError(err)

	err = db.Delete(1, 10)
	require.NoError(err)

	_, err = db.Get(1, 10)
	require.ErrorIs(err, provision.ErrDeploymentNotExists)
	deployments, err := db.ByTwin(1)
	require.NoError(err)
	require.Len(deployments, 1)

	_, err = db.Get(1, 20)
	require.NoError(err)
}
