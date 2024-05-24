package zosapi

import (
	"context"
)

func (g *ZosAPI) storagePoolsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.storageStub.Metrics(ctx)
}
