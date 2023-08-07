package provision

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// func TestEngine(t *testing.T) {
// 	td, err := ioutil.TempDir("", "")
// 	require.NoError(t, err)
// 	defer t.Cleanup(func() {
// 		os.RemoveAll(td)
// 	})

// 	nodeID := "BhPhHVhfU8qerzzh1BGBgcQ7SQxQtND3JwuxSPoRzqkY"
// 	store := &FSStore{
// 		root: td,
// 	}

// 	engine := &defaultEngine{
// 		nodeID: nodeID,
// 		store:  store,
// 	}

// 	mustJSONMarshal := func(v interface{}) []byte {
// 		b, err := json.Marshal(v)
// 		require.NoError(t, err)
// 		return b
// 	}

// 	err = engine.store.Add(&Reservation{
// 		ID:   "1-1",
// 		Type: VolumeReservation,
// 		Data: mustJSONMarshal(Volume{
// 			Size: 1,
// 			Type: HDDDiskType,
// 		}),
// 	})
// 	require.NoError(t, err)

// 	err = engine.store.Add(&Reservation{
// 		ID:   "3-1",
// 		Type: ZDBReservation,
// 		Data: mustJSONMarshal(ZDB{
// 			Size:     15,
// 			Mode:     pkg.ZDBModeSeq,
// 			DiskType: pkg.SSDDevice,
// 		}),
// 	})
// 	require.NoError(t, err)

// 	err = engine.store.Add(&Reservation{
// 		ID:   "4-1",
// 		Type: ContainerReservation,
// 		Data: mustJSONMarshal(Container{
// 			Capacity: ContainerCapacity{
// 				CPU:    2,
// 				Memory: 4096,
// 			},
// 		}),
// 	})
// 	require.NoError(t, err)

// 	resources, workloads := engine.capacityUsed()
// 	assert.Equal(t, uint64(2), resources.Cru)
// 	assert.Equal(t, float64(4), resources.Mru)
// 	assert.Equal(t, float64(15.25), resources.Sru)
// 	assert.Equal(t, float64(1), resources.Hru)

// 	assert.EqualValues(t, 1, workloads.Container)
// 	assert.EqualValues(t, 0, workloads.Network)
// 	assert.EqualValues(t, 1, workloads.Volume)
// 	assert.EqualValues(t, 1, workloads.ZDBNamespace)
// 	assert.EqualValues(t, 0, workloads.K8sVM)
// }

func TestGetZmountSize(t *testing.T) {
	t.Run("invalid type", func(t *testing.T) {
		wl := gridtypes.WorkloadWithID{
			Workload: &gridtypes.Workload{Type: "invalid"},
		}
		_, err := getZmountSize(wl.Workload)
		assert.Error(t, err)
	})
	t.Run("different data type", func(t *testing.T) {
		wl := gridtypes.WorkloadWithID{
			Workload: &gridtypes.Workload{Type: zos.ZDBType, Data: json.RawMessage(`{"size": 10}`)},
		}
		_, err := getZmountSize(wl.Workload)
		assert.Error(t, err)
	})
	t.Run("valid data", func(t *testing.T) {
		wl := gridtypes.WorkloadWithID{
			Workload: &gridtypes.Workload{Type: zos.ZMountType, Data: json.RawMessage(`{"size": 10}`)},
		}
		size, err := getZmountSize(wl.Workload)
		assert.NoError(t, err)
		assert.Equal(t, size, gridtypes.Unit(10))
	})

}

func TestSortZmountWorkloads(t *testing.T) {
	workloads := []*gridtypes.WorkloadWithID{
		{Workload: &gridtypes.Workload{
			Type: zos.ZMountType,
			Data: json.RawMessage(`{"size": 10}`),
		}},
		{Workload: &gridtypes.Workload{
			Type: zos.ZMountType,
			Data: json.RawMessage(`{"size": 30}`),
		}},
		{Workload: &gridtypes.Workload{
			Type: zos.ZMountType,
			Data: json.RawMessage(`{"size": 20}`),
		}},
	}

	expectedWorkloads := []*gridtypes.WorkloadWithID{
		{Workload: &gridtypes.Workload{
			Type: zos.ZMountType,
			Data: json.RawMessage(`{"size": 30}`),
		}},
		{Workload: &gridtypes.Workload{
			Type: zos.ZMountType,
			Data: json.RawMessage(`{"size": 20}`),
		}},
		{Workload: &gridtypes.Workload{
			Type: zos.ZMountType,
			Data: json.RawMessage(`{"size": 10}`),
		}},
	}

	sortZmountWorkloads(workloads)
	assert.Equal(t, expectedWorkloads, workloads)
}
