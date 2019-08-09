package portm

type PortAllocator interface {
	Reserve(ns string) (int, error)
	Release(ns string, port int) error
}
