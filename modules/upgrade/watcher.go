package upgrade

import (
	"context"
	"errors"
	"time"
)

// Upgrade represent an flist that contains new binaries and lib
// for 0-OS
type Upgrade struct {
	Flist         string // url of the upgrade flist
	Signature     string // signature of the upgrade flist
	TransactionID string //id of the upgrade transaction
}

var (
	// ErrNoUpgrade is return by a Publisher when no new upgrade is available
	ErrNoUpgrade = errors.New("no upgrade available")
)

// Publisher is the interface that define how the upgrade are published
type Publisher interface {
	// Check tries to find if an new upgrade is available
	// it should return ErrNoUpgrade if no upgrade as been found
	Check() (Upgrade, error)
}

type blockchainPublisher struct {
	ExplorerURLs []string
}

// NewBlockchainPublisher returns a Publisher that uses a blockchain as a source of upgrade
func NewBlockchainPublisher(explorerURLs []string) Publisher {
	return &blockchainPublisher{
		ExplorerURLs: explorerURLs,
	}
}

func (p *blockchainPublisher) Check() (Upgrade, error) {
	upgrade := Upgrade{}

	// TODO:
	return upgrade, nil
}

// Watcher provies a stream of Upgrade
type Watcher interface {
	// Watch starts a goroutine that will periodically watch an Upgrade publisher
	// and send the Upgrade object to the returned channel
	Watch(ctx context.Context, publisher Publisher) <-chan Upgrade
	// Error returns an error if the watcher stopped be because of an error
	// it return nil if the watcher has been stopped by the context cancellation
	// User needs to check the value of error after looping of the channel return by Watch
	Error() error
}

type perdiodicWatcher struct {
	period time.Duration
	err    error
}

// NewWatcher returns an upgrade Watcher
func NewWatcher(period time.Duration) Watcher {
	return &perdiodicWatcher{
		period: period,
		err:    nil,
	}
}

// Watch implements the Watcher interface
func (w *perdiodicWatcher) Watch(ctx context.Context, publisher Publisher) <-chan Upgrade {
	c := make(chan Upgrade)

	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				upgrade, err := publisher.Check()
				if err != nil {
					if err == ErrNoUpgrade {
						// no upgrade available
						time.Sleep(w.period)
						continue
					}
					w.err = err
					return
				}
				c <- upgrade
			}
		}
	}()

	return c
}

// Error returns an error if the watcher stopped be because of an error
// it return nil if the watcher has been stopped by the context cancellation
// User needs to check the value of error after looping of the channel return by Watch
func (w *perdiodicWatcher) Error() error {
	return w.err
}
