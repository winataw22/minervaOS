package identity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParam(t *testing.T) {
	params, err := readKernelParams()
	require.NoError(t, err)
	for _, param := range params {
		fmt.Println(param.Key(), param.Value())
	}
}
