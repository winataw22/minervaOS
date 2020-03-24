package logger

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
	"io"
	"net/url"
)

// RedisType defines redis logger type name
const RedisType = "redis"

// Redis send stdout/stderr to a Redis PubSub channel
type Redis struct {
	channel string
	conn    redis.Conn
}

// RedisParseURL parse an url and returns interresting part after validation
func RedisParseURL(address string) (host string, channel string, err error) {
	u, err := url.Parse(address)
	if err != nil {
		return "", "", err
	}

	if u.Scheme != "redis" {
		err = fmt.Errorf("invalid scheme, expected redis://")
		return "", "", err
	}

	if u.RequestURI() == "/" {
		err = fmt.Errorf("missing channel name, expected: redis://host/channel")
		return "", "", err
	}

	host = u.Host

	if u.Port() == "" {
		host += ":6379"
	}

	channel = u.RequestURI()[1:]

	return host, channel, err
}

// NewRedis create new redis backend and initialize connection
func NewRedis(stdout string, stderr string) (io.WriteCloser, io.WriteCloser, error) {
	log.Debug().Msg("initializing redis logging")

	ohost, ochannel, err := RedisParseURL(stdout)
	if err != nil {
		return nil, nil, err
	}

	rhost, rchannel, err := RedisParseURL(stderr)
	if err != nil {
		return nil, nil, err
	}

	cout, err := redis.Dial("tcp", ohost)
	if err != nil {
		return nil, nil, err
	}

	cerr, err := redis.Dial("tcp", rhost)
	if err != nil {
		cout.Close()
		return nil, nil, err
	}

	log.Debug().Str("stdout", ohost).Str("channel", ochannel).Msg("redis stdout")
	log.Debug().Str("stderr", rhost).Str("channel", rchannel).Msg("redis stder")

	rstdout := &Redis{
		channel: ochannel,
		conn:    cout,
	}

	rstderr := &Redis{
		channel: rchannel,
		conn:    cerr,
	}

	return rstdout, rstderr, nil
}

// Write will write to the channel
func (c *Redis) Write(data []byte) (int, error) {
	_, err := c.conn.Do("PUBLISH", c.channel, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

// Close closes redis connection
func (c *Redis) Close() error {
	log.Debug().Str("channel", c.channel).Msg("closing redis backend")
	c.conn.Close()
	return nil
}
