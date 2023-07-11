package capacity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDevice(t *testing.T) {
	vendor, device, ok := GetDevice(0x1002, 0x731f)

	require.True(t, ok)
	require.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI]", vendor.Name)
	require.Equal(t, "Navi 10 [Radeon RX 5600 OEM/5600 XT / 5700/5700 XT]", device.Name)
}

func TestGetSubdevice(t *testing.T) {
	subdevice, ok := GetSubdevice(0x10de, 0x1e30, 0x10de, 0x129e)

	require.True(t, ok)
	require.Equal(t, "Quadro RTX 8000", subdevice.Name)

	subdevice, ok = GetSubdevice(0x10de, 0x1e30, 0x10de, 0x12ba)

	require.True(t, ok)
	require.Equal(t, "Quadro RTX 6000", subdevice.Name)
}

func TestListPCI(t *testing.T) {
	devices, err := ListPCI()
	require.NoError(t, err)

	for _, device := range devices {
		fmt.Println(device)
	}
}

func TestListGPU(t *testing.T) {
	devices, err := ListPCI(GPU)
	require.NoError(t, err)

	for _, device := range devices {
		fmt.Println(device)
	}
}
