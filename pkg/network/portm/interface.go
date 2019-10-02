package portm

// PortAllocator is the interface that defines
// the behavior to reserve a port in a specific
// network namespace
type PortAllocator interface {
	Reserve(ns string) (int, error)
	Release(ns string, port int) error
}
