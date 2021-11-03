package monitord

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/threefoldtech/zos/pkg"
)

var (
	_ pkg.SystemMonitor = (*systemMonitor)(nil)
)

// systemMonitor stream
type systemMonitor struct {
	duration time.Duration
	node     uint32
}

// NewSystemMonitor creates new system of system monitor
func NewSystemMonitor(node uint32, duration time.Duration) (pkg.SystemMonitor, error) {
	if duration == 0 {
		duration = 2 * time.Second
	}

	return &systemMonitor{duration: duration, node: node}, nil
}

func (m *systemMonitor) NodeID() uint32 {
	return m.node
}

// Memory starts memory monitor stream
func (m *systemMonitor) Memory(ctx context.Context) <-chan pkg.VirtualMemoryStat {
	ch := make(chan pkg.VirtualMemoryStat)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.duration):
				vm, err := mem.VirtualMemory()
				if err != nil {
					log.Error().Err(err).Msg("failed to read memory status")
					continue
				}
				result := pkg.VirtualMemoryStat(*vm)
				select {
				case ch <- result:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}

// CPU starts cpu monitor stream
func (m *systemMonitor) CPU(ctx context.Context) <-chan pkg.TimesStat {
	ch := make(chan pkg.TimesStat)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.duration):
				percents, err := cpu.PercentWithContext(ctx, 0, false)
				if err != nil {
					log.Error().Err(err).Msg("failed to read cpu usage percentage")
					continue
				}
				times, err := cpu.Times(false)
				if err != nil {
					log.Error().Err(err).Msg("failed to read cpu usage times")
					continue
				}

				result := pkg.TimesStat{
					TimesStat: times[0],
					Percent:   percents[0],
					Time:      time.Now(),
				}

				select {
				case ch <- result:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}

// Disks starts disk monitor stream
func (m *systemMonitor) Disks(ctx context.Context) <-chan pkg.DisksIOCountersStat {
	ch := make(chan pkg.DisksIOCountersStat)
	go func() {
		defer close(ch)
		disks, err := disk.PartitionsWithContext(ctx, false)
		if err != nil {
			log.Error().Err(err).Msg("failed to list machine disks")
			return
		}
		var names []string
		for _, device := range disks {
			names = append(names, device.Device)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.duration):
				now := time.Now()
				counter, err := disk.IOCountersWithContext(ctx, names...)
				if err != nil {
					log.Error().Err(err).Msg("failed to read IO counter for disks")
				}
				result := make(pkg.DisksIOCountersStat)
				for k, v := range counter {
					result[k] = pkg.DiskIOCountersStat{
						IOCountersStat: v,
						Time:           now,
					}
				}

				select {
				case ch <- result:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}

func (m *systemMonitor) rate(h map[string]uint64, k string, v uint64, since, now time.Time) uint64 {
	old, ok := h[k]
	h[k] = v
	if !ok {
		return 0
	}

	rate := float64(v-old) / float64(now.Sub(since)/time.Second)
	return uint64(rate)
}

func (m *systemMonitor) Nics(ctx context.Context) <-chan pkg.NicsIOCounterStat {
	ch := make(chan pkg.NicsIOCounterStat)
	go func() {
		defer close(ch)
		t := time.Now()
		out := make(map[string]uint64)
		in := make(map[string]uint64)

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-time.After(m.duration):
				counters, err := net.IOCountersWithContext(ctx, true)
				if err != nil {
					log.Error().Err(err).Msg("failed to get network counters")
					continue
				}
				var result pkg.NicsIOCounterStat
				for _, nic := range counters {
					result = append(result, pkg.NicIOCounterStat{
						IOCountersStat: nic,
						RateOut:        m.rate(out, nic.Name, nic.BytesSent, t, now),
						RateIn:         m.rate(in, nic.Name, nic.BytesRecv, t, now),
					})
				}

				t = now
				select {
				case ch <- result:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}
