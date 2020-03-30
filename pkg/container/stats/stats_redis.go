package stats

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
	"io"
	"net/url"
)

// RedisType defines the type name of redis backend
const RedisType = "redis"

// Redis defines how to connect a stats redis backend
type Redis struct {
	Endpoint string `bson:"stdout" json:"endpoint"`
}

// RedisBackend define an internal redis backend
type RedisBackend struct {
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
func NewRedis(endpoint string) (io.WriteCloser, error) {
	log.Debug().Msg("initializing redis stats aggregator")

	host, channel, err := RedisParseURL(endpoint)
	if err != nil {
		return nil, err
	}

	conn, err := redis.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("host", host).Str("channel", channel).Msg("redis stats")

	aggregator := &RedisBackend{
		channel: channel,
		conn:    conn,
	}

	return aggregator, nil
}

// Write will write to the channel
func (c *RedisBackend) Write(data []byte) (int, error) {
	_, err := c.conn.Do("PUBLISH", c.channel, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

// Close closes redis connection
func (c *RedisBackend) Close() error {
	log.Debug().Str("channel", c.channel).Msg("closind redis stats backend")
	c.conn.Close()
	return nil
}
