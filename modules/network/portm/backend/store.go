package backend

type Store interface {
	Lock() error
	Unlock() error
	Reserve(ns string, port int) (bool, error)
	Release(ns string, port int) error
	LastReserved(ns string) (int, error)
	GetByNS(ns string) ([]int, error)
	Close() error
}
