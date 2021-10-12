package qsfsd

import (
	"context"
	"fmt"
	"io/ioutil"
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
func (m *QSFS) Metrics() (pkg.QSFSMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	networker := stubs.NewNetworkerStub(m.cl)
	result := make(map[string]pkg.NetMetric)

	items, err := ioutil.ReadDir(m.mountsPath)
	if err != nil {
		return pkg.QSFSMetrics{}, errors.Wrap(err, "failed to list mounts directory")
	}
	for _, item := range items {
		if item.IsDir() {
			name := item.Name()
			nsName := networker.QSFSNamespace(ctx, name)
			netNs, err := namespace.GetByName(nsName)
			if err != nil {
				return pkg.QSFSMetrics{}, errors.Wrap(err, "didn't find qsfs namespace")
			}
			defer netNs.Close()
			metrics := pkg.NetMetric{}
			err = netNs.Do(func(_ ns.NetNS) error {
				metrics, err = metricsForNics([]string{"public", "ygg0"})
				return err
			})
			if err != nil {
				log.Error().Err(err).Msg(fmt.Sprintf("failed to read workload %s's metrics", name))
				continue
			}
			result[name] = metrics
		}
	}
	return pkg.QSFSMetrics{Consumption: result}, nil
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
