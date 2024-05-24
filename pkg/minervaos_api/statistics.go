package zosapi

import (
	"context"
)

func (g *ZosAPI) statisticsGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.statisticsStub.GetCounters(ctx)
}
