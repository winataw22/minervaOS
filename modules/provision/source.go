package provision

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/threefoldtech/zosv2/modules/identity"

	"github.com/rs/zerolog/log"
)

type pipeSource struct {
	p string
}

// FifoSource reads reservations from a fifo file
func FifoSource(p string) (ReservationSource, error) {
	err := syscall.Mkfifo(p, 0600)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	return &pipeSource{p}, nil
}

func (s *pipeSource) readToEnd(ctx context.Context, dec *json.Decoder, ch chan<- Reservation) error {
	var res Reservation
	// problem here that this will block until
	// something is pushed on the file, even
	// if context was canceled
	for {
		err := dec.Decode(&res)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		select {
		case ch <- res:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *pipeSource) Reservations(ctx context.Context) <-chan Reservation {
	ch := make(chan Reservation)
	go func() {
		defer func() {
			close(ch)
		}()

		for {
			file, err := os.Open(s.p)
			if err != nil {
				log.Error().Err(err).Msgf("failed to open pipe")
				break
			}

			dec := json.NewDecoder(file)
			err = s.readToEnd(ctx, dec, ch)
			file.Close()

			if err != nil {
				log.Error().Err(err).Msgf("failed to decode reservation item")
			}
		}
	}()

	return ch
}

type httpSource struct {
	store  ReservationStore
	nodeID string
}

// HTTPSource does a long poll on address to get new
// reservations. the server should only return unique reservations
// stall the connection as long as possible if no new reservations
// are available.
func HTTPSource(store ReservationStore, nodeID identity.Identifier) ReservationSource {
	return &httpSource{
		store:  store,
		nodeID: nodeID.Identity(),
	}
}

func (s *httpSource) Reservations(ctx context.Context) <-chan Reservation {
	ch := make(chan Reservation)
	go func() {
		defer close(ch)
		for {
			// backing off of 1 second
			<-time.After(time.Second)
			res, err := s.store.Poll(identity.StrIdentifier(s.nodeID), false)
			if err != nil {
				log.Error().Err(err).Msg("failed to get reservation")
				continue
			}

			select {
			case <-ctx.Done():
				break
			default:
				for _, r := range res {
					ch <- *r
				}
			}
		}

	}()
	return ch
}

type compinedSource struct {
	s1 ReservationSource
	s2 ReservationSource
}

// CompinedSource compines 2 different sources into one source
// Then it can be nested for any number of sources
func CompinedSource(s1, s2 ReservationSource) ReservationSource {
	return &compinedSource{s1, s2}
}

func (s *compinedSource) Reservations(ctx context.Context) <-chan Reservation {
	ch := make(chan Reservation)
	go func() {
		defer close(ch)

		ch1 := s.s1.Reservations(ctx)
		ch2 := s.s2.Reservations(ctx)

		for {
			var res Reservation
			select {
			case res = <-ch1:
			case res = <-ch2:
			case <-ctx.Done():
				return
			}

			select {
			case ch <- res:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch

}
