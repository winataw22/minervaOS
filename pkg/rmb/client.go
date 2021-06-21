package rmb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	systemLocalBus = "msgbus.system.local"

	// DefaultAddress default redis address when no address is passed
	DefaultAddress = "tcp://127.0.0.1:6379"
)

type redisClient struct {
	pool *redis.Pool
}

// NewClient creates a new rmb client. the given address should
// be to the local redis. If not provided, default redis address
// is used
func NewClient(address string) (Client, error) {
	if len(address) == 0 {
		address = DefaultAddress
	}

	pool, err := newRedisPool(address)
	if err != nil {
		return nil, err
	}

	return &redisClient{
		pool: pool,
	}, nil
}

func (c *redisClient) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request data")
	}

	queue := uuid.NewString()
	msg := Message{
		Version:    1,
		Expiration: 3600,
		Command:    fn,
		TwinDest:   []uint32{twin},
		Data:       base64.StdEncoding.EncodeToString(bytes),
		Retqueue:   queue,
	}

	bytes, err = json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to serialize message")
	}
	con := c.pool.Get()
	defer con.Close()

	_, err = con.Do("RPUSH", systemLocalBus, bytes)
	if err != nil {
		return errors.Wrap(err, "failed to push message to local twin")
	}

	// now wait for response.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slice, err := redis.ByteSlices(con.Do("BLPOP", queue, 5))
		if err != nil && err != redis.ErrNil {
			return errors.Wrap(err, "unexpected error during waiting for the response")
		}

		if err == redis.ErrNil || slice == nil {
			//timeout, just try again immediately
			continue
		}

		// found a response
		bytes = slice[1]
		break
	}

	// we have a response, so load or fail
	if err := json.Unmarshal(bytes, &msg); err != nil {
		return errors.Wrap(err, "failed to load response message")
	}

	// errorred ?
	if len(msg.Err) != 0 {
		return errors.New(msg.Err)
	}

	// not expecting a result
	if result == nil {
		return nil
	}

	if len(msg.Data) == 0 {
		return fmt.Errorf("no response body was returned")
	}

	bytes, err = base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		return errors.Wrap(err, "invalid data body encoding")
	}

	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrap(err, "failed to decode response body")
	}

	return nil
}
