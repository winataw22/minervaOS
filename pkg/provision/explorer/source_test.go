package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/tfexplorer/pkg/capacity/types"
	wrklds "github.com/threefoldtech/tfexplorer/pkg/workloads"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
)

type clientMock struct {
	workloads []workloads.Workloader
}

func (c *clientMock) Create(reservation workloads.Workloader) (resp wrklds.ReservationCreateResponse, err error) {
	return
}

func (c *clientMock) List(nextAction *workloads.NextActionEnum, customerTid int64, page *client.Pager) (reservation []workloads.Reservation, err error) {
	return
}
func (c *clientMock) Get(id schema.ID) (reservation workloads.Workloader, err error) {
	return
}
func (c *clientMock) SignProvision(id schema.ID, user schema.ID, signature string) error {
	return nil
}
func (c *clientMock) SignDelete(id schema.ID, user schema.ID, signature string) error {
	return nil
}

func (c *clientMock) NodeWorkloads(nodeID string, from uint64) ([]workloads.Workloader, uint64, error) {
	return c.workloads, 0, nil
}
func (c *clientMock) NodeWorkloadGet(gwid string) (result workloads.Workloader, err error) {
	return
}
func (c *clientMock) NodeWorkloadPutResult(nodeID, gwid string, result workloads.Result) error {
	return nil
}
func (c *clientMock) NodeWorkloadPutDeleted(nodeID, gwid string) error {
	return nil
}

func (c *clientMock) PoolCreate(reservation types.Reservation) (resp wrklds.CapacityPoolCreateResponse, err error) {
	return
}
func (c *clientMock) PoolGet(poolID string) (result types.Pool, err error) {
	return
}
func (c *clientMock) PoolsGetByOwner(ownerID string) (result []types.Pool, err error) {
	return
}

func TestSkipUnsupportedType(t *testing.T) {
	type UnsupportedWorkload struct {
		workloads.ReservationInfo
		workloads.Capaciter
	}

	client := &clientMock{
		workloads: []workloads.Workloader{
			&workloads.Container{ReservationInfo: workloads.ReservationInfo{
				WorkloadType: workloads.WorkloadTypeContainer,
			}},
			&UnsupportedWorkload{
				ReservationInfo: workloads.ReservationInfo{
					WorkloadType: workloads.WorkloadTypeEnum(99),
				},
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
