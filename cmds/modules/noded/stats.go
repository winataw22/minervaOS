package noded

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	operationTimeout     = 1 * time.Minute
	ReportInterval       = 5 * time.Minute
	ReportMaxElapsedTime = 3 * time.Minute // must be less than report interval
	CyclesToUpdate       = 3
)

func fillCapacityAndVersion(ctx context.Context, report *client.NodeStatus, cl zbus.Client) {
	ctx2, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	report.ZosVersion = stubs.NewVersionMonitorStub(cl).GetVersion(ctx2).String()
	report.Current = stubs.NewStatisticsStub(cl).Current(ctx2)
}

func reportStatistics(ctx context.Context, redis string, cl zbus.Client) error {
	stats := stubs.NewStatisticsStub(cl)
	total := stats.Total(ctx)
	oracle := capacity.NewResourceOracle(stubs.NewStorageModuleStub(cl))
	hypervisor, err := oracle.GetHypervisor()
	if err != nil {
		return errors.Wrap(err, "failed to get hypervisors")
	}
	bus, err := rmb.NewClient(redis)
	if err != nil {
		return errors.Wrap(err, "couldn't get an rmb bus instance")
	}
	updateCounter := CyclesToUpdate
	extended, err := environment.GetConfig()
	if err != nil {
		return err
	}
	for {
		if updateCounter == 0 {
			extended, err = environment.GetConfig()
			if err != nil {
				log.Error().Err(err).Msg("couldn't get twins to report to")
			}
			updateCounter = CyclesToUpdate
		}
		updateCounter--
		report := client.NodeStatus{
			Total:      total,
			Hypervisor: hypervisor,
		}
		fillCapacityAndVersion(ctx, &report, cl)
		for _, twin := range extended.Monitor {
			go func(twinID uint32) {
				log.Debug().Uint32("twin", twinID).Msg("sending status update to twin")
				cl := client.NewProxyClient(twinID, bus)
				if err := sendStatisticsReport(ctx, cl, report); err != nil {
					log.Error().Err(err).Uint32("twin", twinID).Msg("couldn't send report to twin")
				}
			}(twin)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(ReportInterval):
			continue
		}
	}
}

func sendStatisticsReport(ctx context.Context, cl *client.ProxyClient, report client.NodeStatus) error {
	ctx2, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	errHandler := func(err error, t time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while sending twin report")
		}
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 10 * time.Second
	exp.MaxElapsedTime = ReportMaxElapsedTime
	return backoff.RetryNotify(func() error {
		return cl.ReportStats(ctx2, report)
	}, exp, errHandler)

}
