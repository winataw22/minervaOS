package provision

import (
	"encoding/json"
	"fmt"
	"math"
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
	SRU int64 `json:"sru,omitempty"`
	HRU int64 `json:"hru,omitempty"`
	MRU int64 `json:"mru,omitempty"`
	CRU int64 `json:"cru,omitempty"`
}

func processVolume(r *Reservation) (u resourceUnits, err error) {
	var volume Volume
	if err = json.Unmarshal(r.Data, &volume); err != nil {
		return u, err
	}

	// volume.size and SRU is in GiB, not conversion needed
	switch volume.Type {
	case SSDDiskType:
		u.SRU = int64(volume.Size)
	case HDDDiskType:
		u.HRU = int64(volume.Size)
	}

	return u, nil
}

func processContainer(r *Reservation) (u resourceUnits, err error) {
	var cont Container
	if err = json.Unmarshal(r.Data, &cont); err != nil {
		return u, err
	}
	u.CRU = int64(cont.Capacity.CPU)
	// memory is in MiB, but MRU is in GiB
	u.MRU = int64(math.Ceil(float64(cont.Capacity.Memory) / 1024.0))

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
		u.SRU = int64(zdbVolume.Size)
	case "HDD":
		u.HRU = int64(zdbVolume.Size)
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
		u.MRU = 2
		u.SRU = 50
	case 2:
		u.CRU = 2
		u.MRU = 4
		u.SRU = 100
	}

	return u, nil
}
