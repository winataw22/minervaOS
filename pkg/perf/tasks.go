package perf

import (
	"context"
)

type Task interface {
	ID() string
	Cron() string
	Description() string
	Run(ctx context.Context) (interface{}, error)
}
