package types

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
)

// Pipeline changes Reservation R as defined by the reservation pipeline
// returns new reservation object, and true if the reservation has changed
type Pipeline struct {
	r Reservation
}

// NewPipeline creates a reservation pipeline, all reservation must be processes
// through the pipeline before any action is taken. This will always make sure
// that reservation is in the right state.
func NewPipeline(R Reservation) (*Pipeline, error) {
	if err := R.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid reservation object")
	}

	return &Pipeline{R}, nil
}

func (p *Pipeline) checkProvisionSignatures() bool {

	// Note: signatures validatation already done in the
	// signature add operation. Here we just make sure the
	// required quorum has been reached

	request := p.r.DataReservation.SigningRequestProvision
	log.Debug().Msgf("%+v", request)
	if request.QuorumMin == 0 {
		return true
	}

	in := func(i int64, l []int64) bool {
		for _, x := range l {
			if x == i {
				return true
			}
		}
		return false
	}

	signatures := p.r.SignaturesProvision
	var count int64
	for _, signature := range signatures {
		if !in(signature.Tid, request.Signers) {
			continue
		}
		count++
	}

	return count >= request.QuorumMin
}

func (p *Pipeline) checkDeleteSignatures() bool {

	// Note: signatures validatation already done in the
	// signature add operation. Here we just make sure the
	// required quorum has been reached
	request := p.r.DataReservation.SigningRequestDelete
	if request.QuorumMin == 0 {
		// if min quorum is zero, then there is no way
		// you can trigger deleting of this reservation
		return false
	}

	in := func(i int64, l []int64) bool {
		for _, x := range l {
			if x == i {
				return true
			}
		}
		return false
	}

	signatures := p.r.SignaturesDelete
	var count int64
	for _, signature := range signatures {
		if !in(signature.Tid, request.Signers) {
			continue
		}
		count++
	}

	return count >= request.QuorumMin
}

// Next gets new modified reservation, and true if the reservation has changed from the input
func (p *Pipeline) Next() (Reservation, bool) {
	if p.r.NextAction == generated.NextActionDelete ||
		p.r.NextAction == generated.NextActionDeleted {
		return p.r, false
	}

	// reseration expiration time must be checked, once expiration time is exceeded
	// the reservation must be deleted
	if p.r.Expired() || p.checkDeleteSignatures() {
		// reservation has expired
		// set its status (next action) to delete
		log.Debug().Int64("id", int64(p.r.ID)).Msg("expired or to be deleted")
		p.r.NextAction = generated.NextActionDelete
		return p.r, true
	}

	current := p.r.NextAction
	modified := false
	for {
		switch p.r.NextAction {
		case generated.NextActionCreate:
			// provision expiration, if exceeded, the node should not
			// try to deploy this reservation.
			if time.Until(p.r.DataReservation.ExpirationProvisioning.Time) <= 0 {
				// exceeded
				// TODO: I think this should be set to "delete" not "invalid"
				log.Debug().Int64("id", int64(p.r.ID)).Msg("expired")
				p.r.NextAction = generated.NextActionInvalid
			} else {
				log.Debug().Int64("id", int64(p.r.ID)).Msg("ready to sign")
				p.r.NextAction = generated.NextActionSign
			}
		case generated.NextActionSign:
			// this stage will not change unless all
			if p.checkProvisionSignatures() {
				log.Debug().Int64("id", int64(p.r.ID)).Msg("ready to pay")
				p.r.NextAction = generated.NextActionPay
			}
		case generated.NextActionPay:
			// TODO: here we should actually start the payment process
			// but this is not implemented yet, so now we just need to move
			// to deploy
			log.Debug().Int64("id", int64(p.r.ID)).Msg("ready to deploy")
			p.r.NextAction = generated.NextActionDeploy
		case generated.NextActionDeploy:
			//nothing to do
			log.Debug().Int64("id", int64(p.r.ID)).Msg("let's deploy")
		}

		if current == p.r.NextAction {
			// no more changes in stage
			break
		}

		current = p.r.NextAction
		modified = true
	}

	return p.r, modified
}
