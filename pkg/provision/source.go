package provision

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/tools/client"
)

type pollSource struct {
	store    ReservationPoller
	nodeID   string
	maxSleep time.Duration
}

var (
	// ErrPollEOS can be returned by a reservation poll to
	// notify the caller that it has reached end of stream
	// and next calls will not return any more data.
	ErrPollEOS = fmt.Errorf("end of stream")
)

// ReservationPoller define the interface to implement
// to poll the BCDB for new reservation
type ReservationPoller interface {
	// Poll ask the store to send us reservation for a specific node ID
	// from is the used as a filter to which reservation to use as
	// reservation.ID >= from. So a client to the Poll method should make
	// sure to call it with the last (MAX) reservation ID he receieved.
	Poll(nodeID pkg.Identifier, from uint64) ([]*Reservation, error)
}

type reservationPoller struct {
	wl client.Workloads
}

// provisionOrder is used to sort the workload type
// in the right order for provisiond
var provisionOrder = map[ReservationType]int{
	DebugReservation:      0,
	NetworkReservation:    1,
	ZDBReservation:        2,
	VolumeReservation:     3,
	ContainerReservation:  4,
	KubernetesReservation: 5,
}

func (r *reservationPoller) Poll(nodeID pkg.Identifier, from uint64) ([]*Reservation, error) {
	list, err := r.wl.Workloads(nodeID.Identity(), from)
	if err != nil {
		return nil, err
	}

	result := make([]*Reservation, 0, len(list))
	for _, wl := range list {
		r, err := WorkloadToProvisionType(wl)
		if err != nil {
			return nil, err
		}

		result = append(result, r)
	}

	// sorts the workloads in the oder they need to be processed by provisiond
	sort.Slice(result, func(i int, j int) bool {
		return provisionOrder[result[i].Type] < provisionOrder[result[j].Type]
	})

	return result, nil
}

// ReservationPollerFromWorkloads returns a reservation poller from client.Workloads
func ReservationPollerFromWorkloads(wl client.Workloads) ReservationPoller {
	return &reservationPoller{wl: wl}
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

func (s *pollSource) Reservations(ctx context.Context) <-chan *Reservation {
	log.Info().Msg("start reservation http source")
	ch := make(chan *Reservation)

	// on the first run we will get all the reservation
	// ever made to this know, to make sure we provision
	// everything at boot
	// after that, we only ask for the new reservations
	go func() {
		defer close(ch)
		var next uint64
		on := time.Now()
		log.Info().Msg("Started polling for reservations")
		for {
			time.Sleep(time.Until(on))
			on = time.Now().Add(s.maxSleep)
			log.Info().Uint64("next", next).Msg("Polling for reservations")

			res, err := s.store.Poll(pkg.StrIdentifier(s.nodeID), next)
			if err != nil && err != ErrPollEOS {
				// if this is not a temporary error, then skip the reservation entirely
				// and try to get the next one
				if !isNetworkErr(err) {
					log.Error().Err(err).Uint64("next", next).Msg("failed to get reservation")
					next++
				} else {
					log.Error().Err(err).Uint64("next", next).Msg("failed to get reservation, retry same")
				}
				continue
			}

			select {
			case <-ctx.Done():
				return
			default:
				for _, r := range res {
					current, _, err := r.SplitID()
					if err != nil {
						log.Warn().Err(err).Str("id", r.ID).Msg("skipping reservation")
						continue
					}
					if current >= next {
						next = current + 1
					}
					ch <- r
				}
			}

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

func (s *decommissionSource) Reservations(ctx context.Context) <-chan *Reservation {
	log.Info().Msg("start decommission source")
	c := make(chan *Reservation)

	go func() {
		defer close(c)

		for {
			// <-time.After(time.Minute * 10) //TODO: make configuration ? default value ?
			<-time.After(time.Second * 20) //TODO: make configuration ? default value ?
			log.Debug().Msg("check for expired reservation")

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

func isNetworkErr(err error) bool {
	var perr *net.OpError
	return errors.As(err, &perr)
}
