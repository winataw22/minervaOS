package provision

import (
	"encoding/json"

	"github.com/threefoldtech/zos/pkg/schema"
)

//TfgridReservationWorkload1 jsx schema
type TfgridReservationWorkload1 struct {
	WorkloadID string                             `json:"workload_id,omitempty"`
	Type       TfgridReservationWorkload1TypeEnum `json:"type,omitempty"`
	Workload   json.RawMessage                    `json:"content,omitempty"`
	User       string                             `json:"user,omitempty"`
	Created    schema.Date                        `json:"created,omitempty"`
	Duration   int64                              `json:"duration,omitempty"`
	Signature  string                             `json:"signature,omitempty"`
	ToDelete   bool                               `json:"to_delete,omitempty"`
}

//TfgridReservationWorkload1TypeEnum jsx schema
type TfgridReservationWorkload1TypeEnum uint8

// TfgridReservationWorkload1TypeEnum
const (
	TfgridReservationWorkload1TypeZdb TfgridReservationWorkload1TypeEnum = iota
	TfgridReservationWorkload1TypeContainer
	TfgridReservationWorkload1TypeVolume
	TfgridReservationWorkload1TypeNetwork
	TfgridReservationWorkload1TypeKubernetes
)

// String implements Stringer interface
func (e TfgridReservationWorkload1TypeEnum) String() string {
	switch e {
	case TfgridReservationWorkload1TypeNetwork:
		return "network"
	case TfgridReservationWorkload1TypeVolume:
		return "volume"
	case TfgridReservationWorkload1TypeZdb:
		return "zdb"
	case TfgridReservationWorkload1TypeContainer:
		return "container"
	case TfgridReservationWorkload1TypeKubernetes:
		return "kubernetes"
	}
	return "UNKNOWN"
}
