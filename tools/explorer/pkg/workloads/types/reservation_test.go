package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidation(t *testing.T) {
	const (
		input = `{
			"id": 4,
			"json": "{\"expiration_reservation\": 1582937602, \"expiration_provisioning\": 1582937602}",
			"data_reservation": {
			"description": "",
			"signing_request_provision": {
			"signers": null,
			"quorum_min": 0
			},
			"signing_request_delete": {
			"signers": null,
			"quorum_min": 0
			},
			"containers": null,
			"volumes": null,
			"zdbs": null,
			"networks": null,
			"kubernetes": null,
			"expiration_provisioning": 1582937602,
			"expiration_reservation": 1582937602
			},
			"customer_tid": 7,
			"customer_signature": "225af4e3bc6cdaa44e877821bcbb6a8201b9c080ef66dc192a0511927dc51b70447b14d663f492feec5434ee1a1b720785d3d9de6fd5e594c8010720ddb7eb00",
			"next_action": 3,
			"signatures_provision": null,
			"signatures_farmer": null,
			"signatures_delete": null,
			"epoch": 0,
			"results": null
			}`
	)
	var reservation Reservation
	err := json.Unmarshal([]byte(input), &reservation)
	require.NoError(t, err)

	err = reservation.validate()
	require.NoError(t, err)
}
