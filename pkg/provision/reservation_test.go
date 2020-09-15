package provision

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResultNil(t *testing.T) {
	var result Result

	require.True(t, result.IsNil())

	result = Result{}
	require.True(t, result.IsNil())

	null, _ := json.Marshal(nil)
	result = Result{
		Data:    json.RawMessage(null),
		Created: time.Unix(0, 0),
	}

	require.True(t, result.IsNil())
}
