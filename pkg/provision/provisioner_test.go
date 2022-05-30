package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestBuildResult(t *testing.T) {
	type Case struct {
		in  error
		out gridtypes.Result
	}
	cases := []Case{
		{
			in: nil,
			out: gridtypes.Result{
				State: gridtypes.StateOk,
				Error: "",
			},
		},
		{
			in: Ok(),
			out: gridtypes.Result{
				State: gridtypes.StateOk,
				Error: "",
			},
		},
		{
			in: fmt.Errorf("something went wrong"),
			out: gridtypes.Result{
				State: gridtypes.StateError,
				Error: "something went wrong",
			},
		},
		{
			in: UnChanged(fmt.Errorf("failed to update")),
			out: gridtypes.Result{
				State: gridtypes.StateUnChanged,
				Error: "failed to update",
			},
		},
		{
			in: errors.Wrap(UnChanged(fmt.Errorf("failed to update")), "wrapped"),
			out: gridtypes.Result{
				State: gridtypes.StateUnChanged,
				Error: "wrapped: failed to update",
			},
		},
		{
			in: Paused(),
			out: gridtypes.Result{
				State: gridtypes.StatePaused,
				Error: "paused",
			},
		},
		{
			in: errors.Wrap(Paused(), "wrapped for some reason"),
			out: gridtypes.Result{
				State: gridtypes.StatePaused,
				Error: "wrapped for some reason: paused",
			},
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			result, err := buildResult(nil, c.in)
			require.NoError(t, err)

			require.Equal(t, c.out.State, result.State)
			require.Equal(t, c.out.Error, result.Error)
		})
	}
}

var (
	testWorkloadType gridtypes.WorkloadType = "test"
)

type testManagerFull struct {
	mock.Mock
}

func (t *testManagerFull) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	args := t.Called(ctx, wl)
	return args.Get(0), args.Error(1)
}

func (t *testManagerFull) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	args := t.Called(ctx, wl)
	return args.Error(0)
}

func TestProvision(t *testing.T) {
	require := require.New(t)
	var mgr testManagerFull
	provisioner := NewMapProvisioner(map[gridtypes.WorkloadType]Manager{
		testWorkloadType: &mgr,
	})

	ctx := context.Background()
	wl := gridtypes.WorkloadWithID{
		Workload: &gridtypes.Workload{
			Type: testWorkloadType,
		},
	}

	mgr.On("Provision", mock.Anything, &wl).Return(123, nil)
	result, err := provisioner.Provision(ctx, &wl)

	require.NoError(err)
	require.Equal(gridtypes.StateOk, result.State)
	require.Equal(json.RawMessage("123"), result.Data)

	mgr.ExpectedCalls = nil
	mgr.On("Provision", mock.Anything, &wl).Return(nil, fmt.Errorf("failed to run"))
	result, err = provisioner.Provision(ctx, &wl)

	require.NoError(err)
	require.Equal(gridtypes.StateError, result.State)
	require.Equal("failed to run", result.Error)

	mgr.ExpectedCalls = nil
	mgr.On("Pause", mock.Anything, &wl).Return(nil, nil)
	result, err = provisioner.Pause(ctx, &wl)

	require.Errorf(err, "can only pause workloads in ok state")

	mgr.ExpectedCalls = nil
	wl = gridtypes.WorkloadWithID{
		Workload: &gridtypes.Workload{
			Type: testWorkloadType,
			Result: gridtypes.Result{
				State: gridtypes.StateOk,
			},
		},
	}

	// not here paused will set the right state even if manager
	// does not support this state.
	mgr.On("Pause", mock.Anything, &wl).Return(nil, nil)
	result, err = provisioner.Pause(ctx, &wl)
	require.NoError(err)
	require.Equal(gridtypes.StatePaused, result.State)
}
