# Perf

Perf is a performance test module that is scheduled to run and cache those tests results in redis which can be retrieved later over RMB call.

Perf tests are monitored by `noded` service from zos modules.

### Usage Example

1. Create a task that implement `Task` interface

```go
type LoggingTask struct {
	TaskID   string
	Schedule string // should be in cron syntax with seconds ("* 0 * * * *")
}

func (t *LoggingTask) ID() string {
	return t.TaskID
}

func (t *LoggingTask) Cron() string {
	return t.Schedule
}

// a simple task that returns a string with the current time
func (t *LoggingTask) Run(ctx context.Context) (interface{}, error) {
	result := fmt.Sprintf("time is: %v", time.Now())
	return result, nil
}
```

2. Add the created task to scheduler

```go
perfMon.AddTask(&perf.LoggingTask{
    TaskID:   "LoggingTask",
    Schedule: "* 0 * * * *", // when minutes is 0 (aka: every hour)
})
```
