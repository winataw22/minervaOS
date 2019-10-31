package smartctl

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParseScanOutput(t *testing.T) {
	b := []byte(`
/dev/sda -d scsi # /dev/sda, SCSI device
/dev/sdb -d scsi # /dev/sdb, SCSI device`)
	devices, err := parseScan(b)
	require.NoError(t, err)

	assert.Equal(t, 2, len(devices))
	assert.Equal(t, "/dev/sda", devices[0].Path)
	assert.Equal(t, "scsi", devices[0].Type)
	assert.Equal(t, "/dev/sdb", devices[1].Path)
	assert.Equal(t, "scsi", devices[1].Type)
}

func TestParseInfo(t *testing.T) {
	b := []byte(`
	smartctl 7.0 2018-12-30 r4883 [x86_64-linux-4.14.82-Zero-OS] (local build)
Copyright (C) 2002-18, Bruce Allen, Christian Franke, www.smartmontools.org

User Capacity:        480,103,981,056 bytes [480 GB]
Logical block size:   512 bytes
LU is fully provisioned
Rotation Rate:        Solid State Device
Serial number:        E20150714150192
Device type:          disk
Local Time is:        Thu Oct 31 11:24:29 2019 UTC
SMART support is:     Unavailable - device lacks SMART capability.`)

	info, err := parseInfo(b)
	require.NoError(t, err)
	assert.Equal(t, "smartctl 7.0 2018-12-30 r4883", info.Tool)
	assert.Equal(t, "x86_64-linux-4.14.82-Zero-OS", info.Environment)
	assert.Equal(t, info.Information["User Capacity"], "480,103,981,056 bytes [480 GB]")
	assert.Equal(t, info.Information["Logical block size"], "512 bytes")
	assert.Equal(t, info.Information["Rotation Rate"], "Solid State Device")
	assert.Equal(t, info.Information["Serial number"], "E20150714150192")
	assert.Equal(t, info.Information["Device type"], "disk")
	assert.Equal(t, info.Information["SMART support is"], "Unavailable - device lacks SMART capability.")
	_, exists := info.Information["local Time is"]
	assert.False(t, exists, "Local time should not be included in information")
}
