package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/container/stats"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/pkg/versioned"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// ReservationType type
type ReservationType string

const (
	// ContainerReservation type
	ContainerReservation ReservationType = "container"
	// VolumeReservation type
	VolumeReservation ReservationType = "volume"
	// NetworkReservation type
	NetworkReservation ReservationType = "network"
	// ZDBReservation type
	ZDBReservation ReservationType = "zdb"
	// DebugReservation type
	DebugReservation ReservationType = "debug"
	// KubernetesReservation type
	KubernetesReservation ReservationType = "kubernetes"
)

var (
	// reservationSchemaV1 reservation schema version 1
	reservationSchemaV1 = versioned.MustParse("1.0.0")
	// reservationSchemaLastVersion link to latest version
	reservationSchemaLastVersion = reservationSchemaV1
)

// Reservation struct
type Reservation struct {
	// ID of the reservation
	ID string `json:"id"`
	// NodeID of the node where to deploy this reservation
	NodeID string `json:"node_id"`
	// Identification of the user requesting the reservation
	User string `json:"user_id"`
	// Type of the reservation (container, zdb, vm, etc...)
	Type ReservationType `json:"type"`
	// Data is the reservation type arguments.
	Data json.RawMessage `json:"data,omitempty"`
	// Date of creation
	Created time.Time `json:"created"`
	// Duration of the reservation
	Duration time.Duration `json:"duration"`
	// Signature is the signature to the reservation
	// it contains all the field of this struct except the signature itself and the Result field
	Signature []byte `json:"signature,omitempty"`

	// This flag is set to true when a reservation needs to be deleted
	// before its expiration time
	ToDelete bool `json:"to_delete"`

	// Tag object is mainly used for debugging.
	Tag Tag `json:"-"`
}

// AppendTag appends tags
func AppendTag(t, n Tag) Tag {
	if t == nil {
		t = Tag{}
	}

	for k, v := range n {
		t[k] = v
	}

	return t
}

// Tag is custom tag to mark certain reservations
type Tag map[string]string

func (t Tag) String() string {
	var builder strings.Builder
	for k, v := range t {
		if builder.Len() != 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(k)
		builder.WriteString(": ")
		builder.WriteString(v)
	}

	return builder.String()
}

//SplitID gets the reservation part and the workload part from a full reservation ID
func (r *Reservation) SplitID() (reservation uint64, workload uint64, err error) {
	parts := strings.SplitN(r.ID, "-", 2)
	if len(parts) != 2 {
		return reservation, workload, fmt.Errorf("invalid reservation id format (wront length)")
	}
	reservation, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return reservation, workload, errors.Wrap(err, "invalid reservation id format (reservation part)")
	}
	workload, err = strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return reservation, workload, errors.Wrap(err, "invalid reservation id format (workload part)")
	}

	return
}

// Expired returns a boolean depending if the reservation
// has expire or not at the time of the function call
func (r *Reservation) Expired() bool {
	expire := r.Created.Add(r.Duration)
	return time.Now().After(expire)
}

func (r *Reservation) validate() error {
	// TODO: during testnet phase seems we don't need to verify this
	// if err := Verify(r); err != nil {
	// 	log.Warn().
	// 		Err(err).
	// 		Str("id", string(r.ID)).
	// 		Msg("verification of reservation signature failed")
	// 	return errors.Wrapf(err, "verification of reservation %s signature failed", r.ID)
	// }

	if r.Duration <= 0 {
		return fmt.Errorf("reservation %s has not duration", r.ID)
	}

	if r.Created.IsZero() {
		return fmt.Errorf("wrong creation date in reservation %s", r.ID)
	}

	if r.Expired() {
		return fmt.Errorf("reservation %s has expired", r.ID)
	}

	return nil
}

// ToSchemaType creates a TfgridReservation1 from zos provision types
func (r *Reservation) ToSchemaType() (res workloads.Reservation, err error) {

	w, err := workloadFromRaw(r.Data, r.Type)
	if err != nil {
		return res, err
	}

	switch r.Type {
	case ContainerReservation:
		res.DataReservation.Containers = []workloads.Container{
			containerReservation(w, r.NodeID),
		}
	case VolumeReservation:
		res.DataReservation.Volumes = []workloads.Volume{
			volumeReservation(w, r.NodeID),
		}
	case ZDBReservation:
		res.DataReservation.Zdbs = []workloads.ZDB{
			zdbReservation(w, r.NodeID),
		}
	case NetworkReservation:
		res.DataReservation.Networks = []workloads.Network{
			networkReservation(w),
		}
	case KubernetesReservation:
		res.DataReservation.Kubernetes = []workloads.K8S{
			k8sReservation(w, r.NodeID),
		}
	}

	res.Epoch = schema.Date{Time: r.Created}
	res.DataReservation.ExpirationReservation = schema.Date{Time: r.Created.Add(r.Duration)}
	res.DataReservation.ExpirationProvisioning = schema.Date{Time: r.Created.Add(2 * time.Minute)}

	return res, nil
}

func workloadFromRaw(s json.RawMessage, t ReservationType) (interface{}, error) {
	switch t {
	case ContainerReservation:
		c := Container{}
		err := json.Unmarshal([]byte(s), &c)
		return c, err

	case VolumeReservation:
		v := Volume{}
		err := json.Unmarshal([]byte(s), &v)
		return v, err

	case NetworkReservation:
		n := pkg.Network{}
		err := json.Unmarshal([]byte(s), &n)
		return n, err

	case ZDBReservation:
		z := ZDB{}
		err := json.Unmarshal([]byte(s), &z)
		return z, err

	case KubernetesReservation:
		k := Kubernetes{}
		err := json.Unmarshal([]byte(s), &k)
		return k, err
	}

	return nil, fmt.Errorf("unsupported reservation type %v", t)
}

func networkReservation(i interface{}) workloads.Network {
	n := i.(pkg.Network)
	network := workloads.Network{
		Name:             n.Name,
		Iprange:          n.IPRange.ToSchema(),
		WorkloadId:       1,
		NetworkResources: make([]workloads.NetworkNetResource, len(n.NetResources)),
	}

	for i, nr := range n.NetResources {
		network.NetworkResources[i] = workloads.NetworkNetResource{
			NodeId:                       nr.NodeID,
			Iprange:                      nr.Subnet.ToSchema(),
			WireguardPrivateKeyEncrypted: nr.WGPrivateKey,
			WireguardPublicKey:           nr.WGPublicKey,
			WireguardListenPort:          int64(nr.WGListenPort),
			Peers:                        make([]workloads.WireguardPeer, len(nr.Peers)),
		}

		for y, peer := range nr.Peers {
			network.NetworkResources[i].Peers[y] = workloads.WireguardPeer{
				Iprange:        peer.Subnet.ToSchema(),
				Endpoint:       peer.Endpoint,
				PublicKey:      peer.WGPublicKey,
				AllowedIprange: make([]schema.IPRange, len(peer.AllowedIPs)),
			}

			for z, ip := range peer.AllowedIPs {
				network.NetworkResources[i].Peers[y].AllowedIprange[z] = ip.ToSchema()
			}
		}
	}
	return network
}

func containerReservation(i interface{}, nodeID string) workloads.Container {

	c := i.(Container)
	container := workloads.Container{
		NodeId:            nodeID,
		WorkloadId:        1,
		Flist:             c.FList,
		HubUrl:            c.FlistStorage,
		Environment:       c.Env,
		SecretEnvironment: c.SecretEnv,
		Entrypoint:        c.Entrypoint,
		Interactive:       c.Interactive,
		Volumes:           make([]workloads.ContainerMount, len(c.Mounts)),
		StatsAggregator:   make([]workloads.StatsAggregator, len(c.StatsAggregator)),
		Logs:              make([]workloads.Logs, len(c.Logs)),
		NetworkConnection: []workloads.NetworkConnection{
			{
				NetworkId: string(c.Network.NetworkID),
				Ipaddress: c.Network.IPs[0],
				PublicIp6: c.Network.PublicIP6,
			},
		},
		Capacity: workloads.ContainerCapacity{
			Cpu:    int64(c.Capacity.CPU),
			Memory: int64(c.Capacity.Memory),
		},
	}

	for i, v := range c.Mounts {
		container.Volumes[i] = workloads.ContainerMount{
			VolumeId:   v.VolumeID,
			Mountpoint: v.Mountpoint,
		}
	}

	for i, l := range c.Logs {
		if l.Type != logger.RedisType {
			container.Logs[i] = workloads.Logs{
				Type: "unknown",
				Data: workloads.LogsRedis{},
			}

			continue
		}

		container.Logs[i] = workloads.Logs{
			Type: l.Type,
			Data: workloads.LogsRedis{
				Stdout: l.Data.Stdout,
				Stderr: l.Data.Stderr,
			},
		}
	}

	for i, s := range c.StatsAggregator {
		if s.Type != stats.RedisType {
			container.StatsAggregator[i] = workloads.StatsAggregator{
				Type: "unknown",
				Data: workloads.StatsRedis{},
			}

			continue
		}

		container.StatsAggregator[i] = workloads.StatsAggregator{
			Type: s.Type,
			Data: workloads.StatsRedis{
				Endpoint: s.Data.Endpoint,
			},
		}
	}

	return container
}

func volumeReservation(i interface{}, nodeID string) workloads.Volume {
	v := i.(Volume)

	volume := workloads.Volume{
		NodeId:     nodeID,
		WorkloadId: 1,
		Size:       int64(v.Size),
	}

	if v.Type == HDDDiskType {
		volume.Type = workloads.VolumeTypeHDD
	} else if v.Type == SSDDiskType {
		volume.Type = workloads.VolumeTypeSSD
	}

	return volume
}

func zdbReservation(i interface{}, nodeID string) workloads.ZDB {
	z := i.(ZDB)

	zdb := workloads.ZDB{
		WorkloadId: 1,
		NodeId:     nodeID,
		// ReservationID:
		Size:     int64(z.Size),
		Password: z.Password,
		Public:   z.Public,
		// StatsAggregator:
		// FarmerTid:
	}
	if z.DiskType == pkg.SSDDevice {
		zdb.DiskType = workloads.DiskTypeHDD
	} else if z.DiskType == pkg.HDDDevice {
		zdb.DiskType = workloads.DiskTypeSSD
	}

	if z.Mode == pkg.ZDBModeUser {
		zdb.Mode = workloads.ZDBModeUser
	} else if z.Mode == pkg.ZDBModeSeq {
		zdb.Mode = workloads.ZDBModeSeq
	}

	return zdb
}

func k8sReservation(i interface{}, nodeID string) workloads.K8S {
	k := i.(Kubernetes)

	k8s := workloads.K8S{
		WorkloadId:    1,
		NodeId:        nodeID,
		Size:          int64(k.Size),
		NetworkId:     string(k.NetworkID),
		Ipaddress:     k.IP,
		ClusterSecret: k.ClusterSecret,
		MasterIps:     k.MasterIPs,
		SshKeys:       k.SSHKeys,
	}

	return k8s
}

// ResultState type
type ResultState workloads.ResultStateEnum

const (
	// StateError constant
	StateError = ResultState(workloads.ResultStateError)
	// StateOk constant
	StateOk = ResultState(workloads.ResultStateOK)
	//StateDeleted constant
	StateDeleted = ResultState(workloads.ResultStateDeleted)
)

func (s ResultState) String() string {
	return workloads.ResultStateEnum(s).String()
}

// Result is the struct filled by the node
// after a reservation object has been processed
type Result struct {
	Type ReservationType `json:"type"`
	//Reservation ID
	ID string `json:"id"`
	// Time when the result is sent
	Created time.Time `json:"created"`
	// State of the deployment (ok,error)
	State ResultState `json:"state"`
	// if State is "error", then this field contains the error
	// otherwise it's nil
	Error string `json:"message"`
	// Data is the information generated by the provisioning of the workload
	// its type depend on the reservation type
	Data json.RawMessage `json:"data_json"`
	// Signature is the signature to the result
	// is generated by signing the bytes returned from call to Result.Bytes()
	// and hex
	Signature string `json:"signature"`
}

// Bytes returns a slice of bytes container all the information
// used to sign the Result object
func (r *Result) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := buf.WriteByte(byte(r.State)); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(r.Error); err != nil {
		return nil, err
	}
	if _, err := buf.Write(r.Data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ToSchemaType converts result to schema type
func (r *Result) ToSchemaType() workloads.Result {
	var rType workloads.ResultCategoryEnum
	switch r.Type {
	case VolumeReservation:
		rType = workloads.ResultCategoryVolume
	case ContainerReservation:
		rType = workloads.ResultCategoryContainer
	case ZDBReservation:
		rType = workloads.ResultCategoryZDB
	case NetworkReservation:
		rType = workloads.ResultCategoryNetwork
	case KubernetesReservation:
		rType = workloads.ResultCategoryK8S
	default:
		panic(fmt.Errorf("unknown reservation type: %s", r.Type))
	}

	result := workloads.Result{
		Category:   rType,
		WorkloadId: r.ID,
		DataJson:   r.Data,
		Signature:  r.Signature,
		State:      workloads.ResultStateEnum(r.State),
		Message:    r.Error,
		Epoch:      schema.Date{Time: r.Created},
	}

	return result
}
