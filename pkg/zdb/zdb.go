// Package zdb implements a client to 0-db: https://github.com/threefoldtech/0-DB
package zdb

import (
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/utils"
)

type Namespace struct {
	Name              string         `yaml:"name"`
	DataLimit         gridtypes.Unit `yaml:"data_limits_bytes"`
	DataDiskFreespace gridtypes.Unit `yaml:"data_disk_freespace_bytes"`
	Mode              string         `yaml:"mode"`
	PasswordProtected bool           `yaml:"password"`
	Public            bool           `yaml:"public"`
}

// Client interface
type Client interface {
	Connect() error
	Close() error
	CreateNamespace(name string) error
	Exist(name string) (bool, error)
	DeleteNamespace(name string) error
	Namespaces() ([]string, error)
	Namespace(name string) (Namespace, error)
	NamespaceSetSize(name string, size uint64) error
	NamespaceSetPassword(name, password string) error
	NamespaceSetMode(name, mode string) error
	NamespaceSetPublic(name string, public bool) error
	NamespaceSetLock(name string, lock bool) error
	DBSize() (uint64, error)
}

// clientImpl is a connection to a 0-db
type clientImpl struct {
	addr string
	pool *redis.Pool
}

// New creates a client to 0-db pointed by addr
// addr format: TODO:
func New(addr string) Client {
	return &clientImpl{
		addr: addr,
	}
}

// Connect dials addr and creates a pool of connection
func (c *clientImpl) Connect() error {
	if c.pool == nil {
		pool, err := utils.NewRedisPool(c.addr, 3)
		if err != nil {
			return errors.Wrapf(err, "failed to connect to %s", c.addr)
		}

		c.pool = pool
	}

	con := c.pool.Get()
	defer con.Close()
	_, err := con.Do("PING")

	return err
}

// Close releases the resources used by the client.
func (c *clientImpl) Close() error {
	if c.pool == nil {
		return nil
	}

	if err := c.pool.Close(); err != nil {
		return err
	}
	c.pool = nil
	return nil
}
