package zos

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestZMachineSRU(t *testing.T) {

	type Case struct {
		Expected gridtypes.Unit
		VM       ZMachine
	}
	cases := []Case{
		{
			Expected: 2 * gridtypes.Gigabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    1,
					Memory: 8 * gridtypes.Gigabyte,
				},
			},
		},
		{
			Expected: 500 * gridtypes.Megabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    1,
					Memory: 4 * gridtypes.Gigabyte,
				},
			},
		},
		{
			Expected: 2 * gridtypes.Gigabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    2,
					Memory: 4 * gridtypes.Gigabyte,
				},
			},
		},
		{
			Expected: 2 * gridtypes.Gigabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    3,
					Memory: 4 * gridtypes.Gigabyte,
				},
			},
		},
		{
			Expected: 2 * gridtypes.Gigabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    4,
					Memory: 5 * gridtypes.Gigabyte,
				},
			},
		},
		{
			Expected: 500 * gridtypes.Megabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    1,
					Memory: 1 * gridtypes.Gigabyte,
				},
			},
		},
		{
			Expected: 500 * gridtypes.Megabyte,
			VM: ZMachine{

				ComputeCapacity: MachineCapacity{
					CPU:    1,
					Memory: 250 * gridtypes.Megabyte,
				},
			},
		},
	}

	for _, test := range cases {
		expected := test.Expected
		vm := test.VM
		t.Run(vm.ComputeCapacity.String(), func(t *testing.T) {
			require.Equal(t, expected, vm.RootSize())
		})
	}
}
