package provision

import (
	"context"

	"github.com/threefoldtech/zbus"
)

type (
	zbusKey       struct{}
	owerCacheKey  struct{}
	zdbMappingKey struct{}
)

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

// OwnerCache interface
type OwnerCache interface {
	OwnerOf(reservationID string) (string, error)
}

// WithOwnerCache adds the owner cache to context
func WithOwnerCache(ctx context.Context, cache OwnerCache) context.Context {
	return context.WithValue(ctx, owerCacheKey{}, cache)
}

// GetOwnerCache gets the owner cache from context
func GetOwnerCache(ctx context.Context) OwnerCache {
	value := ctx.Value(owerCacheKey{})
	if value == nil {
		panic("no reservation cache associated with context")
	}

	return value.(OwnerCache)
}

// ZDBMapping interface
type ZDBMapping interface {

	// Get returns the container ID where namespace lives
	// if the namespace is not found an empty string and false is returned
	Get(namespace string) (string, bool)

	// Set saves the mapping between the namespace and a container ID
	Set(namespace, container string)
}

// WithZDBMapping set ZDBMapping into the context
func WithZDBMapping(ctx context.Context, mapping ZDBMapping) context.Context {
	return context.WithValue(ctx, zdbMappingKey{}, mapping)
}

// GetZDBMapping gets the zdb mapping from the context
func GetZDBMapping(ctx context.Context) ZDBMapping {
	value := ctx.Value(zdbMappingKey{})
	if value == nil {
		panic("no reservation mapping associated with context")
	}

	return value.(ZDBMapping)
}
