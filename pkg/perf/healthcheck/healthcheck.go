package healthcheck

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	id          = "healthcheck"
	schedule    = "0 */20 * * * *"
	description = "health check task runs multiple checks to ensure the node is in a usable state and set flags for the power daemon to stop reporting uptime if it is not usable"
)

// NewTask returns a new health check task.
func NewTask() perf.Task {
	checks := map[string]checkFunc{
		"cache": cacheCheck,
	}
	return &healthcheckTask{
		checks: checks,
		errors: make(map[string][]string),
	}
}

type checkFunc func(context.Context) []error

type healthcheckTask struct {
	checks map[string]checkFunc
	errors map[string][]string
}

var _ perf.Task = (*healthcheckTask)(nil)

// ID returns task ID.
func (h *healthcheckTask) ID() string {
	return id
}

func (h *healthcheckTask) Jitter() uint32 {
	return 0
}

// Cron returns task cron schedule.
func (h *healthcheckTask) Cron() string {
	return schedule
}

// Description returns task description.
func (h *healthcheckTask) Description() string {
	return description
}

// Run executes the health checks.
func (h *healthcheckTask) Run(ctx context.Context) (interface{}, error) {
	log.Debug().Msg("starting health check task")
	for k := range h.errors {
		// reset errors on each run
		h.errors[k] = nil
	}

	for label, check := range h.checks {
		errors := check(ctx)
		if len(errors) == 0 {
			continue
		}
		stringErrs := errorsToStrings(errors)
		h.errors[label] = append(h.errors[label], stringErrs...)
	}

	cl := perf.GetZbusClient(ctx)
	zui := stubs.NewZUIStub(cl)

	for label, data := range h.errors {
		err := zui.PushErrors(ctx, label, data)
		if err != nil {
			return nil, err
		}
	}
	return h.errors, nil
}

func errorsToStrings(errs []error) []string {
	s := make([]string, 0, len(errs))
	for _, err := range errs {
		s = append(s, err.Error())
	}
	return s
}

func cacheCheck(ctx context.Context) []error {
	var errors []error
	if err := readonlyCheck(ctx); err != nil {
		errors = append(errors, err)
	}
	return errors
}

func readonlyCheck(ctx context.Context) error {
	const checkFile = "/var/cache/healthcheck"

	_, err := os.Create(checkFile)
	if err != nil {
		if err := app.SetFlag(app.ReadonlyCache); err != nil {
			log.Error().Err(err).Msg("failed to set readonly flag")
		}
		return fmt.Errorf("failed to write to cache: %w", err)
	}
	defer os.Remove(checkFile)

	if err := app.DeleteFlag(app.ReadonlyCache); err != nil {
		log.Error().Err(err).Msg("failed to delete readonly flag")
	}
	return nil
}
