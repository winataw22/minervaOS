package qsfsd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/stubs"
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
				dir, err := ioutil.TempDir("/tmp", "qsfs-sysfs")
				if err != nil {
					return errors.Wrap(err, "coudln't create temp dir")
				}
				defer func() {
					if err := os.RemoveAll(dir); err != nil {
						log.Error().Err(err).Msg(fmt.Sprintf("qsfs metrics: couldn't remove: %s", dir))
					}
				}()
				if err := syscall.Mount("newns", dir, "sysfs", 0, ""); err != nil {
					return errors.Wrap(err, "couldn't mount sysfs")
				}
				defer func() {
					if err := syscall.Unmount(dir, syscall.MNT_DETACH); err != nil {
						log.Error().Err(err).Msg("qsfs metrics: couldn't detach sysfs: %s")
					}
				}()

				metrics, err = metricsForNics(dir, []string{"public", "ygg0"})
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

func readFileUint64(p string) (uint64, error) {
	bytes, err := ioutil.ReadFile(p)
	if err != nil {
		// we do skip but may be this is not crre
		return 0, err
	}

	return strconv.ParseUint(strings.TrimSpace(string(bytes)), 10, 64)
}

func metricsForNics(sysfsPath string, nics []string) (m pkg.NetMetric, err error) {
	template := filepath.Join(sysfsPath, "class/net/%s/statistics/")
	dict := map[string]*uint64{
		"rx_bytes":   &m.NetRxBytes,
		"rx_packets": &m.NetRxPackets,
		"tx_bytes":   &m.NetTxBytes,
		"tx_packets": &m.NetTxPackets,
	}
	for _, nic := range nics {
		base := fmt.Sprintf(template, nic)
		for metric, ptr := range dict {
			path := filepath.Join(base, metric)
			value, err := readFileUint64(path)
			if err != nil {
				log.Error().Err(err).Str("path", path).Msg("failed to read statistics")
				continue
			}

			*ptr += value
		}
	}

	return
}
