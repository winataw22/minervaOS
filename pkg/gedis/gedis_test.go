package gedis

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/mock"
)

type mockPool struct {
	mock.Mock
}

func (p *mockPool) Get() redis.Conn {
	args := p.Called()
	return args.Get(0).(redis.Conn)
}

func (p *mockPool) Close() error {
	return p.Called().Error(0)
}

type mockConn struct {
	mock.Mock
}

func (c *mockConn) Close() error {
	return c.Called().Error(0)
}

// Err returns a non-nil value when the connection is not usable.
func (c *mockConn) Err() error {
	return c.Called().Error(0)
}

// Do sends a command to the server and returns the received reply.
func (c *mockConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	set := append([]interface{}{commandName}, args...)
	a := c.Called(set...)
	return a.Get(0), a.Error(1)
}

// Send writes the command to the client's output buffer.
func (c *mockConn) Send(commandName string, args ...interface{}) error {
	set := append([]interface{}{commandName}, args...)
	a := c.Called(set...)
	return a.Error(1)
}

// Flush flushes the output buffer to the Redis server.
func (c *mockConn) Flush() error {
	return c.Called().Error(0)
}

// Receive receives a single reply from the Redis server
func (c *mockConn) Receive() (reply interface{}, err error) {
	a := c.Called()
	return a.Get(0), a.Error(1)
}

func getTestPool() (*mockPool, *mockConn) {
	var pool mockPool
	var conn mockConn
	pool.On("Clone").Return(nil)
	pool.On("Get").Return(&conn)

	conn.On("Close").Return(nil)
	return &pool, &conn
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	r, err := json.Marshal(v)
	require.NoError(t, err)
	return r
}

func EqualJSON(t *testing.T, expected, actual []byte) {
	var exp interface{}
	var act interface{}

	if err := json.Unmarshal(expected, &exp); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(actual, &act); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, exp, act)
}

func TestGedisSend(t *testing.T) {
	require := require.New(t)

	pool, conn := getTestPool()

	gedis := &Gedis{
		pool:      pool,
		namespace: "default",
	}

	conn.On("Do", "default.actor.method", mock.AnythingOfType("[]uint8")).
		Return("mocked result", nil)

	args := Args{
		"key": "value",
	}
	result, err := gedis.Send("actor", "method", args)

	require.NoError(err)
	require.Equal("mocked result", result)

	expected := mustMarshal(t, args)

	conn.AssertCalled(t, "Do", "default.actor.method", expected)
	conn.AssertCalled(t, "Close")
}
