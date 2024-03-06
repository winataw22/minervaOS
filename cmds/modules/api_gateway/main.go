package apigateway

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	apigateway "github.com/threefoldtech/zos/pkg/api_gateway"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/urfave/cli/v2"
)

const module = "api-gateway"

// Module entry point
var Module cli.Command = cli.Command{
	Name:  module,
	Usage: "handles outgoing chain calls and incoming rmb calls",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.UintFlag{
			Name:  "workers",
			Usage: "number of workers `N`",
			Value: 1,
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		workerNr     uint   = cli.Uint("workers")
	)

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return fmt.Errorf("fail to connect to message broker server: %w", err)
	}
	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return fmt.Errorf("fail to connect to message broker server: %w", err)
	}
	idStub := stubs.NewIdentityManagerStub(redis)

	sk := ed25519.PrivateKey(idStub.PrivateKey(cli.Context))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	log.Info().Str("address", id.Address()).Msg("node address")
	if err != nil {
		return err
	}

	manager, err := environment.GetSubstrate()
	if err != nil {
		return fmt.Errorf("failed to create substrate manager: %w", err)
	}

	gw, err := apigateway.NewAPIGateway(manager, id)
	if err != nil {
		return fmt.Errorf("failed to create api gateway: %w", err)
	}

	server.Register(zbus.ObjectID{Name: "api-gateway", Version: "0.0.1"}, gw)

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting api-gateway module")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}
	return nil
}
