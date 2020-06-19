package provision

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Engine is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type Engine struct {
	nodeID         string
	source         ReservationSource
	cache          ReservationCache
	feedback       Feedbacker
	provisioners   map[ReservationType]ProvisionerFunc
	decomissioners map[ReservationType]DecomissionerFunc
	signer         Signer
	statser        Statser
}

// EngineOps are the configuration of the engine
type EngineOps struct {
	// NodeID is the identity of the system running the engine
	NodeID string
	// Source is responsible to retrieve reservation for a remote source
	Source ReservationSource
	// Feedback is used to send provision result to the source
	// after the reservation is provisionned
	Feedback Feedbacker
	// Cache is a used to keep track of which reservation are provisionned on the system
	// and know when they expired so they can be decommissioned
	Cache ReservationCache
	// Provisioners is a function map so the engine knows how to provision the different
	// workloads supported by the system running the engine
	Provisioners map[ReservationType]ProvisionerFunc
	// Decomissioners contains the opposite function from Provisioners
	// they are used to decomission workloads from the system
	Decomissioners map[ReservationType]DecomissionerFunc
	// Signer is used to authenticate the result send to the source
	Signer Signer
	// Statser is responsible to keep track of how much workloads and resource units
	// are reserved on the system running the engine
	// After each provision/decomission the engine sends statistics update to the staster
	Statser Statser
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(opts EngineOps) *Engine {
	return &Engine{
		nodeID:         opts.NodeID,
		source:         opts.Source,
		cache:          opts.Cache,
		feedback:       opts.Feedback,
		provisioners:   opts.Provisioners,
		decomissioners: opts.Decomissioners,
		signer:         opts.Signer,
		statser:        opts.Statser,
	}
}

// Run starts reader reservation from the Source and handle them
func (e *Engine) Run(ctx context.Context) error {
	if err := e.cache.Sync(e.statser); err != nil {
		return fmt.Errorf("failed to synchronize statser: %w", err)
	}

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

			expired := reservation.Expired()
			slog := log.With().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Str("duration", fmt.Sprintf("%v", reservation.Duration)).
				Str("tag", reservation.Tag.String()).
				Bool("to-delete", reservation.ToDelete).
				Bool("expired", expired).
				Logger()

			if expired || reservation.ToDelete {
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

			if err := e.updateStats(); err != nil {
				log.Error().Err(err).Msg("failed to updated the capacity counters")
			}

		}
	}
}

func (e *Engine) provision(ctx context.Context, r *Reservation) error {
	if err := r.validate(); err != nil {
		return errors.Wrapf(err, "failed validation of reservation")
	}

	fn, ok := e.provisioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	_, err := e.cache.Get(r.ID)
	if err == nil {
		log.Info().Str("id", r.ID).Msg("reservation already deployed")
		return nil
	}

	result, err := fn(ctx, r)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", r.ID).
			Msgf("failed to apply provision")
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", result)).
			Msgf("workload deployed")
	}

	if replyErr := e.reply(ctx, r, err, result); replyErr != nil {
		log.Error().Err(replyErr).Msg("failed to send result to BCDB")
	}

	if err != nil {
		return err
	}

	if err := e.cache.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	}

	if err := e.statser.Increment(r); err != nil {
		log.Err(err).Str("reservation_id", r.ID).Msg("failed to increment workloads statistics")
	}

	return nil
}

func (e *Engine) decommission(ctx context.Context, r *Reservation) error {
	fn, ok := e.decomissioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	exists, err := e.cache.Exists(r.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to check if reservation %s exists in cache", r.ID)
	}

	if !exists {
		log.Info().Str("id", r.ID).Msg("reservation not provisioned, no need to decomission")
		if err := e.feedback.Deleted(e.nodeID, r.ID); err != nil {
			log.Error().Err(err).Str("id", r.ID).Msg("failed to mark reservation as deleted")
		}
		return nil
	}

	err = fn(ctx, r)
	if err != nil {
		return errors.Wrap(err, "decommissioning of reservation failed")
	}

	if err := e.cache.Remove(r.ID); err != nil {
		return errors.Wrapf(err, "failed to remove reservation %s from cache", r.ID)
	}

	if err := e.statser.Decrement(r); err != nil {
		log.Err(err).Str("reservation_id", r.ID).Msg("failed to decrement workloads statistics")
	}

	if err := e.feedback.Deleted(e.nodeID, r.ID); err != nil {
		return errors.Wrap(err, "failed to mark reservation as deleted")
	}

	return nil
}

func (e *Engine) reply(ctx context.Context, r *Reservation, err error, info interface{}) error {
	log.Debug().Str("id", r.ID).Msg("sending reply for reservation")

	result := &Result{
		Type:    r.Type,
		Created: time.Now(),
		ID:      r.ID,
	}
	if err != nil {
		result.Error = err.Error()
		result.State = StateError
	} else {
		result.State = StateOk
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

	sig, err := e.signer.Sign(b)
	if err != nil {
		return errors.Wrap(err, "failed to signed the result")
	}
	result.Signature = hex.EncodeToString(sig)

	return e.feedback.Feedback(e.nodeID, result)
}

func (e *Engine) updateStats() error {
	wl := e.statser.CurrentWorkloads()
	r := e.statser.CurrentUnits()

	log.Info().
		Uint16("network", wl.Network).
		Uint16("volume", wl.Volume).
		Uint16("zDBNamespace", wl.ZDBNamespace).
		Uint16("container", wl.Container).
		Uint16("k8sVM", wl.K8sVM).
		Uint16("proxy", wl.Proxy).
		Uint16("reverseProxy", wl.ReverseProxy).
		Uint16("subdomain", wl.Subdomain).
		Uint16("delegateDomain", wl.DelegateDomain).
		Uint64("cru", r.Cru).
		Float64("mru", r.Mru).
		Float64("hru", r.Hru).
		Float64("sru", r.Sru).
		Msgf("provision statistics")

	return e.feedback.UpdateStats(e.nodeID, wl, r)
}

// Counters is a zbus stream that sends statistics from the engine
func (e *Engine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
			}

			wls := e.statser.CurrentWorkloads()
			pc := pkg.ProvisionCounters{
				Container: int64(wls.Container),
				Network:   int64(wls.Network),
				ZDB:       int64(wls.ZDBNamespace),
				Volume:    int64(wls.Volume),
				VM:        int64(wls.K8sVM),
			}

			select {
			case <-ctx.Done():
			case ch <- pc:
			}
		}
	}()

	return ch
}
