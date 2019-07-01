package identity

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules/kernel"
)

// GetFarmID reads the farmer id from the kernel parameters
// return en error if the farmer id is not set
func GetFarmID() (Identifier, error) {
	params := kernel.GetParams()

	farmerID, found := params.Get("farmer_id")
	if !found {
		return nil, fmt.Errorf("farmer id not found in kernel parameters")
	}

	return strIdentifier(farmerID[0]), nil
}
