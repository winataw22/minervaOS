// Package gedis implements a client for Gedis (https://github.com/threefoldtech/digitalmeX/tree/master/docs/Gedis)
package gedis

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/garyburd/redigo/redis"
)

// Gedis struct represent a client to a gedis server
type Gedis struct {
	pool      *redis.Pool
	namespace string
	password  string
	// this never change  we cache it to avoid json serialization all the time
	headers []byte

	conn redis.Conn
}

// New creates a new Gedis client
func New(address, namespace, password string) (*Gedis, error) {
	pool, err := newRedisPool(address)
	if err != nil {
		return nil, err
	}

	headers, err := generateHeaders()
	if err != nil {
		return nil, err
	}

	return &Gedis{
		pool:      pool,
		namespace: namespace,
		password:  password,
		headers:   headers,
	}, nil
}

// Close closes all connection to the gedis server and stops
// the close the connection pool
func (g *Gedis) Close() error {
	if g.conn != nil {
		if err := g.conn.Close(); err != nil {
			return err
		}
		g.conn = nil
	}
	return g.pool.Close()
}

// Ping sends a ping to the server. it should return pong
func (g *Gedis) Ping() (string, error) {
	con := g.pool.Get()
	defer con.Close()

	return redis.String(con.Do("PING"))
}

func (g *Gedis) cmd(actor, method string) string {
	return fmt.Sprintf("%s.%s.%s", g.namespace, actor, method)
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

func generateHeaders() ([]byte, error) {
	h := map[string]string{
		"content_type":  "json",
		"response_type": "json",
	}
	bh, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	return bh, nil
}
