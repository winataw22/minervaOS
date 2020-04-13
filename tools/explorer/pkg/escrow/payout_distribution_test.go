package escrow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayoutDistributionValidation(t *testing.T) {
	distributions := []payoutDistribution{
		{
			farmer:     23,
			burned:     56,
			foundation: 0,
		},
		{
			farmer:     33,
			burned:     33,
			foundation: 33,
		},
		{
			farmer:     50,
			burned:     40,
			foundation: 10,
		},
	}

	assert.Error(t, distributions[0].validate(), "")
	assert.Error(t, distributions[1].validate(), "")
	assert.NoError(t, distributions[2].validate())
}

func TestKnownPayoutDistributions(t *testing.T) {
	for _, pd := range assetDistributions {
		assert.NoError(t, pd.validate())
	}
}
