package powerd

import (
	"context"
	"crypto/ed25519"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/power"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/urfave/cli/v2"
)

const (
	module = "power"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "powerd",
	Usage: "handles the node power events",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
	)

	ctx, _ := utils.WithSignal(cli.Context)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	env := environment.MustGet()

	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(cl)
	register := stubs.NewRegistrarStub(cl)

	nodeID, err := register.NodeID(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get node id")
	}

	twinID, err := register.TwinID(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get node id")
	}

	sk := ed25519.PrivateKey(identity.PrivateKey(ctx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	log.Info().Str("address", id.Address()).Msg("node address")
	if err != nil {
		return err
	}

	sub, err := environment.GetSubstrate()
	if err != nil {
		return err
	}

	uptime, err := power.NewUptime(sub, id)
	if err != nil {
		return errors.Wrap(err, "failed to initialize uptime reported")
	}

	// start uptime reporting
	go uptime.Start(cli.Context)

	// if the feature is globally enabled try to ensure
	// wake on lan is set correctly.
	// then override the enabled flag
	enabled, err := power.EnsureWakeOnLan(cli.Context)
	if err != nil {
		return errors.Wrap(err, "failed to enable wol")
	}

	if !enabled {
		// if the zos nics don't support wol we can automatically
		// disable the feature
		log.Info().Msg("no wol support found by zos nic")
	}

	consumer, err := events.NewConsumer(msgBrokerCon, module)
	if err != nil {
		return errors.Wrap(err, "failed to to create event consumer")
	}

	// start power manager
	power, err := power.NewPowerServer(cl, sub, consumer, enabled, env.FarmID, nodeID, twinID, id, uptime)
	if err != nil {
		return errors.Wrap(err, "failed to initialize power manager")
	}

	if err := power.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
