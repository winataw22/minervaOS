package perf

import (
	"context"
)

type Task interface {
	ID() string
	Cron() string
	Description() string
	Jitter() uint32
	Run(ctx context.Context) (interface{}, error)
}
