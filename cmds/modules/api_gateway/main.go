package apigateway

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
	substrategw "github.com/threefoldtech/zos/pkg/substrate_gateway"
	"github.com/threefoldtech/zos/pkg/utils"
	zosapi "github.com/threefoldtech/zos/pkg/zos_api"
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

	router := peer.NewRouter()
	gw, err := substrategw.NewSubstrateGateway(manager, id)
	if err != nil {
		return fmt.Errorf("failed to create api gateway: %w", err)
	}

	server.Register(zbus.ObjectID{Name: "api-gateway", Version: "0.0.1"}, gw)

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	go func() {
		for {
			if err := server.Run(ctx); err != nil && err != context.Canceled {
				log.Error().Err(err).Msg("unexpected error")
				continue
			}

			break
		}
	}()

	api, err := zosapi.NewZosAPI(manager, redis, msgBrokerCon)
	if err != nil {
		return fmt.Errorf("failed to create zos api: %w", err)
	}
	api.SetupRoutes(router)

	pair, err := id.KeyPair()
	if err != nil {
		return err
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	backoff.Retry(func() error {
		_, err = peer.NewPeer(
			ctx,
			hex.EncodeToString(pair.Seed()),
			manager,
			router.Serve,
			peer.WithKeyType(peer.KeyTypeEd25519),
			peer.WithRelay(environment.MustGet().RelayURL...),
			peer.WithInMemoryExpiration(6*60*60), // 6 hours
		)
		if err != nil {
			return fmt.Errorf("failed to start a new rmb peer: %w", err)
		}

		return nil
	}, bo)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting api-gateway module")

	// block forever
	<-ctx.Done()
	return nil
}
