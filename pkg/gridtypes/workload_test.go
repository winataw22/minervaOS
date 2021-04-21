package gridtypes

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimestamp(t *testing.T) {
	require := require.New(t)

	var v Timestamp

	err := json.Unmarshal([]byte(`1234`), &v)
	require.NoError(err)
	require.Equal(Timestamp(1234), v)

	n := time.Now()
	exp, err := json.Marshal(n)
	require.NoError(err)

	err = json.Unmarshal([]byte(exp), &v)
	require.NoError(err)

	require.Equal(Timestamp(n.Unix()), v)
}
