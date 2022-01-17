package zdb

import (
	"fmt"
	"strings"

	"github.com/gomodule/redigo/redis"
	"gopkg.in/yaml.v2"
)

// CreateNamespace creates a new namespace. Only admin can do this.
// By default, a namespace is not password protected, is public and not size limited.
func (c *clientImpl) CreateNamespace(name string) error {
	con := c.pool.Get()
	defer con.Close()
	ok, err := redis.String(con.Do("NSNEW", name))
	if err != nil {
		return err
	}
	if ok != "OK" {
		return fmt.Errorf(ok)
	}
	return nil
}

// Exist checks if namespace exists
func (c *clientImpl) Exist(name string) (bool, error) {
	con := c.pool.Get()
	defer con.Close()

	_, err := con.Do("NSINFO", name)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// DeleteNamespace deletes a namespace. Only admin can do this.
// You can't remove the namespace you're currently using.
// Any other clients using this namespace will be moved to a special state, awaiting to be disconnected.
func (c *clientImpl) DeleteNamespace(name string) error {
	con := c.pool.Get()
	defer con.Close()
	ok, err := redis.String(con.Do("NSDEL", name))
	if err != nil {
		return err
	}
	if ok != "OK" {
		return fmt.Errorf(ok)
	}
	return nil
}

// Namespaces returns a slice of all available namespaces name.
func (c *clientImpl) Namespaces() ([]string, error) {
	con := c.pool.Get()
	defer con.Close()
	return redis.Strings(con.Do("NSLIST"))
}

// NamespaceSetSize sets the maximum size in bytes, of the namespace's data set
func (c *clientImpl) NamespaceSetSize(name string, size uint64) error {
	con := c.pool.Get()
	defer con.Close()
	ok, err := redis.String(con.Do("NSSET", name, "maxsize", size))
	if err != nil {
		return err
	}
	if ok != "OK" {
		return fmt.Errorf(ok)
	}
	return nil
}

// NamespaceSetPassword locks the namespace by a password, use * password to clear it
func (c *clientImpl) NamespaceSetPassword(name, password string) error {
	con := c.pool.Get()
	defer con.Close()
	ok, err := redis.String(con.Do("NSSET", name, "password", password))
	if err != nil {
		return err
	}
	if ok != "OK" {
		return fmt.Errorf(ok)
	}
	return nil
}

// NamespaceSetMode locks the namespace by a password, use * password to clear it
func (c *clientImpl) NamespaceSetMode(name, mode string) error {
	con := c.pool.Get()
	defer con.Close()
	ok, err := redis.String(con.Do("NSSET", name, "mode", mode))
	if err != nil {
		return err
	}
	if ok != "OK" {
		return fmt.Errorf(ok)
	}
	return nil
}

// NamespaceSetPublic changes the public flag, a public namespace can be read-only if a password is set
func (c *clientImpl) NamespaceSetPublic(name string, public bool) error {
	con := c.pool.Get()
	defer con.Close()

	flag := 0
	if public {
		flag = 1
	}

	ok, err := redis.String(con.Do("NSSET", name, "public", flag))
	if err != nil {
		return err
	}
	if ok != "OK" {
		return fmt.Errorf(ok)
	}
	return nil
}

// DBSize returns the size of the database in bytes
func (c *clientImpl) DBSize() (uint64, error) {
	con := c.pool.Get()
	defer con.Close()

	size, err := redis.Uint64(con.Do("DBSIZE"))
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (c *clientImpl) Namespace(name string) (ns Namespace, err error) {
	con := c.pool.Get()
	defer con.Close()

	data, err := redis.Bytes(con.Do("NSINFO", name))
	if err != nil {
		return ns, err
	}

	if err := yaml.Unmarshal(data, &ns); err != nil {
		return ns, err
	}

	return
}
