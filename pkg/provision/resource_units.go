package provision

import (
	"encoding/json"
	"fmt"
)

const (
	mib = uint64(1024 * 1024)
	gib = uint64(mib * 1024)
)

func (s *FSStore) processResourceUnits(r *Reservation, addOrRemoveBool bool) error {

	var (
		u   resourceUnits
		err error
	)

	switch r.Type {
	case VolumeReservation:
		u, err = processVolume(r)
	case ContainerReservation:
		u, err = processContainer(r)
	case ZDBReservation:
		u, err = processZdb(r)
	case KubernetesReservation:
		u, err = processKubernetes(r)
	}
	if err != nil {
		return err
	}

	if addOrRemoveBool {
		s.counters.CRU.Increment(u.CRU)
		s.counters.MRU.Increment(u.MRU)
		s.counters.SRU.Increment(u.SRU)
		s.counters.HRU.Increment(u.HRU)
	} else {
		s.counters.CRU.Decrement(u.CRU)
		s.counters.MRU.Decrement(u.MRU)
		s.counters.SRU.Decrement(u.SRU)
		s.counters.HRU.Decrement(u.HRU)
	}

	return nil
}

type resourceUnits struct {
	SRU uint64 `json:"sru,omitempty"`
	HRU uint64 `json:"hru,omitempty"`
	MRU uint64 `json:"mru,omitempty"`
	CRU uint64 `json:"cru,omitempty"`
}

func processVolume(r *Reservation) (u resourceUnits, err error) {
	var volume Volume
	if err = json.Unmarshal(r.Data, &volume); err != nil {
		return u, err
	}

	// volume.size and SRU is in GiB
	switch volume.Type {
	case SSDDiskType:
		u.SRU = volume.Size * gib
	case HDDDiskType:
		u.HRU = volume.Size * gib
	}

	return u, nil
}

func processContainer(r *Reservation) (u resourceUnits, err error) {
	var cont Container
	if err = json.Unmarshal(r.Data, &cont); err != nil {
		return u, err
	}
	u.CRU = uint64(cont.Capacity.CPU)
	// memory is in MiB
	u.MRU = cont.Capacity.Memory * mib
	u.SRU = 256 * mib // 250MiB are allocated on SSD for the root filesystem used by the flist

	return u, nil
}

func processZdb(r *Reservation) (u resourceUnits, err error) {
	if r.Type != ZDBReservation {
		return u, fmt.Errorf("wrong type or reservation %s, excepted %s", r.Type, ZDBReservation)
	}
	var zdbVolume ZDB
	if err := json.Unmarshal(r.Data, &zdbVolume); err != nil {
		return u, err
	}

	switch zdbVolume.DiskType {
	case "SSD":
		u.SRU = zdbVolume.Size * gib
	case "HDD":
		u.HRU = zdbVolume.Size * gib
	}

	return u, nil
}

func processKubernetes(r *Reservation) (u resourceUnits, err error) {
	var k8s Kubernetes
	if err = json.Unmarshal(r.Data, &k8s); err != nil {
		return u, err
	}

	// size are defined at https://github.com/threefoldtech/zos/blob/master/pkg/provision/kubernetes.go#L311
	switch k8s.Size {
	case 1:
		u.CRU = 1
		u.MRU = 2 * gib
		u.SRU = 50 * gib
	case 2:
		u.CRU = 2
		u.MRU = 4 * gib
		u.SRU = 100 * gib
	}

	return u, nil
}
