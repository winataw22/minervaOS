package provision

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"

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
	Counters() Counters
}

// Feedbacker defines the method that needs to be implemented
// to send the provision result to BCDB
type Feedbacker interface {
	Feedback(nodeID string, r *Result) error
	Deleted(nodeID, id string) error
	UpdateReservedResources(nodeID string, c Counters) error
}

type defaultEngine struct {
	nodeID string
	source ReservationSource
	store  ReservationCache
	cl     *client.Client
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(nodeID string, source ReservationSource, rw ReservationCache, cl *client.Client) Engine {
	return &defaultEngine{
		nodeID: nodeID,
		source: source,
		store:  rw,
		cl:     cl,
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

			if err := e.updateReservedCapacity(); err != nil {
				log.Error().Err(err).Msg("failed to updated the used resources")
			}
		}
	}
}

func (e defaultEngine) capacityUsed() (directory.ResourceAmount, directory.WorkloadAmount) {
	counters := e.store.Counters()

	resources := directory.ResourceAmount{
		Cru: counters.CRU.Current(),
		Mru: float64(counters.MRU.Current()) / float64(gib),
		Sru: float64(counters.SRU.Current()) / float64(gib),
		Hru: float64(counters.HRU.Current()) / float64(gib),
	}

	workloads := directory.WorkloadAmount{
		Volume:       uint16(counters.volumes.Current()),
		Container:    uint16(counters.containers.Current()),
		ZDBNamespace: uint16(counters.zdbs.Current()),
		K8sVM:        uint16(counters.vms.Current()),
		Network:      uint16(counters.networks.Current()),
	}
	return resources, workloads
}

func (e *defaultEngine) updateReservedCapacity() error {
	resources, workloads := e.capacityUsed()
	log.Info().Msgf("reserved resource %+v", resources)
	log.Info().Msgf("provisionned workloads %+v", workloads)

	return e.cl.Directory.NodeUpdateUsedResources(e.nodeID, resources, workloads)
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
		if err := e.cl.Workloads.WorkloadPutDeleted(e.nodeID, r.ID); err != nil {
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

	if err := e.cl.Workloads.WorkloadPutDeleted(e.nodeID, r.ID); err != nil {
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
		result.State = StateError
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", info)).
			Msgf("workload deployed")
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

	sig, err := identity.Sign(b)
	if err != nil {
		return errors.Wrap(err, "failed to signed the result")
	}
	result.Signature = hex.EncodeToString(sig)

	return e.cl.Workloads.WorkloadPutResult(e.nodeID, r.ID, result.ToSchemaType())
}

func (e *defaultEngine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
			}

			c := e.store.Counters()
			pc := pkg.ProvisionCounters{
				Container: int64(c.containers.Current()),
				Network:   int64(c.networks.Current()),
				ZDB:       int64(c.zdbs.Current()),
				Volume:    int64(c.volumes.Current()),
				VM:        int64(c.vms.Current()),
			}

			select {
			case <-ctx.Done():
			case ch <- pc:
			}
		}
	}()

	return ch
}
