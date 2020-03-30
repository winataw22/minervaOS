package escrow

import (
	"fmt"

	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
)

// assetDistributions for currently supported tokens
var assetDistributions = map[stellar.Asset]payoutDistribution{
	// It's a bit annoying that we don't differntiate between mainnet and standard here,
	// load all assets
	stellar.TFTMainnet:     {farmer: 90, burned: 0, foundation: 10},
	stellar.TFTTestnet:     {farmer: 90, burned: 0, foundation: 10},
	stellar.FreeTFTMainnet: {farmer: 0, burned: 100, foundation: 0},
	stellar.FreeTFTTestnet: {farmer: 0, burned: 100, foundation: 0},
}

// payoutDistribution for a reservation. This represents the percentage of the
// reservation cost that goes to each party.
type payoutDistribution struct {
	farmer     uint8
	burned     uint8
	foundation uint8
}

func (p payoutDistribution) validate() error {
	totalAmount := p.farmer + p.burned + p.foundation
	if totalAmount != 100 {
		return fmt.Errorf("expected total payout distribution to be 100, got %d", totalAmount)
	}

	return nil
}
