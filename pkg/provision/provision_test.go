package provision

import (
	"context"
	"encoding/json"
	"testing"

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
	for _, arg := range args {
		inputs = append(inputs, arg)
	}

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

type TestZDBCache struct {
	mock.Mock
}

func (t *TestZDBCache) Get(namespace string) (string, bool) {
	result := t.Called(namespace)
	return result.String(0), result.Bool(1)
}

// Set saves the mapping between the namespace and a container ID
func (t *TestZDBCache) Set(namespace, container string) {
	t.Called(namespace, container)
}
