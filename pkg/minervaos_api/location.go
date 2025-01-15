package zosapi

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/threefoldtech/zos/pkg/geoip"
)

const (
	locationCacheKey = "location"
)

func (g *ZosAPI) locationGet(ctx context.Context, payload []byte) (interface{}, error) {
	if loc, found := g.inMemCache.Get(locationCacheKey); found {
		return loc, nil
	}

	loc, err := geoip.Fetch()
	if err != nil {
		return nil, err
	}

	g.inMemCache.Set(locationCacheKey, loc, cache.DefaultExpiration)

	return loc, nil
}
