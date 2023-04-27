package client

import (
	"context"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
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
