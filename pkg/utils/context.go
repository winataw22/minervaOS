package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var (
	// DefaultTerminateSignals default signals to handle
	// if not signals are provided
	// NOTE: we use the default st here instead of ALL signals
	// to avoid dying on signals like SIGCHLD and others that
	// are not intend to terminate the process
	DefaultTerminateSignals = []os.Signal{
		syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGQUIT,
	}
)

// WithSignal return a context that is canceld if any of the specified signals was
// received
func WithSignal(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc) {
	if len(sig) == 0 {
		sig = DefaultTerminateSignals
	}

	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, 3)

	signal.Notify(ch, sig...)

	go func() {
		<-ch
		cancel()
		signal.Stop(ch)
	}()

	return ctx, cancel
}

// OnDone registers a callback on a context when it's done
// The ctx.Err() is passed as is to the callback function
func OnDone(ctx context.Context, cb func(error)) {
	go func() {
		<-ctx.Done()
		cb(ctx.Err())
	}()
}
