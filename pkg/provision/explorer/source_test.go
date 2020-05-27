package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	wrklds "github.com/threefoldtech/tfexplorer/pkg/workloads"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
)

type clientMock struct {
	workloads []workloads.ReservationWorkload
}

func (c *clientMock) Create(reservation workloads.Reservation) (resp wrklds.ReservationCreateResponse, err error) {
	return
}
func (c *clientMock) List(nextAction *workloads.NextActionEnum, customerTid int64, page *client.Pager) (reservation []workloads.Reservation, err error) {
	return
}
func (c *clientMock) Get(id schema.ID) (reservation workloads.Reservation, err error) {
	return
}
func (c *clientMock) SignProvision(id schema.ID, user schema.ID, signature string) error {
	return nil
}
func (c *clientMock) SignDelete(id schema.ID, user schema.ID, signature string) error {
	return nil
}

func (c *clientMock) Workloads(nodeID string, from uint64) ([]workloads.ReservationWorkload, uint64, error) {
	return c.workloads, 0, nil
}
func (c *clientMock) WorkloadGet(gwid string) (result workloads.ReservationWorkload, err error) {
	return
}
func (c *clientMock) WorkloadPutResult(nodeID, gwid string, result workloads.Result) error {
	return nil
}
func (c *clientMock) WorkloadPutDeleted(nodeID, gwid string) error {
	return nil
}

func TestSkipUnsupportedType(t *testing.T) {
	type UnsupportedWorkload struct{}

	client := &clientMock{
		workloads: []workloads.ReservationWorkload{
			{
				Content: workloads.Container{},
			},
			{
				Content: UnsupportedWorkload{},
			},
		},
	}

	p := &Poller{
		wl:        client,
		inputConv: primitives.WorkloadToProvisionType,
	}

	result, _, err := p.Poll(pkg.StrIdentifier(""), 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
}
