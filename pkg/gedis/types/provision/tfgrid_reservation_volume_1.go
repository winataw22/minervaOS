package provision

//TfgridReservationVolume1 jsx schema
type TfgridReservationVolume1 struct {
	WorkloadID      int64                               `json:"workload_id"`
	NodeID          int64                               `json:"node_id"`
	ReservationID   int64                               `json:"reservation_id"`
	Size            int64                               `json:"size"`
	Type            TfgridReservationVolume1TypeEnum    `json:"type"`
	StatsAggregator []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	FarmerTid       int64                               `json:"farmer_tid"`
}

//TfgridReservationVolume1TypeEnum jsx schema
type TfgridReservationVolume1TypeEnum uint8

//TfgridReservationVolume1TypeEnum
const (
	TfgridReservationVolume1TypeHDD TfgridReservationVolume1TypeEnum = iota
	TfgridReservationVolume1TypeSSD
)

// String implements Stringer interface
func (e TfgridReservationVolume1TypeEnum) String() string {
	switch e {
	case TfgridReservationVolume1TypeHDD:
		return "HDD"
	case TfgridReservationVolume1TypeSSD:
		return "SSD"
	}
	return "UNKNOWN"
}
