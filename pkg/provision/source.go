package provision

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
)

var (
	// ErrPollEOS can be returned by a reservation poll to
	// notify the caller that it has reached end of stream
	// and next calls will not return any more data.
	ErrPollEOS = fmt.Errorf("end of stream")
)

// ReservationPoller define the interface to implement
// to poll the Explorer for new reservation
type ReservationPoller interface {
	// Poll ask the store to send us reservation for a specific node ID
	// from is the used as a filter to which reservation to use as
	// reservation.ID >= from. So a client to the Poll method should make
	// sure to call it with the last (MAX) reservation ID he receieved.
	Poll(nodeID pkg.Identifier, from uint64) (reservations []*Reservation, lastID uint64, err error)
}

// ReservationJob wraps a reservation type and has
// a boolean to indicate it is the last reservation
type ReservationJob struct {
	Reservation
	last bool
}

// PollSource does a long poll on address to get new and to be deleted
// reservations. the server should only return unique reservations
// stall the connection as long as possible if no new reservations
// are available.
func PollSource(store ReservationPoller, nodeID pkg.Identifier) ReservationSource {
	return &pollSource{
		store:    store,
		nodeID:   nodeID.Identity(),
		maxSleep: 10 * time.Second,
	}
}

type pollSource struct {
	store    ReservationPoller
	nodeID   string
	maxSleep time.Duration
}

func (s *pollSource) Reservations(ctx context.Context) <-chan *ReservationJob {

	ch := make(chan *ReservationJob)
	// On the first run we will get all the reservation ever made to this node to make sure we provision everything at boot.
	// After that, we only ask for the new reservations.
	go func() {
		defer close(ch)
		var next uint64
		var previousLastID uint64
		var triggerCleanup = true
		on := time.Now()
		log.Info().Msg("Start polling for reservations")
		slog := app.SampledLogger()

		for {
			time.Sleep(time.Until(on))
			on = time.Now().Add(s.maxSleep)

			slog.Info().Uint64("next", next).Msg("Polling for reservations")

			res, lastID, err := s.store.Poll(pkg.StrIdentifier(s.nodeID), next)
			if err != nil && err != ErrPollEOS {
				// if this is not a temporary error, then skip the reservation entirely
				// and try to get the next one
				if shouldRetry(err) {
					log.Error().Err(err).Uint64("next", next).Msg("failed to get reservation, retry same")
				} else {
					log.Error().Err(err).Uint64("next", next).Msg("failed to get reservation")
					next = lastID + 1
				}
				continue
			}

			next = lastID + 1

			select {
			case <-ctx.Done():
				return
			default:
				for _, r := range res {
					reservation := ReservationJob{
						*r,
						false,
					}
					ch <- &reservation
				}

				// if the explorer return twice the same last ID
				// it means we have processed all the existing reservation for now
				// it is safe to trigger a cleanup
				if triggerCleanup && previousLastID == lastID {
					ch <- &ReservationJob{
						Reservation: Reservation{},
						last:        true,
					}
					triggerCleanup = false
				}
			}

			previousLastID = lastID

			if err == ErrPollEOS {
				return
			}
		}
	}()

	return ch
}

// ReservationExpirer define the interface to implement
// to get all the reservation that have expired
type ReservationExpirer interface {
	// GetExpired returns all id the the reservations that are expired
	// at the time of the function call
	GetExpired() ([]*Reservation, error)
}

type decommissionSource struct {
	store ReservationExpirer
}

// NewDecommissionSource creates a ReservationSource that sends reservation
// that have expired into it's output channel
func NewDecommissionSource(store ReservationExpirer) ReservationSource {
	return &decommissionSource{
		store: store,
	}
}

func (s *decommissionSource) Reservations(ctx context.Context) <-chan *ReservationJob {
	log.Info().Msg("start decommission source")
	c := make(chan *ReservationJob)
	slog := app.SampledLogger()

	go func() {
		defer close(c)

		for {
			<-time.After(time.Second * 20) //TODO: make configuration ? default value ?

			slog.Info().Msg("check for expired reservation")

			reservations, err := s.store.GetExpired()
			if err != nil {
				log.Error().Err(err).Msg("error while getting expired reservation id")
				continue
			}

			select {
			case <-ctx.Done():
				return
			default:
				for _, r := range reservations {
					log.Info().
						Str("id", string(r.ID)).
						Str("type", string(r.Type)).
						Time("created", r.Created).
						Str("duration", fmt.Sprintf("%v", r.Duration)).
						Msg("reservation expired")

					reservation := ReservationJob{
						*r,
						false,
					}
					c <- &reservation
				}
			}
		}
	}()

	return c
}

type combinedSource struct {
	Sources []ReservationSource
}

// CombinedSource merge different ReservationSources into one ReservationSource
func CombinedSource(sources ...ReservationSource) ReservationSource {
	return &combinedSource{
		Sources: sources,
	}
}

func (s *combinedSource) Reservations(ctx context.Context) <-chan *ReservationJob {
	var wg sync.WaitGroup

	out := make(chan *ReservationJob)

	// Start an send goroutine for each input channel in cs. send
	// copies values from c to out until c is closed, then calls wg.Done.
	send := func(c <-chan *ReservationJob) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(s.Sources))
	for _, source := range s.Sources {
		c := source.Reservations(ctx)
		go send(c)
	}

	// Start a goroutine to close out once all the send goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// shouldRetry check if the error received from the reservation source
// is an error that should make use retry to get the same reservation or
// if we should skip it and ask the next one
func shouldRetry(err error) bool {
	var perr *net.OpError

	if ok := errors.As(err, &perr); ok {
		// retry for any network IO error
		return true
	}

	var hErr client.HTTPError
	if ok := errors.As(err, &hErr); ok {
		// retry for any response out of 2XX range
		resp := hErr.Response()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return true
		}
	}

	return false
}
