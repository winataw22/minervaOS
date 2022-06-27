package noded

import (
	"context"
	"crypto/ed25519"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	reportUptimeEvery = 2 * time.Hour
)

func uptime(ctx context.Context, cl zbus.Client) error {
	var (
		mgr = stubs.NewIdentityManagerStub(cl)
	)

	subMgr, err := environment.GetSubstrate()
	if err != nil {
		return err
	}

	busCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	sk := ed25519.PrivateKey(mgr.PrivateKey(busCtx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return err
	}

	update := func(uptime uint64) (types.Hash, error) {
		sub, err := subMgr.Substrate()
		if err != nil {
			return types.Hash{}, err
		}
		defer sub.Close()
		return sub.UpdateNodeUptime(id, uptime)
	}

	for {
		uptime, err := host.Uptime()
		if err != nil {
			return errors.Wrap(err, "failed to get uptime")
		}
		log.Debug().Msg("updating node uptime")
		hash, err := update(uptime)
		if err != nil {
			return errors.Wrap(err, "failed to report uptime")
		}

		log.Info().Str("hash", hash.Hex()).Msg("node uptime hash")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reportUptimeEvery):
			continue
		}
	}
}
