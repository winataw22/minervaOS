package noded

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const module = "node"

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "noded",
	Usage: "reports the node total resources",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.BoolFlag{
			Name:  "id",
			Usage: "print node id and exit",
		},
		&cli.BoolFlag{
			Name:  "net",
			Usage: "print node network and exit",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		printID      bool   = cli.Bool("id")
		printNet     bool   = cli.Bool("net")
	)
	if app.CheckFlag(app.LimitedCache) {
		for app.CheckFlag(app.LimitedCache) {
			// relog the error in case it got lost
			log.Error().Msg("The node doesn't have ssd attached, it won't register.")
			time.Sleep(time.Minute * 5)
		}
	}

	env := environment.MustGet()

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	if printID {
		sysCl := stubs.NewSystemMonitorStub(redis)
		fmt.Println(sysCl.NodeID(cli.Context))
		return nil
	}

	if printNet {
		fmt.Println(env.RunningMode.String())
		return nil
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	oracle := capacity.NewResourceOracle(stubs.NewStorageModuleStub(redis))
	cap, err := oracle.Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}

	bus, err := rmb.New(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to initialize message bus server")
	}

	dmi, err := oracle.DMI()
	if err != nil {
		return errors.Wrap(err, "failed to get dmi information")
	}

	hypervisor, err := oracle.GetHypervisor()
	if err != nil {
		return errors.Wrap(err, "failed to get hypervisors")
	}

	bus.WithHandler("zos.system.version", func(ctx context.Context, payload []byte) (interface{}, error) {
		ver := stubs.NewVersionMonitorStub(redis)
		output, err := exec.CommandContext(ctx, "zinit", "-V").CombinedOutput()
		var zInitVer string
		if err != nil {
			zInitVer = err.Error()
		} else {
			zInitVer = strings.TrimSpace(strings.TrimPrefix(string(output), "zinit"))
		}

		version := struct {
			ZOS   string `json:"zos"`
			ZInit string `json:"zinit"`
		}{
			ZOS:   ver.GetVersion(ctx).String(),
			ZInit: zInitVer,
		}

		return version, nil
	})

	bus.WithHandler("zos.system.dmi", func(ctx context.Context, payload []byte) (interface{}, error) {
		return dmi, nil
	})

	bus.WithHandler("zos.system.hypervisor", func(ctx context.Context, payload []byte) (interface{}, error) {
		return hypervisor, nil
	})

	// answer calls for dmi
	go func() {
		if err := bus.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("message bus handler failure")
		}
	}()

	secureBoot, err := capacity.IsSecureBoot()
	if err != nil {
		log.Error().Err(err).Msg("failed to detect secure boot flags")
	}

	var info RegistrationInfo
	info = info.WithCapacity(cap).
		WithSerialNumber(dmi.BoardVersion()).
		WithSecureBoot(secureBoot).
		WithVirtualized(len(hypervisor) != 0)

	node, twin, err := registration(ctx, redis, env, info)
	if err != nil {
		return errors.Wrap(err, "failed during node registration")
	}

	sub, err := environment.GetSubstrate()
	if err != nil {
		return err
	}
	events := events.New(sub, node)

	system, err := monitord.NewSystemMonitor(node, 2*time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize system monitor")
	}

	host, err := monitord.NewHostMonitor(2 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize host monitor")
	}

	server.Register(zbus.ObjectID{Name: "host", Version: "0.0.1"}, host)
	server.Register(zbus.ObjectID{Name: "system", Version: "0.0.1"}, system)
	server.Register(zbus.ObjectID{Name: "events", Version: "0.0.1"}, events)

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("unexpected error")
		}
	}()

	log.Info().Uint32("node", node).Uint32("twin", twin).Msg("node registered")

	go func() {
		for {
			if err := public(ctx, node, env, redis); err != nil {
				log.Error().Err(err).Msg("setting public config failed")
				<-time.After(10 * time.Second)
			}
		}
	}()

	// uptime update
	go func() {
		for {
			if err := uptime(ctx, redis); err != nil {
				log.Error().Err(err).Msg("sending uptime failed")
				<-time.After(10 * time.Second)
			}
		}
	}()

	// reporting stats
	go func() {
		for {
			if err := reportStatistics(ctx, msgBrokerCon, redis); err != nil {
				log.Error().Err(err).Msg("sending uptime failed")
				<-time.After(10 * time.Second)
			}
		}
	}()

	log.Info().Uint32("twin", twin).Msg("node has been registered")
	log.Debug().Msg("start message bus")
	identityd := stubs.NewIdentityManagerStub(redis)
	sk := ed25519.PrivateKey(identityd.PrivateKey(ctx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return err
	}

	return runMsgBus(ctx, sub, id)
}
