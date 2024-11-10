package perf

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zbus"
)

// WithZbusClient adds a zbus.Client to the provided context, returning a new context.
// This allows for the retrieval of the client from the context at a later time.
type zbusClientKey struct{}

func WithZbusClient(ctx context.Context, client zbus.Client) context.Context {
	return context.WithValue(ctx, zbusClientKey{}, client)
}

// MustGetZbusClient gets zbus client from the given context
func MustGetZbusClient(ctx context.Context) zbus.Client {
	return ctx.Value(zbusClientKey{}).(zbus.Client)
}

// TryGetZbusClient tries to get zbus client from the given context
func TryGetZbusClient(ctx context.Context) (zbus.Client, error) {
	zcl, ok := ctx.Value(zbusClientKey{}).(zbus.Client)
	if !ok {
		return zcl, fmt.Errorf("context does not have zbus client")
	}
	return zcl, nil
}
