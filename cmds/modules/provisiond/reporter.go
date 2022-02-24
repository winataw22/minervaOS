package provisiond

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/joncrlsn/dque"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	every = 60 * 60 // 1 hour
)

type many []error

func (m many) Error() string {
	return m.WithPrefix("")
}

func (m many) WithPrefix(p string) string {
	var buf bytes.Buffer
	for _, err := range m {
		if buf.Len() > 0 {
			buf.WriteRune('\n')
		}
		if err, ok := err.(many); ok {
			buf.WriteString(err.WithPrefix(p + " "))
			continue
		}

		buf.WriteString(err.Error())
	}

	return buf.String()
}

func (m many) append(err error) many {
	return append(m, err)
}

func (m many) AsError() error {
	if len(m) == 0 {
		return nil
	}

	return m
}

type Consumption struct {
	substrate.Consumption
	TwinID uint32
}

type Report struct {
	Consumption []Consumption
}

// Reporter structure
type Reporter struct {
	cl        zbus.Client
	nodeID    uint32
	identity  substrate.Identity
	engine    provision.Engine
	queue     *dque.DQue
	substrate substrate.Manager
}

func reportBuilder() interface{} {
	return &Report{}
}

// NewReporter creates a new capacity reporter
func NewReporter(engine provision.Engine, nodeID uint32, cl zbus.Client, root string) (*Reporter, error) {
	sub, err := environment.GetSubstrate()
	if err != nil {
		return nil, err
	}

	const queueName = "consumption"
	var queue *dque.DQue
	for i := 0; i < 3; i++ {
		queue, err = dque.NewOrOpen(queueName, root, 1024, reportBuilder)
		if err != nil {
			os.RemoveAll(filepath.Join(root, queueName))
			continue
		}
		break
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to setup report persisted queue")
	}

	idMgr := stubs.NewIdentityManagerStub(cl)
	sk := ed25519.PrivateKey(idMgr.PrivateKey(context.TODO()))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		cl:        cl,
		engine:    engine,
		nodeID:    nodeID,
		identity:  id,
		queue:     queue,
		substrate: sub,
	}, nil
}

func (r *Reporter) pushOne() ([]Consumption, error) {
	item, err := r.queue.PeekBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to peek into capacity queue. #properlyfatal")
	}

	report := item.(*Report)

	log.Info().Int("len", len(report.Consumption)).Msgf("sending capacity report")

	consumptions := make([]substrate.Consumption, 0, len(report.Consumption))
	for _, cmp := range report.Consumption {
		log.Debug().Uint64("contract", uint64(cmp.ContractID)).Msg("has consumption to report")
		consumptions = append(consumptions, cmp.Consumption)
	}
	sub, err := r.substrate.Substrate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to chain")
	}

	defer sub.Close()

	hash, err := sub.Report(r.identity, consumptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to publish consumption report")
	}

	log.Info().Str("hash", hash.Hex()).Msg("report block hash")

	// only removed if report is reported to substrate
	// remove item from queue
	_, err = r.queue.Dequeue()

	return report.Consumption, err
}

func (r *Reporter) pusher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// problem is pushOne is a blocker call. so if ctx is canceled
		// while we are inside pushOne, no way to detect that until the pushOne call
		// returns
		reported, err := r.pushOne()
		if err != nil {
			log.Error().Err(err).Msg("error while processing capacity report")
			select {
			case <-ctx.Done():
				return
			case <-time.After(20 * time.Second):
				continue
			}
		}

		log.Debug().Msg("capacity report pushed to chain")
		if r.queue.Size() == 0 {
			// we only synchronize once ALL queued reports are pushed.
			if err := r.synchronize(ctx, reported); err != nil {
				log.Error().Err(err).Msg("failed to synchronize active contracts")
			}
		}
	}
}

// synchronize will make sure that the node only runs
// active contracts.
func (r *Reporter) synchronize(ctx context.Context, reported []Consumption) error {
	log.Debug().Msg("synchronize active contracts")

	local := make(map[types.U64]Consumption)
	for _, report := range reported {
		local[report.ContractID] = report
	}

	sub, err := r.substrate.Substrate()
	if err != nil {
		return errors.Wrap(err, "failed to connect to chain")
	}

	defer sub.Close()

	// the idea here is that we bring ALL active node contracts from chain.
	// then compare it with what we have atm (the one we just reported)
	contracts, err := sub.GetNodeContracts(r.nodeID)
	if err != nil {
		return err
	}

	for _, contract := range contracts {
		// is there a consumption report for a contract
		delete(local, contract)
	}
	// any LOCAL contract that is not in the map must be decommissioned
	for _, local := range local {
		log.Debug().Uint64("contract", uint64(local.ContractID)).Msg("decomission contract because it has been deleted")
		if err := r.engine.Deprovision(ctx, local.TwinID, uint64(local.ContractID), "contract terminated"); err != nil {
			log.Error().Err(err).Msgf("failed to decomission contract(%d)", local.ContractID)
		}
	}

	return nil
}

// Run runs the reporter
func (r *Reporter) Run(ctx context.Context) error {
	// go over all user reservations
	// take into account the following:
	// every is in seconds.
	events := stubs.NewEventsStub(r.cl)
	stream, err := events.ContractCancelledEvent(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to register to node events")
	}

	go r.pusher(ctx)

	ticker := time.NewTicker(every * time.Second)
	defer ticker.Stop()

	// we always start by reporting capacity, and then once each
	// `every` seconds
report:
	for {
		log.Info().Msg("collecting consumption")
		// align time.
		u := time.Now().Unix()
		u = (u / every) * every
		// so any reservation that is deleted but this
		// happened 'after' this time stamp is still
		// considered as consumption because it lies in the current
		// 5 minute slot.
		// but if it was stopped before that timestamp, then it will
		// not be counted.
		report, err := r.collect(ctx, time.Unix(u, 0))
		if err != nil {
			log.Error().Err(err).Msg("failed to collect users consumptions")
			<-time.After(3 * time.Second)
			continue
		}

		log.Info().Int("size", len(report.Consumption)).Msg("queue consumption report for reproting")
		if err := r.push(report); err != nil {
			log.Error().Err(err).Msg("failed to push capacity report")
		}

		log.Info().Msg("consumption report queued")
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event := <-stream:
				if event.Kind == pkg.EventSubscribed {
					// TODO:
					// possible loss of events. either we synchronize
					// all contracts now. Or we wait until next timer

					// for now, wait until next report cycle
					continue
				}
				log.Debug().Msgf("received a cancel contract event %+v", event)

				// otherwise we know what contract to be deleted
				if err := r.engine.Deprovision(ctx, event.TwinId, event.Contract, "contract canceled"); err != nil {
					log.Error().Err(err).
						Uint32("twin", event.TwinId).
						Uint64("contract", event.Contract).
						Msg("failed to decomission contract")
				}
			case <-ticker.C:
				continue report
			}
		}
	}
}

func (r *Reporter) collect(ctx context.Context, since time.Time) (rep Report, err error) {
	users, err := r.engine.Storage().Twins()
	if err != nil {
		return rep, err
	}

	// to optimize we get ALL vms vmMetrics in one call.
	vmMetrics, err := stubs.NewVMModuleStub(r.cl).Metrics(ctx)
	if err != nil {
		return Report{}, errors.Wrap(err, "failed to get VMs network metrics")
	}

	gwMetrics, err := stubs.NewGatewayStub(r.cl).Metrics(ctx)
	if err != nil && !strings.Contains(err.Error(), "metrics not available") {
		return Report{}, errors.Wrap(err, "failed to get gateway metrics")
	}

	qsfsMetrics, err := stubs.NewQSFSDStub(r.cl).Metrics(ctx)
	if err != nil {
		return Report{}, errors.Wrap(err, "failed to get qsfs metrics")
	}

	for _, user := range users {
		cap, err := r.user(since, user, vmMetrics, gwMetrics, qsfsMetrics)
		if err != nil {
			log.Error().Err(err).Msg("failed to collect all user capacity")
			// NOTE: we intentionally not doing a 'continue' or 'return'
			// here because even if an error is returned we can have partially
			// collected some of the user consumption, we still can report that
		}

		rep.Consumption = append(rep.Consumption, cap...)
	}

	return rep, nil
}

func (r *Reporter) push(report Report) error {
	if len(report.Consumption) == 0 {
		return nil
	}
	return r.queue.Enqueue(&report)
}

func (r *Reporter) user(since time.Time, user uint32, vmMetrics pkg.MachineMetrics, gwMetrics pkg.GatewayMetrics, qsfsMetrics pkg.QSFSMetrics) ([]Consumption, error) {
	var m many

	var consumptions []Consumption
	ids, err := r.engine.Storage().ByTwin(user)
	if err != nil {
		m = m.append(errors.Wrapf(err, "failed to get reservation for user '%s'", user))
	}

	for _, id := range ids {
		dl, err := r.engine.Storage().Get(user, id)
		if err != nil {
			m = m.append(errors.Wrapf(err, "failed to get reservation '%d'", id))
			continue
		}

		consumption := Consumption{
			TwinID: user,
			Consumption: substrate.Consumption{
				ContractID: types.U64(id),
				Timestamp:  types.U64(since.Unix()),
			},
		}

		for i := range dl.Workloads {
			wl := &dl.Workloads[i]

			if wl.Result.IsNil() {
				// no results yet
				continue
			}

			wlID, err := gridtypes.NewWorkloadID(user, id, wl.Name)
			if err != nil {
				log.Error().Err(err).Msg("invalid workload id (shouldn't happen here)")
				continue
			}

			if !r.shouldCount(since, &wl.Result) {
				continue
			}

			cap, err := wl.Capacity()
			if err != nil {
				m = m.append(errors.Wrapf(err, "failed to get reservation '%s' capacity", id))
				continue
			}

			consumption.CRU += types.U64(cap.CRU)
			consumption.MRU += types.U64(cap.MRU)
			consumption.HRU += types.U64(cap.HRU)
			consumption.SRU += types.U64(cap.SRU)

			// special handling for ZMachine types. if they exist
			// we also need to get network usage.
			metric, ok := vmMetrics[wlID.String()]
			if ok {
				// add metric to consumption
				consumption.NRU += types.U64(r.computeNU(metric))
			}
			consumption.NRU += types.U64(qsfsMetrics.Nu(wlID.String()))
			// special handling for gw types.
			consumption.NRU += types.U64(gwMetrics.Nu(wlID.String()))
		}

		if !consumption.IsEmpty() {
			consumptions = append(consumptions, consumption)
		}
	}

	return consumptions, m.AsError()
}

func (r *Reporter) computeNU(m pkg.MachineMetric) gridtypes.Unit {
	const (
		// weights knobs for nu calculations
		public  float64 = 1.0
		private float64 = 0
	)

	nu := m.Public.Nu()*public + m.Private.Nu()*private

	return gridtypes.Unit(nu)
}

func (r *Reporter) shouldCount(since time.Time, result *gridtypes.Result) bool {
	if result.State == gridtypes.StateOk {
		return true
	}

	if result.State == gridtypes.StateDeleted {
		// if it was stopped before the 'since' .
		return !result.Created.Time().Before(since)
	}

	return false
}
