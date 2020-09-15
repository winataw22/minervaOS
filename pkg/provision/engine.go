package provision

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jbenet/go-base58"
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

	if r.Reference != "" {
		if err := e.migrateToPool(ctx, r); err != nil {
			return err
		}
	}

	if cached, err := e.cache.Get(r.ID); err == nil {
		log.Info().Str("id", r.ID).Msg("reservation have already been processed")
		if cached.Result.IsNil() {
			// this is probably an older reservation that is cached BEFORE
			// we start caching the result along with the reservation
			// then we just need to return here.
			return nil
		}

		// otherwise, it's safe to resend the same result
		// back to the grid.
		if err := e.reply(ctx, &cached.Result); err != nil {
			log.Error().Err(err).Msg("failed to send result to BCDB")
		}

		return nil
	}

	// to ensure old reservation workload that are already running
	// keeps running as it is, we use the reference as new workload ID
	realID := r.ID
	if r.Reference != "" {
		r.ID = r.Reference
	}

	returned, provisionError := fn(ctx, r)
	if provisionError != nil {
		log.Error().
			Err(provisionError).
			Str("id", r.ID).
			Msgf("failed to apply provision")
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", returned)).
			Msgf("workload deployed")
	}

	result, err := e.buildResult(realID, r.Type, provisionError, returned)
	if err != nil {
		return errors.Wrapf(err, "failed to build result object for reservation: %s", result.ID)
	}

	// we make sure we store the reservation in cache first before
	// returning the reply back to the grid, this is to make sure
	// if the reply failed for any reason, the node still doesn't
	// try to redeploy that reservation.
	r.ID = realID
	r.Result = *result
	if err := e.cache.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	}

	if err := e.reply(ctx, result); err != nil {
		log.Error().Err(err).Msg("failed to send result to BCDB")
	}

	// we skip the counting.
	if provisionError != nil {
		return provisionError
	}

	// If an update occurs on the network we don't increment the counter
	if r.Type == "network_resource" {
		nr := pkg.NetResource{}
		if err := json.Unmarshal(r.Data, &nr); err != nil {
			return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
		}

		uniqueID := NetworkID(r.User, nr.Name)
		exists, err := e.cache.NetworkExists(string(uniqueID))
		if err != nil {
			return errors.Wrap(err, "failed to check if network exists")
		}
		if exists {
			return nil
		}
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

	// to ensure old reservation can be deleted
	// we use the reference as workload ID
	realID := r.ID
	if r.Reference != "" {
		r.ID = r.Reference
	}

	err = fn(ctx, r)
	if err != nil {
		return errors.Wrap(err, "decommissioning of reservation failed")
	}

	r.ID = realID

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

func (e *Engine) reply(ctx context.Context, result *Result) error {
	log.Debug().Str("id", result.ID).Msg("sending reply for reservation")

	if err := e.signResult(result); err != nil {
		return err
	}

	return e.feedback.Feedback(e.nodeID, result)
}

func (e *Engine) buildResult(id string, typ ReservationType, err error, info interface{}) (*Result, error) {
	result := &Result{
		Type:    typ,
		Created: time.Now(),
		ID:      id,
	}

	if err != nil {
		result.Error = err.Error()
		result.State = StateError
	} else {
		result.State = StateOk
	}

	br, err := json.Marshal(info)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}

func (e *Engine) signResult(result *Result) error {

	b, err := result.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to convert the result to byte for signature")
	}

	sig, err := e.signer.Sign(b)
	if err != nil {
		return errors.Wrap(err, "failed to signed the result")
	}
	result.Signature = hex.EncodeToString(sig)

	return nil
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

func (e *Engine) migrateToPool(ctx context.Context, r *Reservation) error {

	oldRes, err := e.cache.Get(r.Reference)
	if err == nil && oldRes.ID == r.Reference {
		// we have received a reservation that reference another one.
		// This is the sign user is trying to migrate his workloads to the new capacity pool system

		log.Info().Str("reference", r.Reference).Msg("reservation referencing another one")

		if string(oldRes.Type) != "network" { //we skip network cause its a PITA
			// first let make sure both are the same
			if !bytes.Equal(oldRes.Data, r.Data) {
				return fmt.Errorf("trying to upgrade workloads to new version. new workload content is different from the old one. upgrade refused")
			}
		}

		// remove the old one from the cache and store the new one
		log.Info().Msgf("migration: remove %v from cache", oldRes.ID)
		if err := e.cache.Remove(oldRes.ID); err != nil {
			return err
		}
		log.Info().Msgf("migration: add %v to cache", r.ID)
		if err := e.cache.Add(r); err != nil {
			return err
		}

		r.Result.ID = r.ID
		if err := e.signResult(&r.Result); err != nil {
			return errors.Wrap(err, "error while signing reservation result")
		}

		if err := e.feedback.Feedback(e.nodeID, &r.Result); err != nil {
			return err
		}

		log.Info().Str("old_id", oldRes.ID).Str("new_id", r.ID).Msg("reservation upgraded to new system")
	}

	return nil
}

// NetworkID construct a network ID based on a userID and network name
func NetworkID(userID, name string) pkg.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(name)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return pkg.NetID(string(b))
}
