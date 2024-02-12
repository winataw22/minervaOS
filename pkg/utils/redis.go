package utils

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisDialParams struct {
	Scheme  string
	Host    string
	Options []redis.DialOption
}

func parseRedisAddress(address string) (RedisDialParams, error) {
	var params RedisDialParams
	u, err := url.Parse(address)
	if err != nil {
		return params, err
	}

	params.Scheme = u.Scheme

	switch params.Scheme {
	case "redis":
		params.Scheme = "tcp"
		fallthrough
	case "tcp":
		params.Host = u.Host
	case "unix":
		params.Host = u.Path
	default:
		return params, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}

	if u.User != nil {
		params.Options = append(
			params.Options,
			redis.DialPassword(u.User.Username()),
		)
	}

	return params, nil
}

func NewRedisPool(address string, size ...uint32) (*redis.Pool, error) {
	var poolSize uint32 = 20
	if len(size) == 1 {
		poolSize = size[0]
	} else if len(size) > 1 {
		panic("invalid pool size")
	}

	params, err := parseRedisAddress(address)
	if err != nil {
		return nil, err
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(params.Scheme, params.Host, params.Options...)
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

func NewRedisConn(address string) (redis.Conn, error) {
	params, err := parseRedisAddress(address)
	if err != nil {
		return nil, err
	}
	return redis.Dial(params.Scheme, params.Host, params.Options...)
}
