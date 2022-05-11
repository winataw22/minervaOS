# Network module

## ZBus

Network module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| network|[network](#interface)| 0.0.1|

## Home Directory

network keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/network`|


## Interface

```go
//Networker is the interface for the network module
type Networker interface {
	// Create a new network resource
	CreateNR(Network) (string, error)
	// Delete a network resource
	DeleteNR(Network) error

	// Join a network (with network id) will create a new isolated namespace
	// that is hooked to the network bridge with a veth pair, and assign it a
	// new IP from the network resource range. The method return the new namespace
	// name.
	// The member name specifies the name of the member, and must be unique
	// The NetID is the network id to join
	Join(networkdID NetID, containerID string, addrs []string) (join Member, err error)

	// ZDBPrepare creates a network namespace with a macvlan interface into it
	// to allow the 0-db container to be publicly accessible
	// it retusn the name of the network namespace created
	ZDBPrepare() (string, error)

	// Addrs return the IP addresses of interface
	// if the interface is in a network namespace netns needs to be not empty
	Addrs(iface string, netns string) ([]net.IP, error)
}
```