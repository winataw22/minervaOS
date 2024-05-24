package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
)

func (g *ZosAPI) perfGetHandler(ctx context.Context, payload []byte) (interface{}, error) {
	type Payload struct {
		Name string
	}
	var request Payload
	err := json.Unmarshal(payload, &request)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload %v: %w", payload, err)
	}
	return g.performanceMonitorStub.Get(ctx, request.Name)
}

func (g *ZosAPI) perfGetAllHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.performanceMonitorStub.GetAll(ctx)
}
