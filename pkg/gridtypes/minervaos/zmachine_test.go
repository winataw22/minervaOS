package zos

import (
	"encoding/json"
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

func TestResultDeprecated(t *testing.T) {
	raw := ` {
		"id": "192-74881-testing2",
		"ip": "10.20.2.2",
		"ygg_ip": "32b:8310:9b03:5529:ff0f:37cd:de80:b322",
		"console_url": "10.20.2.0:20002"
	  }`

	var result ZMachineResult

	err := json.Unmarshal([]byte(raw), &result)
	require.NoError(t, err)

	require.EqualValues(t, ZMachineResult{
		ID:          "192-74881-testing2",
		IP:          "10.20.2.2",
		PlanetaryIP: "32b:8310:9b03:5529:ff0f:37cd:de80:b322",
		ConsoleURL:  "10.20.2.0:20002",
	}, result)
}
