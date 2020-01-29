package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ReservationCache define the interface to store
// some reservations
type ReservationCache interface {
	Add(r *Reservation) error
	Get(id string) (*Reservation, error)
	Remove(id string) error
	Exists(id string) (bool, error)
	Counters() pkg.ProvisionCounters
}

// Feedbacker defines the method that needs to be implemented
// to send the provision result to BCDB
type Feedbacker interface {
	Feedback(id string, r *Result) error
	Deleted(id string) error
}

type defaultEngine struct {
	source ReservationSource
	store  ReservationCache
	fb     Feedbacker
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(source ReservationSource, rw ReservationCache, fb Feedbacker) Engine {
	return &defaultEngine{
		source: source,
		store:  rw,
		fb:     fb,
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
				Str("tag", reservation.Tag.String()).
				Logger()

			if reservation.Expired() || reservation.ToDelete {
				slog.Info().Msg("start decommissioning reservation")
				if err := e.decommission(ctx, reservation); err != nil {
					log.Error().Err(err).Msgf("failed to decommission reservation %s", reservation.ID)
					continue
				}
			} else {
				slog.Info().Msg("start provisioning reservation")
				if err := e.provision(ctx, reservation); err != nil {
					log.Error().Err(err).Msgf("failed to provision reservation %s", reservation.ID)
					continue
				}
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

	_, err := e.store.Get(r.ID)
	if err == nil {
		log.Info().Str("id", r.ID).Msg("reservation already deployed")
		return nil
	}

	result, err := fn(ctx, r)

	if replyErr := e.reply(ctx, r, err, result); replyErr != nil {
		log.Error().Err(replyErr).Msg("failed to send result to BCDB")
	}

	if err != nil {
		return err
	}

	if err := e.store.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	}

	return nil
}

func (e *defaultEngine) decommission(ctx context.Context, r *Reservation) error {
	fn, ok := decommissioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	exists, err := e.store.Exists(r.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to check if reservation %s exists in cache", r.ID)
	}

	if !exists {
		log.Info().Str("id", r.ID).Msg("reservation not provisioned, no need to decomission")
		if err := e.fb.Deleted(r.ID); err != nil {
			log.Error().Err(err).Str("id", r.ID).Msg("failed to mark reservation as deleted")
		}
		return nil
	}

	err = fn(ctx, r)
	if err != nil {
		return errors.Wrap(err, "decommissioning of reservation failed")
	}

	if err := e.store.Remove(r.ID); err != nil {
		return errors.Wrapf(err, "failed to remove reservation %s from cache", r.ID)
	}

	if err := e.fb.Deleted(r.ID); err != nil {
		return errors.Wrap(err, "failed to mark reservation as deleted")
	}

	return nil
}

func (e *defaultEngine) reply(ctx context.Context, r *Reservation, rErr error, info interface{}) error {
	log.Debug().Str("id", r.ID).Msg("sending reply for reservation")

	zbus := GetZBus(ctx)
	identity := stubs.NewIdentityManagerStub(zbus)
	result := &Result{
		Type:    r.Type,
		Created: time.Now(),
		ID:      r.ID,
	}

	if rErr != nil {
		log.Error().
			Err(rErr).
			Str("id", r.ID).
			Msgf("failed to apply provision")
		result.Error = rErr.Error()
		result.State = "error" //TODO: create enum
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", info)).
			Msgf("workload deployed")
		result.State = "ok"
	}

	br, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	b, err := result.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to convert the result to byte for signature")
	}

	sig, err := identity.Sign(b)
	if err != nil {
		return errors.Wrap(err, "failed to signed the result")
	}
	result.Signature = sig

	return e.fb.Feedback(r.ID, result)
}

func (e *defaultEngine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
			}

			select {
			case <-ctx.Done():
			case ch <- e.store.Counters():
			}
		}
	}()

	return ch
}
