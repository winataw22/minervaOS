package pkg

//go:generate mkdir -p stubs
//go:generate zbusc -module container -version 0.0.1 -name container -package stubs github.com/threefoldtech/zos/pkg+ContainerModule stubs/container_stub.go

import (
	"time"

	"github.com/threefoldtech/zos/pkg/container/logger"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	RootFSPropagationSlave    = "slave"
	RootFSPropagationRslave   = "rslave"
	RootFSPropagationShared   = "shared"
	RootFSPropagationRshared  = "rshared"
	RootFSPropagationPrivate  = "private"
	RootFSPropagationRprivate = "rprivate"
)

// RootFSPropagation defines how the submounts are propagated to its mountpoint peers
type RootFSPropagation string

// ContainerID type
type ContainerID string

// NetworkInfo defines a network configuration for a container
type NetworkInfo struct {
	// Currently a container can only join one (and only one)
	// network namespace that has to be pre defined on the node
	// for the container tenant

	// Containers don't need to know about anything about bridges,
	// IPs, wireguards since this is all is only known by the network
	// resource which is out of the scope of this module
	Namespace string
}

// MountInfo defines a mount point
type MountInfo struct {
	Source string // source of the mount point on the host
	Target string // target of mount inside the container
}

// Stats endpoints
type Stats struct {
	Type     string `bson:"type" json:"type"`
	Endpoint string `bson:"endpoint" json:"endpoint"`
}

//Container creation info
type Container struct {
	// Name of container
	Name string
	// path to the rootfs of the container
	RootFS string
	// Env env variables to container in format {'KEY=VALUE', 'KEY2=VALUE2'}
	Env []string
	// WorkingDir of the entrypoint command
	WorkingDir string
	// Network network info for container
	Network NetworkInfo
	// Mounts extra mounts for container
	Mounts []MountInfo
	// Entrypoint the process to start inside the container
	Entrypoint string
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool
	// CPU count limit
	CPU uint
	// Memory limit in
	Memory gridtypes.Unit
	// Logs backends
	Logs []logger.Logs
	// Stats container metrics backend
	Stats []Stats
	// Elevated privileges (to use fuse inside)
	Elevated bool
	// CreatedAt time
	CreatedAt time.Time
	// RootfsPropagation root fs propagation mode (rshared,shared,slave,...)
	RootFsPropagation RootFSPropagation
}

// ContainerModule defines rpc interface to containerd
type ContainerModule interface {
	// Run creates and starts a container on the node. It also auto
	// starts command defined by `entrypoint` inside the container
	// ns: tenant namespace
	// data: Container info
	Run(ns string, data Container) (ContainerID, error)

	// Exec executes a command in the container
	// stdout and stderr is ignored
	Exec(ns string, containerID string, timeout time.Duration, args ...string) error
	// ListNS list the name of all the container namespaces
	ListNS() ([]string, error)

	// List all the existing container IDs from a certain namespace ns
	// if ns is empty, then the container IDs from all existing namespaces will be return
	List(ns string) ([]ContainerID, error)

	// Inspect, return information about the container, given its container id
	Inspect(ns string, id ContainerID) (Container, error)
	Delete(ns string, id ContainerID) error
}
