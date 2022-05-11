# VMD Module

## ZBus

Storage module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| vmd|[vmd](#interface)| 0.0.1|

## Home Directory

contd keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/containerd`|

## Introduction

The vmd module, manages all virtual machines processes, it provide the interface to, create, inspect, and delete virtual machines. It also monitor the vms to make sure they are re-spawned if crashed. Internally it uses `cloud-hypervisor` to start the Vm processes.

It also provide the interface to configure VM logs streamers.

### zinit unit

`contd` must run after containerd is running, and the node boot process is complete. Since it doesn't keep state, no dependency on `stroaged` is needed

```yaml
exec: vmd --broker unix:///var/run/redis.sock
after:
  - boot
  - networkd
```

## Interface

```go

// VMModule defines the virtual machine module interface
type VMModule interface {
	Run(vm VM) error
	Inspect(name string) (VMInfo, error)
	Delete(name string) error
	Exists(name string) bool
	Logs(name string) (string, error)
	List() ([]string, error)
	Metrics() (MachineMetrics, error)

	// VM Log streams

	// StreamCreate creates a stream for vm `name`
	StreamCreate(name string, stream Stream) error
	// delete stream by stream id.
	StreamDelete(id string) error
}
```
