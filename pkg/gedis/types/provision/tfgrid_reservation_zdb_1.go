package provision

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
)

//TfgridReservationZdb1 jsx schema
type TfgridReservationZdb1 struct {
	WorkloadID      int64                               `json:"workload_id"`
	NodeID          string                              `json:"node_id"`
	ReservationID   int64                               `json:"reservation_id"`
	Size            int64                               `json:"size"`
	Mode            TfgridReservationZdb1ModeEnum       `json:"mode"`
	Password        string                              `json:"password"`
	DiskType        TfgridReservationZdb1DiskTypeEnum   `json:"disk_type"`
	Public          bool                                `json:"public"`
	StatsAggregator []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
}

//ToProvisionType converts TfgridReservationZdb1 to provision.ZDB
func (z TfgridReservationZdb1) ToProvisionType() (provision.ZDB, error) {
	zdb := provision.ZDB{
		Size:     uint64(z.Size),
		Password: z.Password,
		Public:   z.Public,
	}
	switch z.DiskType.String() {
	case "hdd":
		zdb.DiskType = pkg.HDDDevice
	case "ssd":
		zdb.DiskType = pkg.SSDDevice
	default:
		return zdb, fmt.Errorf("device type %s not supported", z.DiskType.String())
	}

	switch z.Mode.String() {
	case "seq":
		zdb.Mode = pkg.ZDBModeSeq
	case "user":
		zdb.Mode = pkg.ZDBModeUser
	default:
		return zdb, fmt.Errorf("0-db mode %s not supported", z.Mode.String())
	}

	return zdb, nil
}

//TfgridReservationZdb1ModeEnum jsx schema
type TfgridReservationZdb1ModeEnum uint8

// TfgridReservationZdb1ModeEnum
const (
	TfgridReservationZdb1ModeSeq TfgridReservationZdb1ModeEnum = iota
	TfgridReservationZdb1ModeUser
)

func (e TfgridReservationZdb1ModeEnum) String() string {
	switch e {
	case TfgridReservationZdb1ModeSeq:
		return "seq"
	case TfgridReservationZdb1ModeUser:
		return "user"
	}
	return "UNKNOWN"
}

//TfgridReservationZdb1DiskTypeEnum jsx schema
type TfgridReservationZdb1DiskTypeEnum uint8

//TfgridReservationZdb1DiskTypeEnum
const (
	TfgridReservationZdb1DiskTypeHdd TfgridReservationZdb1DiskTypeEnum = iota
	TfgridReservationZdb1DiskTypeSsd
)

//String implement Stringer interface
func (e TfgridReservationZdb1DiskTypeEnum) String() string {
	switch e {
	case TfgridReservationZdb1DiskTypeHdd:
		return "hdd"
	case TfgridReservationZdb1DiskTypeSsd:
		return "ssd"
	}
	return "UNKNOWN"
}
