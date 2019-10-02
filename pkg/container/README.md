## Container module

A Zbus service to start, stop and inspect containers. The service
provides the interface defined [here](../../specs/container/readme.md#module-interface)

## Dependency
The module depends on the [flister module](../flist) to mount
the container rootfs

## Example usage
```go
package main

import (
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func main() {
	client, err := zbus.NewRedisClient("tcp://localhost:6379")
	if err != nil {
		panic(client)
	}

	containerd := stubs.NewContainerModuleStub(client)
	namespace := "example"

	// make sure u have a network namespace ready using ip
	// sudo ip netns add mynetns

	info := pkg.Container{
		Name: "test",
		FList: "https://hub.grid.tf/thabet/redis.flist",
		Env: []string{},
		Network: pkg.NetworkInfo{Namespace: "mynetns"},
		Mounts: nil,
		Entrypoint: "redis-server",
	}

	id, err := containerd.Run(namespace, info)

	if err != nil {
		panic(err)
	}

	// DO WORK WITH CONTAINER ...

	if err = containerd.Delete(namespace, id); err != nil {
		panic(err)
	}

}
```
