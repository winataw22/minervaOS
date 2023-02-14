package utils

import (
	"context"
	"sync"
)

// Mark defined a placeholder where multiple routines can
// use to synchronize their operation. Routines calling Done
// will be blocked until one calls Signal. After that all calls
// to Done should return immediately
type Mark interface {
	Done(ctx context.Context) error
	Signal()
}

type mark struct {
	ch chan struct{}
	o  sync.Once
}

func NewMark() Mark {
	return &mark{ch: make(chan struct{})}
}

// Done blocks until either ctx times out (returns error)
// or Signal has been called
// if Signal was already called, return immediately
func (m *mark) Done(ctx context.Context) error {
	select {
	case <-m.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Signal the mark to release anyone blocked
// on Done
func (m *mark) Signal() {
	m.o.Do(func() {
		close(m.ch)
	})
}
