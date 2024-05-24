package perf

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

const (
	moduleName = "perf"
)

var (
	ErrResultNotFound = errors.New("result not found")
)

// generateKey is helper method to add moduleName as prefix for the taskName
func generateKey(taskName string) string {
	return fmt.Sprintf("%s.%s", moduleName, taskName)
}

// setCache set result in redis
func (pm *PerformanceMonitor) setCache(ctx context.Context, result pkg.TaskResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data to JSON")
	}

	conn := pm.pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", generateKey(result.Name), data)
	return err
}

// get directly gets result for some key
func get(conn redis.Conn, key string) (pkg.TaskResult, error) {
	var res pkg.TaskResult

	data, err := conn.Do("GET", key)
	if err != nil {
		return res, errors.Wrap(err, "failed to get the result")
	}

	if data == nil {
		return res, ErrResultNotFound
	}

	err = json.Unmarshal(data.([]byte), &res)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal data from json")
	}

	return res, nil
}

// Get gets data from redis
func (pm *PerformanceMonitor) Get(taskName string) (pkg.TaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()
	return get(conn, generateKey(taskName))
}

// GetAll gets the results for all the tests with moduleName as prefix
func (pm *PerformanceMonitor) GetAll() ([]pkg.TaskResult, error) {
	var res []pkg.TaskResult

	conn := pm.pool.Get()
	defer conn.Close()

	var keys []string

	cursor := 0
	for {
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", generateKey("*")))
		if err != nil {
			return nil, err
		}

		_, err = redis.Scan(values, &cursor, &keys)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			result, err := get(conn, key)
			if err != nil {
				continue
			}
			res = append(res, result)
		}

		if cursor == 0 {
			break
		}

	}
	return res, nil
}

// exists check if a key exists
func (pm *PerformanceMonitor) exists(key string) (bool, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	ok, err := redis.Bool(conn.Do("EXISTS", generateKey(key)))
	if err != nil {
		return false, errors.Wrapf(err, "error checking if key %s exists", generateKey(key))
	}
	return ok, nil
}
