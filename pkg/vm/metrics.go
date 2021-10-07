package vm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/vishvananda/netlink"
)

var (
	errMacVTapNotFound = errors.New("macvtap not found")
)

// Metrics gets running machines network metrics
func (m *Module) Metrics() (pkg.MachineMetrics, error) {
	vms, err := FindAll()
	if err != nil {
		return nil, err
	}
	result := pkg.MachineMetrics{}
	for name, ps := range vms {
		metric, err := m.metrics(ps)
		if err != nil {
			log.Error().Err(err).Int("pid", ps.Pid).Msg("failed to get metrics for CH process")
			continue
		}
		result[name] = metric
	}

	return result, nil
}

func (m *Module) metrics(ps Process) (pkg.MachineMetric, error) {
	// from the pid we need the following:
	// - parse net arguments list
	// if tap, add this to list of taps to be monitored
	// if also an fd, readlink
	// look up name from link index
	// add to list ot taps to be monitored
	// collect all metrics of a vm
	nics, ok := ps.GetParam("--net")
	if !ok {
		return pkg.MachineMetric{}, fmt.Errorf("failed to parse cloud-hypervisor net config: %d", ps.Pid)
	}

	//--net tap=t-BQ4Q6CG4raLk6 fd=3,mac=6e:80:13:58:26:40
	parse := func(s string) map[string]string {
		values := make(map[string]string)
		pairs := strings.Split(s, ",")
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				values[kv[0]] = kv[1]
			}
		}
		return values
	}

	var priv []string
	var pub []string
	for _, nic := range nics {
		params := parse(nic)

		if tap, ok := params["tap"]; ok {
			priv = append(priv, tap)
		} else if fd, ok := params["fd"]; ok {
			// fd is an open file descriptor
			// so first, we readlink of the open fd
			// then parse the index and get the device
			// then get the name
			fdInt, err := strconv.Atoi(fd)
			if err != nil {
				log.Error().Err(err).
					Int("pid", ps.Pid).
					Str("net", nic).
					Msg("failed to parse fd")
				continue
			}

			tap, err := readMacVTap(ps.Pid, fdInt)
			if errors.Is(err, errMacVTapNotFound) {
				continue
			} else if err != nil {
				log.Error().Err(err).
					Int("pid", ps.Pid).
					Str("net", nic).
					Msg("failed to get macvtap for public ip")
				continue
			}

			pub = append(pub, tap)
		} else {
			log.Warn().
				Int("pid", ps.Pid).
				Str("net", nic).
				Msg("failed to parse net config for process")
		}
	}

	var metrics pkg.MachineMetric
	var err error
	metrics.Private, err = metricsForNics(priv)
	if err != nil {
		return pkg.MachineMetric{}, errors.Wrap(err, "failed to get metrics for vm private traffic")
	}
	metrics.Public, err = metricsForNics(pub)
	if err != nil {
		return pkg.MachineMetric{}, errors.Wrap(err, "failed to get metrics for vm public traffic")
	}

	return metrics, nil
}

func readFileUint64(p string) (uint64, error) {
	bytes, err := ioutil.ReadFile(p)
	if err != nil {
		// we do skip but may be this is not crre
		return 0, err
	}

	return strconv.ParseUint(strings.TrimSpace(string(bytes)), 10, 64)
}

func metricsForNics(nics []string) (m pkg.NetMetric, err error) {
	const template = "/sys/class/net/%s/statistics/"
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

func readMacVTap(pid, fd int) (string, error) {
	const template = "/proc/%d/fd/%d"

	path := fmt.Sprintf(template, pid, fd)

	target, err := os.Readlink(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read link of '%s'", path)
	}

	// the target should be in this format /dev/tap544
	var index int
	if _, err := fmt.Sscanf(target, "/dev/tap%d", &index); err != nil {
		return "", errors.Wrap(err, "failed to parse tap index")
	}

	link, err := netlink.LinkByIndex(index)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return "", errMacVTapNotFound
	} else if err != nil {
		return "", errors.Wrap(err, "failed to get the error")
	}

	return link.Attrs().Name, nil
}
