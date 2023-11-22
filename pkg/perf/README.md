# Perf

Perf is a performance test module that is scheduled to run and cache those tests results in redis which can be retrieved later over RMB call.

Perf tests are monitored by `noded` service from zos modules.

### Usage Example

1. Create a task that implement `Task` interface

```go
type LoggingTask struct {
	taskID      string
	schedule    string // should be in cron syntax with seconds ("* 0 * * * *")
	description string
}

func (t *LoggingTask) ID() string {
	return t.taskID
}

func (t *LoggingTask) Cron() string {
	return t.schedule
}

func (t *LoggingTask) Description() string {
	return t.description
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
	taskID:      "LoggingTask",
	schedule:    "* 0 * * * *", // when minutes is 0 (aka: every hour)
	description: "Simple task that logs the time every hour",
})
```
