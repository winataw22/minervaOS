// Package gedis implements a client for Gedis (https://github.com/threefoldtech/digitalmeX/tree/master/docs/Gedis)
package gedis

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
)

var (
	// Bytes is a helper that converts a command reply to a slice of bytes. If err
	// is not equal to nil, then Bytes returns nil, err. Otherwise Bytes converts
	// the reply to a slice of bytes as follows:
	//
	//  Reply type      Result
	//  bulk string     reply, nil
	//  simple string   []byte(reply), nil
	//  nil             nil, ErrNil
	//  other           nil, error
	Bytes = redis.Bytes

	// Bool is a helper that converts a command reply to a boolean. If err is not
	// equal to nil, then Bool returns false, err. Otherwise Bool converts the
	// reply to boolean as follows:
	//
	//  Reply type      Result
	//  integer         value != 0, nil
	//  bulk string     strconv.ParseBool(reply)
	//  nil             false, ErrNil
	//  other           false, error
	Bool = redis.Bool
)

// Args is a helper to create map easily
type Args map[string]interface{}

// Pool is interface for a redis pool
type Pool interface {
	Get() redis.Conn
	Close() error
}

// Gedis struct represent a client to a gedis server
type Gedis struct {
	pool      Pool
	namespace string
	password  string
}

// New creates a new Gedis client
func New(address, namespace, password string) (*Gedis, error) {
	pool, err := newRedisPool(address)
	if err != nil {
		return nil, err
	}

	return &Gedis{
		pool:      pool,
		namespace: namespace,
		password:  password,
	}, nil
}

// Close closes all connection to the gedis server and stops
// the close the connection pool
func (g *Gedis) Close() error {
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
			con, err := redis.Dial(u.Scheme, host, opts...)
			if err != nil {
				return nil, err
			}
			_, err = con.Do("config_format", "json")
			return con, err
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

// Send send a command and read response.
// args, is a map with argument name, and value as defined by the actor schema.
// Usually you need to process the returned value through `Bytes`, `Bool` or other wrappers based
// on the expected return value
func (g *Gedis) Send(actor, method string, args Args) (interface{}, error) {
	con := g.pool.Get()
	defer con.Close()

	bytes, err := json.Marshal(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize arguments")
	}

	result, err := con.Do(g.cmd(actor, method), bytes)
	if err != nil {
		return result, parseError(err)
	}

	return result, nil
}
