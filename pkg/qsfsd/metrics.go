package qsfsd

import (
	"context"
	"path/filepath"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/vishvananda/netlink"
)

// Metrics gets running qsfs network metrics
func (q *QSFS) Metrics() (pkg.QSFSMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result := make(map[string]pkg.NetMetric)

	items, err := filepath.Glob(filepath.Join(q.mountsPath, "*"))
	if err != nil {
		return pkg.QSFSMetrics{}, errors.Wrap(err, "failed to list mounts directory")
	}
	for _, item := range items {
		name := filepath.Base(item)
		metrics, err := q.qsfsMetrics(ctx, name)
		if err != nil {
			log.Error().Err(err).Str("id", name).Msg("failed to get qsfs metrics")
			continue
		}
		result[name] = metrics
	}
	return pkg.QSFSMetrics{Consumption: result}, nil
}
func (q *QSFS) qsfsMetrics(ctx context.Context, wlID string) (pkg.NetMetric, error) {
	var m pkg.NetMetric
	networker := stubs.NewNetworkerStub(q.cl)
	nsName := networker.QSFSNamespace(ctx, wlID)
	netNs, err := namespace.GetByName(nsName)
	if err != nil {
		return m, errors.Wrap(err, "didn't find qsfs namespace")
	}
	defer netNs.Close()
	err = netNs.Do(func(_ ns.NetNS) error {
		m, err = metricsForNics([]string{"public", "ygg0"})
		return err
	})
	if err != nil {
		return m, errors.Wrap(err, "failed to read metrics")
	}
	return m, nil
}
func metricsForNics(nics []string) (pkg.NetMetric, error) {
	var m pkg.NetMetric
	for _, nic := range nics {
		link, err := netlink.LinkByName(nic)
		if err != nil {
			return m, errors.Wrapf(err, "couldn't get nic %s info", nic)
		}
		stats := link.Attrs().Statistics
		m.NetRxBytes += stats.RxBytes
		m.NetRxPackets += stats.RxPackets
		m.NetTxBytes += stats.TxBytes
		m.NetTxPackets += stats.TxPackets
	}

	return m, nil
}
