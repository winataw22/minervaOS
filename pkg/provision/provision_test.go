package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
)

func MustMarshal(t *testing.T, v interface{}) []byte {
	bytes, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

type TestClient struct {
	mock.Mock
}

// Request makes a request and return the response data
func (t *TestClient) Request(module string, object zbus.ObjectID, method string, args ...interface{}) (*zbus.Response, error) {
	inputs := []interface{}{
		module, object, method,
	}
	inputs = append(inputs, args...)

	return zbus.NewResponse("", "", t.Called(inputs...)...)
}

// Stream listens to a stream of events from the server
func (t *TestClient) Stream(ctx context.Context, module string, object zbus.ObjectID, event string) (<-chan zbus.Event, error) {
	panic("not implemented")
}

func TestClientOperation(t *testing.T) {
	require := require.New(t)
	var client TestClient

	client.On("Request", "module", zbus.ObjectID{}, "test", 123, []string{"hello", "world"}).
		Return("result", nil)

	response, err := client.Request("module", zbus.ObjectID{}, "test", 123, []string{"hello", "world"})
	require.NoError(err)

	var str string
	var rerr *zbus.RemoteError
	require.NoError(response.Unmarshal(0, &str))
	require.NoError(response.Unmarshal(1, &rerr))

	require.Equal("result", str)
	require.Nil(rerr)
}

type TestOwnerCache struct {
	mock.Mock
}

func (t *TestOwnerCache) OwnerOf(id string) (string, error) {
	result := t.Called(id)
	return result.String(0), result.Error(1)
}

func TestErrTemporary(t *testing.T) {
	err := fmt.Errorf("base error")
	wrap1 := NewErrTemporary(err)
	wrap2 := fmt.Errorf("upper err: %w", wrap1)

	assert.True(t, errors.Is(wrap1, ErrTemporary{}))
	assert.True(t, errors.Is(wrap2, ErrTemporary{}))
	assert.Equal(t, err.Error(), wrap1.Error())
}
