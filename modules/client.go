package modules

// NOTE: this file is causing cyclc dependencies
// in case the modules interfaces are defining other data types
// this way stubs will also have to import modules

// import (
// 	"github.com/threefoldtech/zbus"
// 	"github.com/threefoldtech/zosv2/modules/stubs"
// )

// const redisSocket = "unix://var/run/redis.sock"

// var zbusClient zbus.Client

// func init() {
// 	var err error
// 	zbusClient, err = zbus.NewRedisClient(redisSocket)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// // Flist returns a client to the flist module
// func Flist() (Flister, error) {
// 	return stubs.NewFlisterStub(zbusClient), nil
// }
