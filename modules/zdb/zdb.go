// Package zdb implements a client to 0-db: https://github.com/threefoldtech/0-DB
package zdb

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

// Client is a connection to a 0-db
type Client struct {
	pool *redis.Pool
}

// New creates a client to 0-db pointed by addr
// addr format: TODO:
func New() *Client {
	return &Client{}
}

// Connect dials addr and creates a pool of connection
func (c *Client) Connect(addr string) error {
	if c.pool != nil {
		return fmt.Errorf("already connected")
	}

	pool, err := newRedisPool(addr)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to %s", addr)
	}
	c.pool = pool

	return nil
}

// Close releases the resources used by the client.
func (c *Client) Close() error {
	if c.pool == nil {
		return nil
	}

	if err := c.pool.Close(); err != nil {
		return err
	}
	c.pool = nil
	return nil
}

func newRedisPool(address string) (*redis.Pool, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var host string
	switch u.Scheme {
	case "tcp":
		host = u.Host
	case "unix":
		host = u.Path
	default:
		return nil, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}
	var opts []redis.DialOption

	if u.User != nil {
		opts = append(
			opts,
			redis.DialPassword(u.User.Username()),
		)
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(u.Scheme, host, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				//only check connection if more than 10 second of inactivity
				_, err := c.Do("PING")
				return err
			}

			return nil
		},
		MaxActive:   10,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}
