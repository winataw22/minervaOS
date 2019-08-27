package provision

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ReservationReadWriter define the interface to store
// some reservations
type ReservationReadWriter interface {
	Add(r *Reservation) error
	Remove(id string) error
}

type defaultEngine struct {
	source ReservationSource
	store  ReservationReadWriter
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(source ReservationSource, rw ReservationReadWriter) Engine {
	return &defaultEngine{
		source: source,
		store:  rw,
	}
}

// Run starts processing reservation resource. Then try to allocate
// reservations
func (e *defaultEngine) Run(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cReservation := e.source.Reservations(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("provision engine context done, exiting")
			return nil

		case reservation, ok := <-cReservation:
			if !ok {
				log.Info().Msg("reservation source is emptied. stopping engine")
				return nil
			}

			slog := log.With().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Str("duration", fmt.Sprintf("%v", reservation.Duration)).
				Logger()

			if !reservation.expired() {
				slog.Info().Msg("start provisioning reservation")
				if err := e.provision(ctx, reservation); err != nil {
					log.Error().Err(err).Msgf("failed to provision reservation %s", reservation.ID)
				}
			} else {
				slog.Info().Msg("start decommissioning reservation")
				if err := e.decommission(ctx, reservation); err != nil {
					log.Error().Err(err).Msgf("failed to decommission reservation %s", reservation.ID)
				}
				slog.Info().Msg("reservation decommission successful")
			}
		}
	}
}

func (e *defaultEngine) provision(ctx context.Context, r *Reservation) error {
	if err := r.validate(); err != nil {
		return errors.Wrapf(err, "failed validation of reservation")
	}

	fn, ok := provisioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	result, err := fn(ctx, r)
	if err != nil {
		return errors.Wrap(err, "provisioning of reservation failed")
	}

	if err := e.store.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	}

	e.reply(r.ID, result, err)
	return nil
}

func (e *defaultEngine) decommission(ctx context.Context, r *Reservation) error {
	fn, ok := decommissioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	err := fn(ctx, r)
	if err != nil {
		errors.Wrap(err, "decommissioning of reservation failed")
	}

	if err := e.store.Remove(r.ID); err != nil {
		errors.Wrapf(err, "failed to remove reservation %s from cache", r.ID)
	}
	return nil
}

func (e *defaultEngine) reply(id string, result interface{}, err error) {
	//TODO: actually push the reply to the endpoint defined by `to`
	if err != nil {
		log.Error().
			Err(err).
			Str("id", id).
			Msgf("failed to apply provision")

		return
	}

	log.Info().Str("reservation", id).Str("result", fmt.Sprint(result)).Msg("reservation result")
}
