package provision

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
)

type httpSource struct {
	store  ReservationPoller
	nodeID string
}

// ReservationPoller define the interface to implement
// to poll the BCDB for new reservation
type ReservationPoller interface {
	// Poll ask the store to send us reservation for a specific node ID
	// if all is true, the store sends all the reservation every registered for the node ID
	// otherwise it just sends reservation not pulled yet.
	Poll(nodeID modules.Identifier, all bool) ([]*Reservation, error)
}

// HTTPSource does a long poll on address to get new
// reservations. the server should only return unique reservations
// stall the connection as long as possible if no new reservations
// are available.
func HTTPSource(store ReservationPoller, nodeID modules.Identifier) ReservationSource {
	return &httpSource{
		store:  store,
		nodeID: nodeID.Identity(),
	}
}

func (s *httpSource) Reservations(ctx context.Context) <-chan *Reservation {
	log.Info().Msg("start reservation http source")
	ch := make(chan *Reservation)

	// on the first run we will get all the reservation
	// ever made to this know, to make sure we provision
	// everything at boot
	// after that, we only ask for the new reservations
	firstRun := true
	go func() {
		defer close(ch)
		for {
			// backing off of 1 second
			<-time.After(time.Second)
			log.Info().Msg("check for new reservations")

			res, err := s.store.Poll(modules.StrIdentifier(s.nodeID), firstRun)
			if err != nil {
				log.Error().Err(err).Msg("failed to get reservation")
				time.Sleep(time.Second * 10)
			}
			firstRun = false

			select {
			case <-ctx.Done():
				return
			default:
				for _, r := range res {
					ch <- r
				}
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

func (s *decommissionSource) Reservations(ctx context.Context) <-chan *Reservation {
	log.Info().Msg("start decommission source")
	c := make(chan *Reservation)

	go func() {
		defer close(c)

		for {
			// <-time.After(time.Minute * 10) //TODO: make configuration ? default value ?
			<-time.After(time.Second * 10) //TODO: make configuration ? default value ?
			log.Info().Msg("check for expired reservation")

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
					c <- r
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

func (s *combinedSource) Reservations(ctx context.Context) <-chan *Reservation {
	var wg sync.WaitGroup

	out := make(chan *Reservation)

	// Start an send goroutine for each input channel in cs. send
	// copies values from c to out until c is closed, then calls wg.Done.
	send := func(c <-chan *Reservation) {
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
