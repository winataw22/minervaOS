package provision

import (
	schema "github.com/threefoldtech/zos/pkg/schema"
)

// TfgridReservation1 jsx schema
type TfgridReservation1 struct {
	ID                  uint                                 `json:"id"`
	JSON                string                               `json:"json"`
	DataReservation     TfgridReservationData1               `json:"data_reservation"`
	CustomerTid         int64                                `json:"customer_tid"`
	CustomerSignature   string                               `json:"customer_signature"`
	NextAction          TfgridReservation1NextActionEnum     `json:"next_action"`
	SignaturesProvision []TfgridReservationSigningSignature1 `json:"signatures_provision"`
	SignaturesFarmer    []TfgridReservationSigningSignature1 `json:"signatures_farmer"`
	SignaturesDelete    []TfgridReservationSigningSignature1 `json:"signatures_delete"`
	Epoch               schema.Date                          `json:"epoch"`
	Results             []TfgridReservationResult1           `json:"results"`
}

// TfgridReservationData1 jsx schema
type TfgridReservationData1 struct {
	Description             string                           `json:"description"`
	SigningRequestProvision TfgridReservationSigningRequest1 `json:"signing_request_provision"`
	SigningRequestDelete    TfgridReservationSigningRequest1 `json:"signing_request_delete"`
	Containers              []TfgridReservationContainer1    `json:"containers"`
	Volumes                 []TfgridReservationVolume1       `json:"volumes"`
	Zdbs                    []TfgridReservationZdb1          `json:"zdbs"`
	Networks                []TfgridReservationNetwork1      `json:"networks"`
	ExpirationProvisioning  schema.Date                      `json:"expiration_provisioning"`
	ExpirationReservation   schema.Date                      `json:"expiration_reservation"`
}

// TfgridReservationSigningRequest1 jsx schema
type TfgridReservationSigningRequest1 struct {
	Signers   []int64 `json:"signers"`
	QuorumMin int64   `json:"quorum_min"`
}

//TfgridReservationSigningSignature1 jsx schema
type TfgridReservationSigningSignature1 struct {
	Tid       int64       `json:"tid"`
	Signature string      `json:"signature"`
	Epoch     schema.Date `json:"epoch"`
}

//TfgridReservation1NextActionEnum jsx schema
type TfgridReservation1NextActionEnum uint8

// TfgridReservation1NextActionEnum enum
const (
	TfgridReservation1NextActionCreate TfgridReservation1NextActionEnum = iota
	TfgridReservation1NextActionSign
	TfgridReservation1NextActionPay
	TfgridReservation1NextActionDeploy
	TfgridReservation1NextActionDelete
	TfgridReservation1NextActionInvalid
	TfgridReservation1NextActionDeleted
)

func (e TfgridReservation1NextActionEnum) String() string {
	switch e {
	case TfgridReservation1NextActionCreate:
		return "create"
	case TfgridReservation1NextActionSign:
		return "sign"
	case TfgridReservation1NextActionPay:
		return "pay"
	case TfgridReservation1NextActionDeploy:
		return "deploy"
	case TfgridReservation1NextActionDelete:
		return "delete"
	case TfgridReservation1NextActionInvalid:
		return "invalid"
	case TfgridReservation1NextActionDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}
