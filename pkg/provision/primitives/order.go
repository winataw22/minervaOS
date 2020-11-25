package primitives

import "github.com/threefoldtech/zos/pkg/provision"

const (
	// ContainerReservation type
	ContainerReservation provision.ReservationType = "container"
	// VolumeReservation type
	VolumeReservation provision.ReservationType = "volume"
	// NetworkReservation type
	NetworkReservation provision.ReservationType = "network"
	// NetworkResourceReservation type
	NetworkResourceReservation provision.ReservationType = "network_resource"
	// ZDBReservation type
	ZDBReservation provision.ReservationType = "zdb"
	// DebugReservation type
	DebugReservation provision.ReservationType = "debug"
	// KubernetesReservation type
	KubernetesReservation provision.ReservationType = "kubernetes"
	// PublicIPReservation type
	PublicIPReservation provision.ReservationType = "public_ip"
)

// ProvisionOrder is used to sort the workload type
// in the right order for provision engine
var ProvisionOrder = map[provision.ReservationType]int{
	DebugReservation:           0,
	NetworkReservation:         1,
	NetworkResourceReservation: 2,
	ZDBReservation:             3,
	VolumeReservation:          4,
	ContainerReservation:       5,
	KubernetesReservation:      6,
	PublicIPReservation:        7,
}
