package provision

import (
	"context"

	"github.com/threefoldtech/zbus"

	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/tnodb"
)

type (
	tnodbKey struct{}
	zbusKey  struct{}
)

// WithTnoDB adds a tnodb middleware
func WithTnoDB(ctx context.Context, address string) context.Context {
	value := tnodb.NewHTTPHTTPTNoDB(address)
	return context.WithValue(ctx, tnodbKey{}, value)
}

// GetTnoDB get tnodb client from context
func GetTnoDB(ctx context.Context) network.TNoDB {
	value := ctx.Value(tnodbKey{})
	if value == nil {
		panic("no tnodb middleware associated with context")
	}

	return value.(network.TNoDB)
}

// WithZBus adds a zbus client middleware to context
func WithZBus(ctx context.Context, client zbus.Client) context.Context {
	return context.WithValue(ctx, zbusKey{}, client)
}

// GetZBus gets a zbus client from context
func GetZBus(ctx context.Context) zbus.Client {
	value := ctx.Value(zbusKey{})
	if value == nil {
		panic("no tnodb middleware associated with context")
	}

	return value.(zbus.Client)
}
