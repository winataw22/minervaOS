package pkg

//go:generate zbusc -module node -version 0.0.1 -name performance-monitor -package stubs github.com/threefoldtech/zos/pkg+PerformanceMonitor stubs/performance_monitor_stub.go

type PerformanceMonitor interface {
	Get(taskName string) (TaskResult, error)
	GetAll() ([]TaskResult, error)
}

// TaskResult the result test schema
type TaskResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Timestamp   uint64      `json:"timestamp"`
	Result      interface{} `json:"result"`
}
