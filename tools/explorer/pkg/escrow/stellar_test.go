package escrow

import (
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
)

func TestPayoutDistribution(t *testing.T) {
	// for these tests, keep in mind that the `amount` given is in the highest
	// precision of the underlying wallet, but the reservation costs only have
	// up to 6 digits precision. In case of stellar, the wallet has 7 digits precision.
	// This means the smallest amount that will be expressed is `10` rather than `1`.
	//
	// note that the actual amount to be paid can have up to the wallets precision,
	// i.e. it is possible to have greater than 6 digits precision
	pds := []payoutDistribution{
		{
			farmer:     50,
			burned:     50,
			foundation: 0,
		},
		{
			farmer:     34,
			burned:     33,
			foundation: 33,
		},
		{
			farmer:     40,
			burned:     40,
			foundation: 20,
		},
		{
			farmer:     0,
			burned:     73,
			foundation: 27,
		},
	}

	for _, pd := range pds {
		assert.NoError(t, pd.validate())
	}

	w, err := stellar.New("", stellar.NetworkTest, nil)
	assert.NoError(t, err)

	e := NewStellar(w, nil, "")

	// check rounding in some trivial cases
	farmer, burn, fd := e.splitPayout(10, pds[0])
	assert.Equal(t, xdr.Int64(5), farmer)
	assert.Equal(t, xdr.Int64(5), burn)
	assert.Equal(t, xdr.Int64(0), fd)

	farmer, burn, fd = e.splitPayout(10, pds[1])
	assert.Equal(t, xdr.Int64(4), farmer)
	assert.Equal(t, xdr.Int64(3), burn)
	assert.Equal(t, xdr.Int64(3), fd)

	farmer, burn, fd = e.splitPayout(10, pds[2])
	assert.Equal(t, xdr.Int64(4), farmer)
	assert.Equal(t, xdr.Int64(4), burn)
	assert.Equal(t, xdr.Int64(2), fd)

	farmer, burn, fd = e.splitPayout(10, pds[3])
	assert.Equal(t, xdr.Int64(0), farmer)
	assert.Equal(t, xdr.Int64(8), burn)
	assert.Equal(t, xdr.Int64(2), fd)

	farmer, burn, fd = e.splitPayout(330, pds[0])
	assert.Equal(t, xdr.Int64(165), farmer)
	assert.Equal(t, xdr.Int64(165), burn)
	assert.Equal(t, xdr.Int64(0), fd)

	farmer, burn, fd = e.splitPayout(330, pds[1])
	assert.Equal(t, xdr.Int64(114), farmer)
	assert.Equal(t, xdr.Int64(108), burn)
	assert.Equal(t, xdr.Int64(108), fd)

	farmer, burn, fd = e.splitPayout(330, pds[2])
	assert.Equal(t, xdr.Int64(132), farmer)
	assert.Equal(t, xdr.Int64(132), burn)
	assert.Equal(t, xdr.Int64(66), fd)

	farmer, burn, fd = e.splitPayout(330, pds[3])
	assert.Equal(t, xdr.Int64(0), farmer)
	assert.Equal(t, xdr.Int64(241), burn)
	assert.Equal(t, xdr.Int64(89), fd)
}
