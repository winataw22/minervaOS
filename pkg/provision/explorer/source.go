package explorer

import (
	"errors"
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
)

// Poller is an implementation of the provision.ReservationPoller
// that retrieve reservation from the TFExplorer: https://github.com/threefoldtech/tfexplorer
type Poller struct {
	wl             client.Workloads
	inputConv      provision.ReservationConverterFunc
	provisionOrder map[provision.ReservationType]int
}

// NewPoller returns a reservation poller
// inputConv is a function used by the provision.Engine to convert date received from the explorer to into the internal type of the system
// provisionOrder is a map with each primitive type as key. It is used to order the reservation before sending them to the engine.
//  This can be useful if some workloads in a same reservation depends on each other
func NewPoller(cl *client.Client, inputConv provision.ReservationConverterFunc, provisionOrder map[provision.ReservationType]int) *Poller {
	return &Poller{
		wl:             cl.Workloads,
		inputConv:      inputConv,
		provisionOrder: provisionOrder,
	}
}

// Get gets a reservation by the global workload id
func (r *Poller) Get(gwid string) (*provision.Reservation, error) {
	workload, err := r.wl.NodeWorkloadGet(gwid)
	if err != nil {
		return nil, err
	}

	return r.inputConv(workload)
}

// Poll implements provision.ReservationPoller
func (r *Poller) Poll(nodeID pkg.Identifier, from uint64) ([]*provision.Reservation, uint64, error) {

	list, lastID, err := r.wl.NodeWorkloads(nodeID.Identity(), from)
	if err != nil {
		return nil, 0, fmt.Errorf("error while retrieving workloads from explorer: %w", err)
	}

	result := make([]*provision.Reservation, 0, len(list))
	for _, wl := range list {
		r, err := r.inputConv(wl)
		if err != nil {
			if errors.Is(err, primitives.ErrUnsupportedWorkload) {
				log.Warn().Err(err).Msgf("received unsupported workload, skipping")
				continue
			}
			return nil, 0, err
		}

		result = append(result, r)
	}

	if r.provisionOrder != nil {
		// sorts the workloads in the oder they need to be processed by provisiond
		sort.Slice(result, func(i int, j int) bool {
			return r.provisionOrder[result[i].Type] < r.provisionOrder[result[j].Type]
		})
	}

	return result, lastID, nil
}
