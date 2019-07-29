package provision

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

type defaultEngine struct {
	source ReservationSource
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(source ReservationSource) Engine {
	return &defaultEngine{source}
}

// Run starts processing reservation resource. Then try to allocate
// reservations
func (e *defaultEngine) Run(ctx context.Context) error {
	for reservation := range e.source.Reservations(ctx) {
		log.Info().
			Str("id", string(reservation.ID)).
			Str("type", string(reservation.Type)).
			Msg("got reservation")

		if err := Verify(reservation); err != nil {
			log.Warn().
				Err(err).
				Str("id", string(reservation.ID)).
				Msg("verification of reservation signature failed")
			continue
		}

		fn, ok := types[reservation.Type]
		if !ok {
			log.Error().Str("type", string(reservation.Type)).Msgf("type of reservation not supported")
			continue
		}

		result, err := fn(ctx, reservation)
		if err != nil {
			log.Error().Err(err).Msgf("provisioning of reservation %s failed", reservation.ID)
		}
		e.reply(reservation.ID, result, err)
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
