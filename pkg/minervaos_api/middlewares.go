package zosapi

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func (g *ZosAPI) authorized(ctx context.Context, _ []byte) (context.Context, error) {
	user := peer.GetTwinID(ctx)
	if user != g.farmerID {
		return nil, fmt.Errorf("unauthorized")
	}

	return ctx, nil
}

func (g *ZosAPI) log(ctx context.Context, _ []byte) (context.Context, error) {
	env := peer.GetEnvelope(ctx)
	request := env.GetRequest()
	if request != nil {
		log.Debug().Str("command", request.Command).Msg("received rmb request")
	}
	return ctx, nil
}
