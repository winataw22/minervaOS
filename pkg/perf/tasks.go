package perf

import (
	"context"
)

type Task interface {
	ID() string
	Cron() string
	Run(ctx context.Context) (interface{}, error)
}
