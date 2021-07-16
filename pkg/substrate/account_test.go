package substrate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddress(t *testing.T) {
	require := require.New(t)

	address := "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY"

	account, err := FromAddress(address)
	require.NoError(err)

	require.Equal(address, account.String())
}
