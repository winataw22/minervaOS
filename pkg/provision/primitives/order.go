package primitives

import "github.com/threefoldtech/zos/pkg/provision"

const (
	// ContainerReservation type
	ContainerReservation provision.ReservationType = "container"
	// VolumeReservation type
	VolumeReservation provision.ReservationType = "volume"
	// NetworkReservation type
	NetworkReservation provision.ReservationType = "network"
	// ZDBReservation type
	ZDBReservation provision.ReservationType = "zdb"
	// DebugReservation type
	DebugReservation provision.ReservationType = "debug"
	// KubernetesReservation type
	KubernetesReservation provision.ReservationType = "kubernetes"
)

// ProvisionOrder is used to sort the workload type
// in the right order for provision engine
var ProvisionOrder = map[provision.ReservationType]int{
	DebugReservation:      0,
	NetworkReservation:    1,
	ZDBReservation:        2,
	VolumeReservation:     3,
	ContainerReservation:  4,
	KubernetesReservation: 5,
}
