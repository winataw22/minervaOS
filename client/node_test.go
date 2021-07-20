package client

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zos/pkg/rmb"
)

func ExampleClient() {
	client, err := rmb.Default()
	if err != nil {
		panic(err)
	}

	node := NewNodeClient(10, client)

	node.Counters(context.Background())
	fmt.Println("ok")
	//Output: ok
}
