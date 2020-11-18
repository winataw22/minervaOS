package provision

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"

	"github.com/robfig/cron/v3"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const gib = 1024 * 1024 * 1024

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
	zbusCl         zbus.Client
	janitor        *Janitor
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
	// ZbusCl is a client to Zbus
	ZbusCl zbus.Client

	// Janitor is used to clean up some of the resources that might be lingering on the node
	// if not set, no cleaning up will be done
	Janitor *Janitor
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
		zbusCl:         opts.ZbusCl,
		janitor:        opts.Janitor,
	}
}

// Run starts reader reservation from the Source and handle them
func (e *Engine) Run(ctx context.Context) error {
	cReservation := e.source.Reservations(ctx)

	isAllWorkloadsProcessed := false
	// run a cron task that will fire the cleanup at midnight
	cleanUp := make(chan struct{}, 2)
	c := cron.New()
	_, err := c.AddFunc("@midnight", func() {
		cleanUp <- struct{}{}
	})
	if err != nil {
		return fmt.Errorf("failed to setup cron task: %w", err)
	}

	c.Start()
	defer c.Stop()

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

			if reservation.last {
				isAllWorkloadsProcessed = true
				// Trigger cleanup by sending a struct onto the channel
				cleanUp <- struct{}{}
				continue
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
				if err := e.decommission(ctx, &reservation.Reservation); err != nil {
					log.Error().Err(err).Msgf("failed to decommission reservation %s", reservation.ID)
					continue
				}
			} else {
				slog.Info().Msg("start provisioning reservation")
				if err := e.provision(ctx, &reservation.Reservation); err != nil {
					log.Error().Err(err).Msgf("failed to provision reservation %s", reservation.ID)
					continue
				}
			}

			if err := e.updateStats(); err != nil {
				log.Error().Err(err).Msg("failed to updated the capacity counters")
			}

		case <-cleanUp:
			if !isAllWorkloadsProcessed {
				// only allow cleanup triggered by the cron to run once
				// we are sure all the workloads from the cache/explorer have been processed
				log.Info().Msg("all workloads not yet processed, delay cleanup")
				continue
			}
			log.Info().Msg("start cleaning up resources")
			if e.janitor == nil {
				log.Info().Msg("janitor is not configured, skipping clean up")
				continue
			}

			if err := e.janitor.CleanupResources(ctx); err != nil {
				log.Error().Err(err).Msg("failed to cleanup resources")
				continue
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

	// send response to explorer
	if err := e.reply(ctx, result); err != nil {
		log.Error().Err(err).Msg("failed to send result to BCDB")
	}

	// if we fail to decomission the reservation then must be marked
	// as deleted so it's never tried again. we also skip caching
	// the reservation object. this is similar to what decomission does
	// since on a decomission we also clear up the cache.
	if provisionError != nil {
		// we need to mark the reservation as deleted as well
		if err := e.feedback.Deleted(e.nodeID, realID); err != nil {
			log.Error().Err(err).Msg("failed to mark failed reservation as deleted")
		}

		return provisionError
	}

	// we only cache successful reservations
	r.ID = realID
	r.Result = *result
	if err := e.cache.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
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

	if r.Result.State == StateError {
		// this reservation already failed to deploy
		// this code here shouldn't be executing because if
		// the reservation has error-ed, it means is should
		// not be in cache.
		// BUT
		// that was not always the case, so instead we
		// will just return. here
		log.Warn().Str("id", realID).Msg("skipping reservation because it is not provisioned")
		return nil
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

// DecommissionCached is used by other module to ask provisiond that
// a certain reservation is dead beyond repair and owner must be informed
// the decommission method will take care to update the reservation instance
// and also decommission the reservation normally
func (e *Engine) DecommissionCached(id string, reason string) error {
	r, err := e.cache.Get(id)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, err := e.buildResult(id, r.Type, fmt.Errorf(reason), nil)
	if err != nil {
		return errors.Wrapf(err, "failed to build result object for reservation: %s", id)
	}

	if err := e.decommission(ctx, r); err != nil {
		log.Error().Err(err).Msgf("failed to update reservation result with failure: %s", id)
	}

	bf := backoff.NewExponentialBackOff()
	bf.MaxInterval = 10 * time.Second
	bf.MaxElapsedTime = 1 * time.Minute

	return backoff.Retry(func() error {
		err := e.reply(ctx, result)
		if err != nil {
			log.Error().Err(err).Msgf("failed to update reservation result with failure: %s", id)
		}
		return err
	}, bf)
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

	if e.zbusCl != nil {
		// TODO: this is a very specific zos code that should not be
		// here. this is a quick fix for the tfgateways
		// but should be implemented cleanely after
		storaged := stubs.NewStorageModuleStub(e.zbusCl)

		cache, err := storaged.GetCacheFS()
		if err != nil {
			return err
		}

		switch cache.DiskType {
		case pkg.SSDDevice:
			r.Sru += float64(cache.Usage.Size / gib)
		case pkg.HDDDevice:
			r.Hru += float64(cache.Usage.Size / gib)
		}

	}

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
