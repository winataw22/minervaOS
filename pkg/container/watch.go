package container

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/events"
	"github.com/containerd/typeurl"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (c *Module) handlerEventTaskExit(ns string, event *events.TaskExit) {
	log := log.With().
		Str("namespace", ns).
		Str("container", event.ContainerID).Logger()

	log.Debug().Msg("task exited")

	marker, ok := c.failures.Get(event.ContainerID)
	if !ok {
		// no previous value. so this is the first failure
		c.failures.Set(event.ContainerID, int(0), cache.DefaultExpiration)
	}

	if marker == permanent {
		// if the marker is permanent. it means that this container
		// is being deleted. we don't need to take any more action here
		// (don't try to restart or delete)
		log.Debug().Msg("permanent delete marker is set")
		return
	}

	count, err := c.failures.IncrementInt(event.ContainerID, 1)
	if err != nil {
		// this should never happen because we make sure value
		// is set
		log.Error().Err(err).Msg("error while checking number of failures")
		return
	}

	log.Debug().Int("count", count).Msg("recorded stops")

	var reason error
	if count < failuresBeforeDestroy {
		log.Debug().Msg("trying to restart the container")
		<-time.After(restartDelay) // wait for 2 seconds
		reason = c.start(ns, event.ContainerID)
	} else {
		reason = fmt.Errorf("deleting container due to so many crashes")
	}

	if reason != nil {
		log.Debug().Err(reason).Msg("deleting container due to restart error")

		stub := stubs.NewProvisionStub(c.client)
		if err := stub.DecommissionCached(event.ContainerID, reason.Error()); err != nil {
			log.Error().Err(err).Msg("failed to decommission reservation")
		}
	}
}

func (c *Module) handleEvent(ns string, event interface{}) {
	switch event := event.(type) {
	case *events.TaskExit:
		// we run this handler in a go routine because
		// - we don't want the restarts to slow down the event stream processing
		// - this method does not return any useful value anyway, so safe to run
		//   it in the background.
		go c.handlerEventTaskExit(ns, event)
	default:
		log.Debug().Msgf("unhandled event: %+v", event)
	}
}

// watch method will start a connection, and register for events
// once an event is received, it will be handled. and exit on
// first error or in case context was cancelled.
// the caller must make sure this is called again in case of an
// error
func (c *Module) watch(ctx context.Context) error {
	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}

	defer client.Close()
	log.Debug().Str("containerd", c.containerd).Msg("subscribe to events from containerd")

	source, errors := client.Subscribe(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case envelope := <-source:
			event, err := typeurl.UnmarshalAny(envelope.Event)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal event envelope")
				continue
			}

			c.handleEvent(envelope.Namespace, event)
		case err := <-errors:
			return err
		}
	}
}

// Watch start watching for events coming from containerd.
// Blocks forever. caller need to run this in a go routine
//
// different events types are handled differently. Now, only
// TaskExit event is handled.
func (c *Module) Watch(ctx context.Context) {
	for {
		err := c.watch(ctx)
		if err == nil {
			break // end of context
		}

		log.Err(err).Msg("error while watching events from containerd")
	}
}
