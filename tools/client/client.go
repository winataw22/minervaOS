package client

import (
	"fmt"
	"net/url"

	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/phonebook"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// Client structure
type Client struct {
	Phonebook Phonebook
	Directory Directory
	Workloads Workloads
}

// Directory API interface
type Directory interface {
	FarmRegister(farm directory.Farm) (schema.ID, error)
	FarmList(tid schema.ID, name string, page *Pager) (farms []directory.Farm, err error)
	FarmGet(id schema.ID) (farm directory.Farm, err error)

	NodeRegister(node directory.Node) error
	NodeList(filter NodeFilter) (nodes []directory.Node, err error)
	NodeGet(id string, proofs bool) (node directory.Node, err error)

	NodeSetInterfaces(id string, ifaces []directory.Iface) error
	NodeSetPorts(id string, ports []uint) error
	NodeSetPublic(id string, pub directory.PublicIface) error

	//TODO: this method call uses types from zos that is not generated
	//from the schema. Which is wrong imho.
	NodeSetCapacity(
		id string,
		resources directory.ResourceAmount,
		dmiInfo dmi.DMI,
		disksInfo capacity.Disks,
		hypervisor []string,
	) error

	NodeUpdateUptime(id string, uptime uint64) error
	NodeUpdateUsedResources(id string, amount directory.ResourceAmount) error
}

// Phonebook interface
type Phonebook interface {
	Create(user phonebook.User) (schema.ID, error)
	List(name, email string, page *Pager) (output []phonebook.User, err error)
	Get(id schema.ID) (phonebook.User, error)
	// Update() #TODO
	Validate(id schema.ID, message, signature string) (bool, error)
}

// Workloads interface
type Workloads interface {
	Create(reservation workloads.Reservation) (id schema.ID, err error)
	List(nextAction *workloads.NextActionEnum, customerTid int64, page *Pager) (reservation []workloads.Reservation, err error)
	Get(id schema.ID) (reservation workloads.Reservation, err error)

	SignProvision(id schema.ID, user schema.ID, signature string) error
	SignDelete(id schema.ID, user schema.ID, signature string) error

	Workloads(nodeID string, from uint64) ([]workloads.ReservationWorkload, error)
	WorkloadGet(gwid string) (result workloads.ReservationWorkload, err error)
	WorkloadPutResult(nodeID, gwid string, result workloads.Result) error
	WorkloadPutDeleted(nodeID, gwid string) error
}

// Pager for listing
type Pager struct {
	p int
	s int
}

func (p *Pager) apply(v url.Values) {
	if p == nil {
		return
	}

	if p.p < 1 {
		p.p = 1
	}

	if p.s == 0 {
		p.s = 10
	}

	v.Set("page", fmt.Sprint(p.p))
	v.Set("size", fmt.Sprint(p.s))
}

// Page returns a pager
func Page(page, size int) *Pager {
	return &Pager{p: page, s: size}
}

// NewClient creates a new client
func NewClient(u string, kp identity.KeyPair) (*Client, error) {
	h, err := newHTTPClient(u, kp)
	if err != nil {
		return nil, err
	}
	cl := &Client{
		Phonebook: &httpPhonebook{h},
		Directory: &httpDirectory{h},
		Workloads: &httpWorkloads{h},
	}

	return cl, nil
}
