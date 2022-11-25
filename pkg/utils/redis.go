package utils

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
)

func NewRedisPool(address string, size ...uint32) (*redis.Pool, error) {
	var poolSize uint32 = 20
	if len(size) == 1 {
		poolSize = size[0]
	} else if len(size) > 1 {
		panic("invalid pool size")
	}

	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var host string
	switch u.Scheme {
	case "redis":
		u.Scheme = "tcp"
		fallthrough
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
		MaxActive:   int(poolSize),
		MaxIdle:     3,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}
