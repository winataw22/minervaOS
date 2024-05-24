package zosapi

import (
	"context"
)

func (g *ZosAPI) gpuListHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.statisticsStub.ListGPUs(ctx)
}
