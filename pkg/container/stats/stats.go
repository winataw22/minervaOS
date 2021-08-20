package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	v1 "github.com/containerd/cgroups/stats/v1"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/typeurl"
	"github.com/rs/zerolog/log"
)

// StatsPushInterval defines how many times we push metrics
const StatsPushInterval = 2 * time.Second

// Metrics define a one-shot set of stats with differents metrics
type Metrics struct {
	Timestamp   int64  `json:"timestamp"`
	MemoryUsage uint64 `json:"memory_usage"`
	MemoryLimit uint64 `json:"memory_limit"`
	MemoryCache uint64 `json:"memory_cache"`
	CPUUsage    uint64 `json:"cpu_usage"`
	PidsCurrent uint64 `json:"pids_current"`
}

// Stats defines a stats backend
type Stats struct {
	Type string `bson:"type" json:"type"`
	Data Redis  `bson:"data" json:"data"`
}

// Monitor enable continuous metric fetching and forwarding to a backend
func Monitor(addr string, ns string, id string, backend io.WriteCloser) error {
	log.Info().Msg("fetching metrics")

	client, err := containerd.New(addr)
	if err != nil {
		log.Error().Err(err).Msg("metric client")
		return err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	container, err := client.LoadContainer(ctx, string(id))
	if err != nil {
		log.Error().Err(err).Msg("metric container")
		return err
	}

	for {
		task, err := container.Task(ctx, nil)
		if err != nil {
			// container probably down
			log.Error().Err(err).Msg("stopping metric task")
			return err
		}

		// fetching metric
		b, err := monitor(ctx, task)
		if err != nil {
			log.Error().Err(err).Msg("metric fetching")
			return err
		}

		// sending metric to the backend
		if _, err := backend.Write(b); err != nil {
			log.Error().Err(err).Msg("failed to send metrics to backend")
		}

		time.Sleep(StatsPushInterval)
	}
}

func monitor(ctx context.Context, task containerd.Task) ([]byte, error) {
	metric, err := task.Metrics(ctx)
	if err != nil {
		log.Error().Err(err).Msg("metrics")
		return nil, err
	}

	anydata, err := typeurl.UnmarshalAny(metric.Data)
	if err != nil {
		return nil, err
	}

	var data *v1.Metrics
	switch v := anydata.(type) {
	case *v1.Metrics:
		data = v
	default:
		return nil, fmt.Errorf("wrong metric type")
	}

	s := &Metrics{
		Timestamp:   metric.Timestamp.Unix(),
		MemoryUsage: data.Memory.Usage.Usage,
		MemoryLimit: data.Memory.Usage.Limit,
		MemoryCache: data.Memory.TotalCache,
		CPUUsage:    data.CPU.Usage.Total,
		PidsCurrent: data.Pids.Current,
	}

	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}
