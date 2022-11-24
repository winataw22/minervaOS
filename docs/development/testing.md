# Testing

Beside unit testing, you might want to test your change in an integrated environment, the following are two options to do it.

- [Testing](#testing)
  - [Using grid/node client](#using-gridnode-client)
  - [Using a test app](#using-a-test-app)
    - [An example to talk to container and qsfs modules](#an-example-to-talk-to-container-and-qsfs-modules)
    - [An example of directly using zinit package](#an-example-of-directly-using-zinit-package)


## Using grid/node client

You can simply use any grid client, e.g. [grid_client_ts](https://github.com/threefoldtech/grid3_client_ts/tree/development/scripts) to deploy a workload of any type, you should specify your node's twin ID (and make sure you are on the correct network). 

Inside the node, you can do `noded -id` and `noded -net` to get your current node ID and network. Also, [you can check your farm](https://dashboard.dev.grid.tf/explorer/farms) and get node information from there.

Another option is the golang [node client](../manual/manual.md#interaction).

While deploying on your local node, logs with `zinit log` would be helpful to see any possible errors and to debug your code.

## Using a test app

If you need to test a specific module or functionality, you can create a simple test app inside e.g. [tools directory](../../tools/).

Inside this simple test app, you can import any module or talk to another one using [zbus](../internals/internals.md#ipc).

### An example to talk to container and qsfs modules


```go
// tools/del/main.go

package main

import (
	"context"
	"flag"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	zbus, err := zbus.NewRedisClient("unix:///var/run/redis.sock")
	if err != nil {
		log.Err(err).Msg("cannot init zbus client")
		return
	}

	var workloadType, workloadID string

	flag.StringVar(&workloadType, "type", "", "workload type (qsfs or container)")
	flag.StringVar(&workloadID, "id", "", "workload ID")

	flag.Parse()

	if workloadType == "" || workloadID == "" {
		log.Error().Msg("you need to provide both type and id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if workloadType == "qsfs" {
		qsfsd := stubs.NewQSFSDStub(zbus)
		err := qsfsd.SignalDelete(ctx, workloadID)
		if err != nil {
			log.Err(err).Msg("cannot delete qsfs workload")
		}
	} else if workloadType == "container" {
		args := strings.Split(workloadID, ":")
		if len(args) != 2 {
			log.Error().Msg("container id must contain namespace, e.g. qsfs:wl129")
		}

		containerd := stubs.NewContainerModuleStub(zbus)
		err := containerd.SignalDelete(ctx, args[0], pkg.ContainerID(args[1]))
		if err != nil {
			log.Err(err).Msg("cannot delete container workload")
		}
	}

}
```

Then we can simply build, upload and execute this in our node:

```
cd tools/del
go build
scp del root@192.168.123.44:/root/del
```

Then ssh into `192.168.123.44` and simply execute your test app:

```
./del
```

### An example of directly using zinit package

```go
// tools/zinit_test
package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/zinit"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	z := zinit.New("/var/run/zinit.sock")

	regex := fmt.Sprintf(`^ip netns exec %s %s`, "ndmz", "/sbin/udhcpc")
	_, err := regexp.Compile(regex)
	if err != nil {
		log.Err(err).Msgf("cannot compile %s", regex)
		return
	}

	// try match
	matched, err := z.Matches(zinit.WithExecRegex(regex))
	if err != nil {
		log.Err(err).Msg("cannot filter services")
	}

	matchedStr, err := json.Marshal(matched)
	if err != nil {
		log.Err(err).Msg("cannot convert matched map to json")
	}

	log.Debug().Str("matched", string(matchedStr)).Msg("matched services")

	// // try destroy
	// err = z.Destroy(10*time.Second, matched...)
	// if err != nil {
	// 	log.Err(err).Msg("cannot destroy matched services")
	// }
}
```