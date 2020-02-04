package provision

import (
	schema "github.com/threefoldtech/zos/pkg/schema"
)

//TfgridReservationStatsaggregator1 jsx schema
type TfgridReservationStatsaggregator1 struct {
	Addr   string `json:"addr"`
	Port   int64  `json:"port"`
	Secret string `json:"secret"`
}

//TfgridReservationResult1 jsx schema
type TfgridReservationResult1 struct {
	Category   TfgridReservationResult1CategoryEnum `json:"category"`
	WorkloadID string                               `json:"workload_id"`
	DataJSON   string                               `json:"data_json"`
	Signature  []byte                               `json:"signature"`
	State      TfgridReservationResult1StateEnum    `json:"state"`
	Message    string                               `json:"message"`
	Epoch      schema.Date                          `json:"epoch"`
}

//TfgridReservationResult1CategoryEnum jsx schema
type TfgridReservationResult1CategoryEnum uint8

//TfgridReservationResult1CategoryEnum
const (
	TfgridReservationResult1CategoryZdb TfgridReservationResult1CategoryEnum = iota
	TfgridReservationResult1CategoryContainer
	TfgridReservationResult1CategoryNetwork
	TfgridReservationResult1CategoryVolume
)

//String implements Stringer interface
func (e TfgridReservationResult1CategoryEnum) String() string {
	switch e {
	case TfgridReservationResult1CategoryZdb:
		return "zdb"
	case TfgridReservationResult1CategoryContainer:
		return "container"
	case TfgridReservationResult1CategoryNetwork:
		return "network"
	case TfgridReservationResult1CategoryVolume:
		return "volume"
	}
	return "UNKNOWN"
}

//TfgridReservationResult1StateEnum jsx schema
type TfgridReservationResult1StateEnum uint8

//TfgridReservationResult1StateEnum
const (
	TfgridReservationResult1StateError TfgridReservationResult1StateEnum = iota
	TfgridReservationResult1StateOk
	TfgridReservationResult1StateDeleted
)

func (e TfgridReservationResult1StateEnum) String() string {
	switch e {
	case TfgridReservationResult1StateError:
		return "error"
	case TfgridReservationResult1StateOk:
		return "ok"
	case TfgridReservationResult1StateDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}
