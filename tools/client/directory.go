package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
)

type httpDirectory struct {
	*httpClient
}

func (d *httpDirectory) FarmRegister(farm directory.Farm) (schema.ID, error) {
	var output struct {
		ID schema.ID `json:"id"`
	}

	err := d.post(d.url("farms"), farm, &output, http.StatusCreated)
	return output.ID, err
}

func (d *httpDirectory) FarmUpdate(farm directory.Farm) error {
	return d.put(d.url("farms", fmt.Sprintf("%d", farm.ID)), farm, nil, http.StatusOK)
}

func (d *httpDirectory) FarmList(tid schema.ID, name string, page *Pager) (farms []directory.Farm, err error) {
	query := url.Values{}
	page.apply(query)
	if tid > 0 {
		query.Set("owner", fmt.Sprint(tid))
	}
	if len(name) != 0 {
		query.Set("name", name)
	}
	err = d.get(d.url("farms"), query, &farms, http.StatusOK)
	return
}

func (d *httpDirectory) FarmGet(id schema.ID) (farm directory.Farm, err error) {
	err = d.get(d.url("farms", fmt.Sprint(id)), nil, &farm, http.StatusOK)
	return
}

func (d *httpDirectory) NodeRegister(node directory.Node) error {
	return d.post(d.url("nodes"), node, nil, http.StatusCreated)
}

func (d *httpDirectory) NodeList(filter NodeFilter) (nodes []directory.Node, err error) {
	query := url.Values{}
	filter.Apply(query)
	err = d.get(d.url("nodes"), query, &nodes, http.StatusOK)
	return
}

func (d *httpDirectory) NodeGet(id string, proofs bool) (node directory.Node, err error) {
	query := url.Values{}
	query.Set("proofs", fmt.Sprint(proofs))
	err = d.get(d.url("nodes", id), query, &node, http.StatusOK)
	return
}

func (d *httpDirectory) NodeSetInterfaces(id string, ifaces []directory.Iface) error {
	return d.post(d.url("nodes", id, "interfaces"), ifaces, nil, http.StatusCreated)
}

func (d *httpDirectory) NodeSetPorts(id string, ports []uint) error {
	var input struct {
		P []uint `json:"ports"`
	}
	input.P = ports

	return d.post(d.url("nodes", id, "ports"), input, nil, http.StatusOK)
}

func (d *httpDirectory) NodeSetPublic(id string, pub directory.PublicIface) error {
	return d.post(d.url("nodes", id, "configure_public"), pub, nil, http.StatusCreated)
}

func (d *httpDirectory) NodeSetCapacity(
	id string,
	resources directory.ResourceAmount,
	dmiInfo dmi.DMI,
	disksInfo capacity.Disks,
	hypervisor []string) error {

	payload := struct {
		Capacity   directory.ResourceAmount `json:"capacity"`
		DMI        dmi.DMI                  `json:"dmi"`
		Disks      capacity.Disks           `json:"disks"`
		Hypervisor []string                 `json:"hypervisor"`
	}{
		Capacity:   resources,
		DMI:        dmiInfo,
		Disks:      disksInfo,
		Hypervisor: hypervisor,
	}

	return d.post(d.url("nodes", id, "capacity"), payload, nil, http.StatusOK)
}

func (d *httpDirectory) NodeUpdateUptime(id string, uptime uint64) error {
	input := struct {
		U uint64 `json:"uptime"`
	}{
		U: uptime,
	}

	return d.post(d.url("nodes", id, "uptime"), input, nil, http.StatusOK)
}

func (d *httpDirectory) NodeUpdateUsedResources(id string, resources directory.ResourceAmount, workloads directory.WorkloadAmount) error {
	input := struct {
		directory.ResourceAmount
		directory.WorkloadAmount
	}{
		resources,
		workloads,
	}
	return d.post(d.url("nodes", id, "used_resources"), input, nil, http.StatusOK)
}

func (d *httpDirectory) NodeSetFreeToUse(id string, free bool) error {
	choice := struct {
		FreeToUse bool `json:"free_to_use"`
	}{FreeToUse: free}

	return d.post(d.url("nodes", id, "configure_free"), choice, nil, http.StatusOK)
}
