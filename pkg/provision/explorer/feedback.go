package explorer

import (
	"fmt"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Feedback is an implementation of the provision.Feedbacker
// that u sends results to the TFExplorer: https://github.com/threefoldtech/tfexplorer
type Feedback struct {
	client    *client.Client
	converter provision.ResultConverterFunc
}

// NewFeedback creates an ExplorerFeedback
func NewFeedback(client *client.Client, converter provision.ResultConverterFunc) *Feedback {
	return &Feedback{
		client:    client,
		converter: converter,
	}
}

// Feedback implements provision.Feedbacker
func (e *Feedback) Feedback(nodeID string, r *provision.Result) error {
	wr, err := e.converter(*r)
	if err != nil {
		return fmt.Errorf("failed to convert result into schema type: %w", err)
	}

	return e.client.Workloads.WorkloadPutResult(nodeID, r.ID, *wr)
}

// Deleted implements provision.Feedbacker
func (e *Feedback) Deleted(nodeID, id string) error {
	return e.client.Workloads.WorkloadPutDeleted(nodeID, id)
}

// UpdateStats implements provision.Feedbacker
func (e *Feedback) UpdateStats(nodeID string, w directory.WorkloadAmount, u directory.ResourceAmount) error {
	return e.client.Directory.NodeUpdateUsedResources(nodeID, u, w)
}
