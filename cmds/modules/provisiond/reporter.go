package provisiond

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/joncrlsn/dque"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rrd"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	every           = 60 * 60 // 1 hour
	lastReportedKey = ".last-reported-ts"
)

type Report struct {
	Consumption []substrate.NruConsumption
}

// Reporter structure
type Reporter struct {
	cl  zbus.Client
	rrd rrd.RRD

	identity   substrate.Identity
	queue      *dque.DQue
	apiGateway *stubs.APIGatewayStub
}

func reportBuilder() interface{} {
	return &Report{}
}

func ReportChecks(metricsPath string) error {
	rrd, err := rrd.NewRRDBolt(metricsPath, 5*time.Minute, 24*time.Hour)
	if err != nil {
		return errors.Wrap(err, "failed to create metrics database")
	}

	if _, err := rrd.Slot(); err != nil {
		return err
	}

	return rrd.Close()
}

// NewReporter creates a new capacity reporter
func NewReporter(metricsPath string, cl zbus.Client, root string) (*Reporter, error) {
	idMgr := stubs.NewIdentityManagerStub(cl)
	sk := ed25519.PrivateKey(idMgr.PrivateKey(context.TODO()))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
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

	apiGateway := stubs.NewAPIGatewayStub(cl)

	rrd, err := rrd.NewRRDBolt(metricsPath, 5*time.Minute, 24*time.Hour)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metrics database")
	}

	return &Reporter{
		cl:         cl,
		rrd:        rrd,
		identity:   id,
		queue:      queue,
		apiGateway: apiGateway,
	}, nil
}

func (r *Reporter) pushOne() error {
	item, err := r.queue.PeekBlock()
	if err != nil {
		return errors.Wrap(err, "failed to peek into capacity queue. #properlyfatal")
	}

	report := item.(*Report)

	log.Info().Int("len", len(report.Consumption)).Msgf("sending capacity report")

	hash, err := r.apiGateway.Report(context.Background(), report.Consumption)
	if err != nil {
		return errors.Wrap(err, "failed to publish consumption report")
	}

	log.Info().Str("hash", hash.Hex()).Msg("report block hash")

	// only removed if report is reported to substrate
	// remove item from queue
	_, err = r.queue.Dequeue()

	return err
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
		err := r.pushOne()
		if err != nil {
			log.Error().Err(err).Msg("error while processing capacity report")
			select {
			case <-ctx.Done():
				return
			case <-time.After(20 * time.Second):
				continue
			}
		}
	}
}

// getVmMetrics will collect network consumption for vms and store it in the given slot
func (r *Reporter) getVmMetrics(ctx context.Context, slot rrd.Slot) error {
	log.Debug().Msg("collecting networking metrics")
	vmd := stubs.NewVMModuleStub(r.cl)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	metrics, err := vmd.Metrics(ctx)
	if err != nil {
		return err
	}

	for vm, consumption := range metrics {
		nu := r.computeNU(consumption)
		log.Debug().Str("vm", vm).Uint64("computed", uint64(nu)).Msgf("consumption: %+v", consumption)
		if err := slot.Counter(vm, float64(nu)); err != nil {
			return errors.Wrapf(err, "failed to store metrics for '%s'", vm)
		}
	}

	return nil
}

// getNetworkMetrics will collect network consumption for network resource and store it in the given slot
func (r *Reporter) getNetworkMetrics(ctx context.Context, slot rrd.Slot) error {
	log.Debug().Msg("collecting networking metrics")
	stub := stubs.NewNetworkerStub(r.cl)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	metrics, err := stub.Metrics(ctx)
	if err != nil {
		return err
	}

	for wl, consumption := range metrics {
		nu := consumption.Nu()
		log.Debug().Str("network", wl).Uint64("computed", uint64(nu)).Msgf("consumption: %+v", consumption)
		if err := slot.Counter(wl, float64(nu)); err != nil {
			return errors.Wrapf(err, "failed to store metrics for '%s'", wl)
		}
	}

	return nil
}

// getVmMetrics will collect network consumption every 5 min and store
// it in the rrd database.
func (r *Reporter) getGwMetrics(ctx context.Context, slot rrd.Slot) error {
	log.Debug().Msg("collecting networking metrics")
	gw := stubs.NewGatewayStub(r.cl)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	metrics, err := gw.Metrics(ctx)
	if err != nil {
		return err
	}
	requests := metrics.Request
	responses := metrics.Response

	sums := make(map[string]float64)
	for wl, v := range requests {
		sums[wl] = v + responses[wl]
		delete(responses, wl)
	}

	for wl, v := range responses {
		sums[wl] = v
	}

	for wl, nu := range sums {
		log.Debug().Str("gw", wl).Uint64("computed", uint64(nu)).Msg("consumption")
		if err := slot.Counter(wl, float64(nu)); err != nil {
			return errors.Wrapf(err, "failed to store metrics for '%s'", wl)
		}
	}

	return nil
}

func (r *Reporter) getMetrics(ctx context.Context) error {
	slot, err := r.rrd.Slot()
	if err != nil {
		return err
	}

	// NOTICE: disable collecting traffic consumption
	// for network resources. So ygg and wg traffic
	// will not counter.
	// To enable, uncomment the following section
	//
	// if err := r.getNetworkMetrics(ctx, slot); err != nil {
	// 	log.Error().Err(err).Msg("failed to get network resource consumption")
	// }

	if err := r.getVmMetrics(ctx, slot); err != nil {
		log.Error().Err(err).Msg("failed to get vm public ip consumption")
	}

	if err := r.getGwMetrics(ctx, slot); err != nil {
		log.Error().Err(err).Msg("failed to get gateway network consumption")
	}

	return nil
}

func (r *Reporter) metrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.getMetrics(ctx); err != nil {
				log.Error().Err(err).Msg("failed to collect network metrics")
			}
		}
	}
}

func (r *Reporter) setLastReportTime(ts int64) error {
	slot, err := r.rrd.Slot()
	if err != nil {
		return errors.Wrap(err, "failed to create rrd slot")
	}
	// we set it in storage anyway in case of reboot
	if err := slot.Counter(lastReportedKey, float64(ts)); err != nil {
		return errors.Wrap(err, "failed to set last report time")
	}

	return nil
}

func (r *Reporter) getLastReportTime() (int64, bool, error) {
	stored, ok, err := r.rrd.Last(lastReportedKey)
	if err != nil {
		return 0, false, errors.Wrap(err, "failed to get timestamp of last report")
	}
	return int64(stored), ok, nil
}

func (r *Reporter) Close() {
	_ = r.rrd.Close()
	_ = r.queue.Close()
}

// Run runs the reporter
func (r *Reporter) Run(ctx context.Context) error {
	// go over all user reservations
	// take into account the following:
	// every is in seconds.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go r.metrics(ctx)
	go r.pusher(ctx)

	// we always start by reporting capacity, and then once each
	// `every` seconds
	for {
		log.Debug().Msg("collecting consumption since last report")
		u := time.Now().Unix()

		lastReport, ok, err := r.getLastReportTime()
		if err != nil {
			return err
		}
		if !ok {
			log.Debug().Msg("no previous reports found")
			// no reports where made. we can't report consumption now
			// because we have no window. It won't be fair for the user.
			// (only during update of old nodes that has running workloads)
			// hence we only set last report to now
			lastReport = u
			if err := r.setLastReportTime(u); err != nil {
				return err
			}
		}
		log.Debug().Time("last-report", time.Unix(lastReport, 0)).Msg("time of last report")
		// compute when we should send next report.
		delay := (lastReport + every) - u

		if delay < 0 {
			delay = 0
		}
		// wait for delay
		log.Debug().Int64("duration", delay).Msg("seconds to wait before collecting consumption")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(delay) * time.Second):
		}

		since := time.Unix(lastReport, 0)
		log.Debug().Time("since", since).Msg("collecting consumption since")
		ts, err := r.report(ctx, since)
		if err != nil {
			return errors.Wrap(err, "failed to create consumption report")
		}

		if err := r.setLastReportTime(ts.Unix()); err != nil {
			return err
		}
	}
}

func (r *Reporter) report(ctx context.Context, since time.Time) (time.Time, error) {
	now := time.Now()
	window := now.Sub(since)
	values, err := r.rrd.Counters(since)
	if err != nil {
		return now, errors.Wrap(err, "failed to get stored metrics from rrd")
	}

	reports := make(map[uint64]substrate.NruConsumption)
	for key, value := range values {
		if key == lastReportedKey {
			continue
		}

		_, deployment, _, err := gridtypes.WorkloadID(key).Parts()
		if err != nil {
			log.Error().Err(err).Msgf("failed to parse metric key '%s'", key)
			continue
		}

		rep, ok := reports[deployment]
		if !ok {
			rep = substrate.NruConsumption{
				ContractID: types.U64(deployment),
				Timestamp:  types.U64(now.Unix()),
				Window:     types.U64(window / time.Second),
			}
		}

		rep.NRU += types.U64(value)
		reports[deployment] = rep
	}

	var report Report

	for _, v := range reports {
		if v.NRU == 0 {
			continue
		}
		report.Consumption = append(report.Consumption, v)
	}

	return now, r.push(report)
}

func (r *Reporter) push(report Report) error {
	if len(report.Consumption) == 0 {
		return nil
	}
	return r.queue.Enqueue(&report)
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
