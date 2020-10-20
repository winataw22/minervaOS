package explorer

import (
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Feedback is an implementation of the provision.Feedbacker
// that u sends results to the TFExplorer: https://github.com/threefoldtech/tfexplorer
type Feedback struct {
	client    *client.Client
	converter provision.ResultConverterFunc

	strategy backoff.BackOff
}

// NewFeedback creates an ExplorerFeedback
func NewFeedback(client *client.Client, converter provision.ResultConverterFunc) *Feedback {
	strategy := backoff.NewExponentialBackOff()
	strategy.MaxInterval = 15 * time.Second
	strategy.MaxElapsedTime = 3 * time.Minute

	return &Feedback{
		client:    client,
		converter: converter,
		strategy:  strategy,
	}
}

// Feedback implements provision.Feedbacker
func (e *Feedback) Feedback(nodeID string, r *provision.Result) error {
	wr, err := e.converter(*r)
	if err != nil {
		return fmt.Errorf("failed to convert result into schema type: %w", err)
	}

	return backoff.Retry(func() error {
		err := e.client.Workloads.NodeWorkloadPutResult(nodeID, r.ID, *wr)
		if err == nil || errors.Is(err, client.ErrRequestFailure) {
			// we only retry if err is a request failure err.
			return err
		}

		// otherwise retrying won't fix it, so we can terminate
		return backoff.Permanent(err)
	}, e.strategy)
}

// Deleted implements provision.Feedbacker
func (e *Feedback) Deleted(nodeID, id string) error {
	return backoff.Retry(func() error {
		err := e.client.Workloads.NodeWorkloadPutDeleted(nodeID, id)
		if err == nil || errors.Is(err, client.ErrRequestFailure) {
			// we only retry if err is a request failure err.
			return err
		}

		// otherwise retrying won't fix it, so we can terminate
		return backoff.Permanent(err)
	}, e.strategy)
}

// UpdateStats implements provision.Feedbacker
func (e *Feedback) UpdateStats(nodeID string, w directory.WorkloadAmount, u directory.ResourceAmount) error {
	return backoff.Retry(func() error {
		err := e.client.Directory.NodeUpdateUsedResources(nodeID, u, w)
		if err == nil || errors.Is(err, client.ErrRequestFailure) {
			// we only retry if err is a request failure err.
			return err
		}

		// otherwise retrying won't fix it, so we can terminate
		return backoff.Permanent(err)
	}, e.strategy)
}
