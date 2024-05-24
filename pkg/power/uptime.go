package power

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/host"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	reportUptimeEvery = 40 * time.Minute
)

type Uptime struct {
	// Mark is set to done after the first uptime is sent
	Mark utils.Mark

	id               substrate.Identity
	substrateGateway *stubs.SubstrateGatewayStub
	m                sync.Mutex
}

func NewUptime(substrateGateway *stubs.SubstrateGatewayStub, id substrate.Identity) (*Uptime, error) {
	return &Uptime{
		id:               id,
		substrateGateway: substrateGateway,
		Mark:             utils.NewMark(),
	}, nil
}

func (u *Uptime) SendNow() (types.Hash, error) {
	if !isNodeHealthy() {
		log.Error().Msg("node is not healthy skipping uptime reports")
		return types.Hash{}, nil
	}

	// the mutex is to avoid race when SendNow is called
	// while the times reporting is working
	u.m.Lock()
	defer u.m.Unlock()

	// this can take sometime in case of connection problems
	// hence we first establish a connection THEN get the node
	// uptime.
	// to make sure the uptime is correct at the time of reporting
	uptime, err := host.Uptime()
	if err != nil {
		return types.Hash{}, errors.Wrap(err, "failed to get uptime")
	}

	return u.substrateGateway.UpdateNodeUptimeV2(context.Background(), uptime, uint64(time.Now().Unix()))
}

func (u *Uptime) uptime(ctx context.Context) error {
	for {
		log.Debug().Msg("updating node uptime")
		hash, err := u.SendNow()
		if err != nil {
			return errors.Wrap(err, "failed to report uptime")
		}

		u.Mark.Signal()

		log.Info().Str("hash", hash.Hex()).Msg("node uptime hash")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reportUptimeEvery):
			continue
		}
	}
}

// start uptime reporting. returns a channel that is closed immediately after
// the first uptime is reported.
func (u *Uptime) Start(ctx context.Context) {
	// uptime update
	defer log.Info().Msg("uptime reporting exited permanently")
	safeUptime := func(ctx context.Context) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("uptime reporting has panicked: %+v\n%s", p, string(debug.Stack()))
			}
		}()

		err = u.uptime(ctx)
		return err
	}

	for {
		err := safeUptime(ctx)
		if errors.Is(err, context.Canceled) {
			log.Info().Msg("stop uptime reporting. context cancelled")
			return
		} else if err != nil {
			log.Error().Err(err).Msg("sending uptime failed")
		} else {
			// context was cancelled
			return
		}
		// even there is no error we try again until ctx is cancelled
		<-time.After(10 * time.Second)
	}
}

func isNodeHealthy() bool {
	healthy := true
	if app.CheckFlag(app.ReadonlyCache) {
		log.Error().Msg("node cache is read only")
		healthy = false
	}
	if app.CheckFlag(app.LimitedCache) {
		log.Error().Msg("node is running on limited cache")
		healthy = false
	}
	if app.CheckFlag(app.NotReachable) {
		log.Error().Msg("node can not reach grid services")
		healthy = false
	}
	return healthy
}
