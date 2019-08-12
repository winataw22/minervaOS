# Container Module

## ZBus

Storage module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| container|[container](#interface)| 0.0.1|

## Home Directory
contd keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/containerd`|

## Introduction

The container module, is a proxy to [containerd](https://github.com/containerd/containerd). The proxy provides integration with zbus.

The implementation is the moment is straight forward, which includes preparing the OCI spec for the container, the tenant containerd namespace,
setting up proper capabilities, and finally creating the container instance on `containerd`.

The module is fully stateless, all container information is queried during runtime from `containerd`.

### zinit unit

`contd` must run after containerd is running, and the node boot process is complete. Since it doesn't keep state, no dependency on `stroaged` is needed

```yaml
exec: contd -broker unix:///var/run/redis.sock -root /var/cache/modules/containerd
after:
  - containerd
  - boot
```

## Interface

```go
package modules

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
	Source  string   // source of the mount point on the host
	Target  string   // target of mount inside the container
	Type    string   // mount type
	Options []string // mount options
}

//Container creation info
type Container struct {
	// Name of container
	Name string
	// path to the rootfs of the container
	RootFS string
	// Env env variables to container in format {'KEY=VALUE', 'KEY2=VALUE2'}
	Env []string
	// Network network info for container
	Network NetworkInfo
	// Mounts extra mounts for container
	Mounts []MountInfo
	// Entrypoint the process to start inside the container
	Entrypoint string
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool
}

// ContainerModule defines rpc interface to containerd
type ContainerModule interface {
	// Run creates and starts a container on the node. It also auto
	// starts command defined by `entrypoint` inside the container
	// ns: tenant namespace
	// data: Container info
	Run(ns string, data Container) (ContainerID, error)

	// Inspect, return information about the container, given its container id
	Inspect(ns string, id ContainerID) (Container, error)
	Delete(ns string, id ContainerID) error
}
```