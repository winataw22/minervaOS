# Node module

## Zbus

Flist module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
|host   |[host](#)| 0.0.1
|system   |[system](#)| 0.0.1
|events  |[events](#)| 0.0.1


## Introduction

This module is responsible of registering the node on the grid, and handling of grid events. The node daemon broadcast the intended events on zbus for other modules that are interested in those events.

The node also provide zbus interfaces to query some of the node information

```go

//SystemMonitor interface (provided by noded)
type SystemMonitor interface {
	NodeID() uint32
	Memory(ctx context.Context) <-chan VirtualMemoryStat
	CPU(ctx context.Context) <-chan TimesStat
	Disks(ctx context.Context) <-chan DisksIOCountersStat
	Nics(ctx context.Context) <-chan NicsIOCounterStat
}

// HostMonitor interface (provided by noded)
type HostMonitor interface {
	Uptime(ctx context.Context) <-chan time.Duration
}

// Events interface
type Events interface {
	PublicConfigEvent(ctx context.Context) <-chan PublicConfigEvent
	ContractCancelledEvent(ctx context.Context) <-chan ContractCancelledEvent
}
```
