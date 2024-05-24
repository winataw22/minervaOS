package perf

import (
	"context"
	"math/rand"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/utils"
)

// PerformanceMonitor holds the module data
type PerformanceMonitor struct {
	scheduler  *gocron.Scheduler
	pool       *redis.Pool
	zbusClient zbus.Client
	tasks      []Task
}

var _ pkg.PerformanceMonitor = (*PerformanceMonitor)(nil)

// NewPerformanceMonitor returns PerformanceMonitor instance
func NewPerformanceMonitor(redisAddr string) (*PerformanceMonitor, error) {
	redisPool, err := utils.NewRedisPool(redisAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating new redis pool")
	}
	zbusClient, err := zbus.NewRedisClient(redisAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to zbus")
	}

	scheduler := gocron.NewScheduler(time.UTC)

	return &PerformanceMonitor{
		scheduler:  scheduler,
		pool:       redisPool,
		zbusClient: zbusClient,
		tasks:      []Task{},
	}, nil
}

// AddTask a simple helper method to add new tasks
func (pm *PerformanceMonitor) AddTask(task Task) {
	pm.tasks = append(pm.tasks, task)
}

// runTask runs the task and store its result
func (pm *PerformanceMonitor) runTask(ctx context.Context, task Task) error {
	if task.Jitter() != 0 {
		sleepInterval := time.Duration(rand.Int31n(int32(task.Jitter()))) * time.Second
		time.Sleep(sleepInterval)
	}

	res, err := task.Run(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to run task: %s", task.ID())
	}

	err = pm.setCache(ctx, pkg.TaskResult{
		Name:        task.ID(),
		Timestamp:   uint64(time.Now().Unix()),
		Description: task.Description(),
		Result:      res,
	})
	if err != nil {
		return errors.Wrap(err, "failed to set cache")
	}

	return nil
}

// Run adds the tasks to the cron queue and start the scheduler
func (pm *PerformanceMonitor) Run(ctx context.Context) error {
	ctx = withZbusClient(ctx, pm.zbusClient)
	for _, task := range pm.tasks {
		task := task
		if _, err := pm.scheduler.CronWithSeconds(task.Cron()).Do(func() error {
			return pm.runTask(ctx, task)
		}); err != nil {
			return errors.Wrapf(err, "failed to schedule the task: %s", task.ID())
		}

		go func(task Task) {
			ok, err := pm.exists(task.ID())
			if err != nil {
				log.Error().Err(err).Msgf("failed to find key %s", task.ID())
			}

			if !ok {
				if err := pm.runTask(ctx, task); err != nil {
					log.Error().Err(err).Msgf("failed to run task: %s", task.ID())
				}
			}
		}(task)

	}

	pm.scheduler.StartAsync()
	return nil
}

type zbusClient struct{}

func withZbusClient(ctx context.Context, client zbus.Client) context.Context {
	return context.WithValue(ctx, zbusClient{}, client)
}

// GetZbusClient gets zbus client from the given context
func GetZbusClient(ctx context.Context) zbus.Client {
	return ctx.Value(zbusClient{}).(zbus.Client)
}
