package filesystem

import (
	"testing"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fsString = `Label: 'two'  uuid: 5efab9c9-55d8-4f0f-b5b9-4c521b567c70
	Total devices 2 FS bytes used 294912
	devid    1 size 1048576000 used 218103808 path /dev/loop1
	devid    2 size 1048576000 used 218103808 path /dev/loop2

Label: 'single'  uuid: 74595911-0f79-4c2e-925f-105d1279fb48
	Total devices 1 FS bytes used 196608
    devid    1 size 1048576000 used 138412032 path /dev/loop3
`
	fsString2 = `Label: 'sp_zos-cache'  uuid: e8784776-6288-49e2-9cb0-29e50707bd73
	Total devices 1 FS bytes used 544014336
	devid    1 size 5366611968 used 5354029056 path /dev/vdc1

Label: 'dmdm'  uuid: 7977d80d-f9c9-4343-82d3-018bc54698b1
	Total devices 2 FS bytes used 114688
	devid    1 size 5368709120 used 1619001344 path /dev/vdd
	devid    2 size 5368709120 used 1619001344 path /dev/vde
`

	fsStringWithWarnings = `
	Label: 'sp_zos-cache'  uuid: c739f6f4-a02b-429c-9ffa-7899b65ad566
	Total devices 1 FS bytes used 262144
	devid    1 size 1071644672 used 132251648 path /dev/vda1

warning, device 2 is missing
Label: 'dsds'  uuid: 70059ae1-6b5a-4e44-a4e2-13cabc10b8bf
	Total devices 2 FS bytes used 114688
	devid    1 size 5368709120 used 1619001344 path /dev/vdf
	*** Some devices missing

`
	fsStringWithWarnings2 = `warning, device 2 is missing
	warning, device 2 is missing
	Label: 'sp_zos-cache'  uuid: c739f6f4-a02b-429c-9ffa-7899b65ad566
	Total devices 1 FS bytes used 262144
	devid    1 size 1071644672 used 132251648 path /dev/vda1

Label: 'dsds'  uuid: 70059ae1-6b5a-4e44-a4e2-13cabc10b8bf
	Total devices 2 FS bytes used 114688
	devid    1 size 5368709120 used 1619001344 path /dev/vdf
	*** Some devices missing


	Label: 'dmdm'  uuid: 1a6c5498-758f-4490-add2-e151a7bad1de
		Total devices 2 FS bytes used 132251648
		devid    1 size 5368709120 used 1619001344 path /dev/vdd
		*** Some devices missing
	`
)

func TestParseFS(t *testing.T) {

	fss, err := parseList(fsString)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(fss))

	fs := fss[0]
	assert.Nil(t, err)
	assert.Equal(t, "two", fs.Label)
	assert.Equal(t, "5efab9c9-55d8-4f0f-b5b9-4c521b567c70", fs.UUID)
	assert.Equal(t, 2, fs.TotalDevices)
	assert.Equal(t, int64(294912), fs.Used)

	assert.Equal(t, 2, len(fs.Devices))
	dev := fs.Devices[0]
	assert.Equal(t, 1, dev.DevID)
	assert.Equal(t, int64(1048576000), dev.Size)
	assert.Equal(t, int64(218103808), dev.Used)
	assert.Equal(t, "/dev/loop1", dev.Path)
}

func TestParseFS2(t *testing.T) {
	fss, err := parseList(fsString2)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(fss))

	fs := fss[0]
	assert.Equal(t, 1, fs.TotalDevices)

	fs = fss[1]
	assert.Equal(t, 2, fs.TotalDevices)
}

func TestParseFSWithWarnings(t *testing.T) {
	fss, err := parseList(fsStringWithWarnings)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(fss))

	fs := fss[0]
	assert.Nil(t, err)
	assert.Equal(t, "sp_zos-cache", fs.Label)
	assert.Equal(t, "c739f6f4-a02b-429c-9ffa-7899b65ad566", fs.UUID)
	assert.Equal(t, 1, fs.TotalDevices)

	fs = fss[1]
	assert.Equal(t, "dsds", fs.Label)
	assert.Equal(t, "70059ae1-6b5a-4e44-a4e2-13cabc10b8bf", fs.UUID)
	assert.Equal(t, 2, fs.TotalDevices)
	assert.Equal(t, 1, len(fs.Devices))
	dev := fs.Devices[0]
	assert.Equal(t, 1, dev.DevID)
	assert.Equal(t, "/dev/vdf", dev.Path)
}

func TestParseFSWithWarnings2(t *testing.T) {
	fss, err := parseList(fsStringWithWarnings2)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(fss))

	fs := fss[0]
	assert.Nil(t, err)
	assert.Equal(t, "sp_zos-cache", fs.Label)
	assert.Equal(t, "c739f6f4-a02b-429c-9ffa-7899b65ad566", fs.UUID)
	assert.Equal(t, 1, fs.TotalDevices)

	fs = fss[1]
	assert.Equal(t, "dsds", fs.Label)
	assert.Equal(t, "70059ae1-6b5a-4e44-a4e2-13cabc10b8bf", fs.UUID)
	assert.Equal(t, 2, fs.TotalDevices)
	assert.Equal(t, 1, len(fs.Devices))
	dev := fs.Devices[0]
	assert.Equal(t, 1, dev.DevID)
	assert.Equal(t, "/dev/vdf", dev.Path)

	fs = fss[2]
	assert.Equal(t, "dmdm", fs.Label)
	assert.Equal(t, "1a6c5498-758f-4490-add2-e151a7bad1de", fs.UUID)
	assert.Equal(t, 2, fs.TotalDevices)
	assert.Equal(t, 1, len(fs.Devices))
	dev = fs.Devices[0]
	assert.Equal(t, 1, dev.DevID)
	assert.Equal(t, "/dev/vdd", dev.Path)

}

func TestParseDF(t *testing.T) {
	const dfString = `Data, single: total=8388608, used=65536
System, single: total=4194304, used=16384
Metadata, single: total=276824064, used=163840
GlobalReserve, single: total=16777216, used=0
	`

	df, err := parseFilesystemDF(dfString)
	require.NoError(t, err)

	assert.Equal(t, BtrfsDiskUsage{
		Data: DiskUsage{
			Profile: modules.Single, Total: 8388608, Used: 65536,
		},
		System: DiskUsage{
			Profile: modules.Single, Total: 4194304, Used: 16384,
		},
		Metadata: DiskUsage{
			Profile: modules.Single, Total: 276824064, Used: 163840,
		},
		GlobalReserve: DiskUsage{
			Profile: modules.Single, Total: 16777216, Used: 0,
		},
	}, df)

}

func TestParseSubvolume(t *testing.T) {
	const subvolStr = `ID 259 gen 14 top level 5 path svol
	ID 262 gen 21 top level 5 path cobavol

	`

	svs, err := parseSubvolList(subvolStr)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(svs))

	sv := svs[0]
	assert.Equal(t, sv.ID, 259)
	assert.Equal(t, sv.Generation, 14)
	assert.Equal(t, sv.ParentID, 5)
	assert.Equal(t, sv.Path, "svol")
}

func TestBtrfsSubvolumeInfo(t *testing.T) {
	const input = `
a/b
	Name: 			b
	UUID: 			4dd82d3b-71fe-fb4b-9f5d-72d6a83e3bb6
	Parent UUID: 		-
	Received UUID: 		-
	Creation time: 		2019-06-18 12:05:53 +0200
	Subvolume ID: 		263
	Generation: 		24
	Gen at creation: 	24
	Parent ID: 		261
	Top level ID: 		261
	Flags: 			-
	Snapshot(s):
`

	volume, err := parseSubvolInfo(input)
	require.NoError(t, err)

	assert.Equal(t, BtrfsVolume{ID: 263, ParentID: 261, Generation: 24}, volume)
}

func TestQGroupParse(t *testing.T) {
	const s = `
qgroupid         rfer         excl     max_rfer     max_excl
--------         ----         ----     --------     --------
0/5             16384        16384         none         none
0/257           16384        16384         none         none
0/258         5804032      5804032   1073741824         none
0/259           16384        16384         none         none
	`

	groups := parseQGroups(s)
	assert.Len(t, groups, 4)

	g, ok := groups["0/258"]
	assert.True(t, ok)

	assert.Equal(t, BtrfsQGroup{
		ID:      "0/258",
		Rfer:    5804032,
		Excl:    5804032,
		MaxRfer: 1073741824,
		MaxExcl: 0,
	}, g)
}
