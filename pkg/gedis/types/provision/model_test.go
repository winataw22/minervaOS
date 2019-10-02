package provision

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnum(t *testing.T) {
	r := TfgridReservationWorkload1{
		Type: TfgridReservationWorkload1TypeContainer,
	}

	bytes, err := json.Marshal(r)
	require.NoError(t, err)

	var o TfgridReservationWorkload1

	require.NoError(t, json.Unmarshal(bytes, &o))

	require.Equal(t, r.Type, o.Type)
}
