package provision

//TfgridReservationZdb1 jsx schema
type TfgridReservationZdb1 struct {
	WorkloadID      int64                               `json:"workload_id"`
	NodeID          int64                               `json:"node_id"`
	ReservationID   int64                               `json:"reservation_id"`
	Size            int64                               `json:"size"`
	Mode            TfgridReservationZdb1ModeEnum       `json:"mode"`
	Password        string                              `json:"password"`
	DiskType        TfgridReservationZdb1DiskTypeEnum   `json:"disk_type"`
	Public          bool                                `json:"public"`
	StatsAggregator []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	FarmerTid       int64                               `json:"farmer_tid"`
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
