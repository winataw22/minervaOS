package zos

import (
	"fmt"
	"io"
	"math"
	"net"
	"sort"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Member struct
type Member struct {
	Network string `json:"network"`
	// IP to give to the container
	IPs         []net.IP `json:"ips"`
	PublicIP6   bool     `json:"public_ip6"`
	YggdrasilIP bool     `json:"yggdrasil_ip"`
}

// Challenge creates signature challenge
func (m Member) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", m.Network); err != nil {
		return err
	}

	for _, addr := range m.IPs {
		if _, err := fmt.Fprintf(w, "%s", addr.String()); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%t", m.PublicIP6); err != nil {
		return err
	}
	// TODO: re-enable when working on https://github.com/threefoldtech/zos/issues/868
	// if _, err := fmt.Fprintf(w, "%t", n.YggdrasilIP); err != nil {
	// 	return err
	// }
	return nil
}

// Mount defines a container volume mounted inside the container
type Mount struct {
	Volume     string `json:"volume"`
	Mountpoint string `json:"mountpoint"`
}

// Challenge creates signature challenge
func (m Mount) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", m.Volume); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s", m.Mountpoint); err != nil {
		return err
	}
	return nil
}

// Logs defines a custom backend with variable settings
type Logs struct {
	Type string   `json:"type"`
	Data LogsData `json:"data"`
}

// LogsData structure
type LogsData struct {
	// Stdout is the redis url for stdout (redis://host/channel)
	Stdout string `json:"stdout"`

	// Stderr is the redis url for stderr (redis://host/channel)
	Stderr string `json:"stderr"`
}

// Stats defines a stats backend
type Stats struct {
	Type     string `bson:"type" json:"type"`
	Endpoint string `bson:"endpoint" json:"endpoint"`
}

// ContainerCapacity is the amount of resource to allocate to the container
type ContainerCapacity struct {
	// Number of CPU
	CPU uint `json:"cpu"`
	// Memory in MiB
	Memory uint64 `json:"memory"`
	//DiskType is the type of disk to use for root fs
	DiskType DeviceType `json:"disk_type"`
	// DiskSize of the root fs in MiB
	DiskSize uint64 `json:"disk_size"`
}

// Challenge creates signature challenge
func (c ContainerCapacity) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", c.CPU); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%d", c.Memory); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%d", c.DiskSize); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s", c.DiskType.String()); err != nil {
		return err
	}
	return nil
}

func (c ContainerCapacity) capacity() (gridtypes.Capacity, error) {
	rsu := gridtypes.Capacity{
		CRU: uint64(c.CPU),
		// round mru to 4 digits precision
		MRU: uint64(math.Round(float64(c.Memory)/1024*10000) / 10000),
	}
	storageSize := math.Round(float64(c.DiskSize)/1024*10000) / 10000
	storageSize = math.Max(0, storageSize-50) // we offer the 50 first GB of storage for container root
	switch c.DiskType {
	case HDDDevice:
		rsu.HRU = uint64(storageSize)
	case SSDDevice:
		rsu.SRU = uint64(storageSize)
	}

	return rsu, nil
}

//Container creation info
type Container struct {
	// URL of the flist
	FList string `json:"flist"`
	// URL of the storage backend for the flist
	HubURL string `json:"hub_url"`
	// Env env variables to container in format
	Env map[string]string `json:"env"`
	// Entrypoint the process to start inside the container
	Entrypoint string `json:"entrypoint"`
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool `json:"interactive"`
	// Mounts extra mounts in the container
	Mounts []Mount `json:"mounts"`
	// Network network info for container
	Network Member `json:"network"`
	// ContainerCapacity is the amount of resource to allocate to the container
	ContainerCapacity ContainerCapacity `json:"capacity"`
	// Logs contains a list of endpoint where to send containerlogs
	Logs []Logs `json:"logs,omitempty"`
	// Stats container metrics backend
	Stats []Stats `json:"stats,omitempty"`
}

// Valid implement the validation interface for container data
func (c Container) Valid(getter gridtypes.WorkloadGetter) error {
	for _, mnt := range c.Mounts {
		wl, err := getter.Get(mnt.Volume)
		if err != nil {
			return errors.Wrap(err, "mount volume is not found")
		}
		if wl.Type != VolumeType {
			return errors.Wrapf(err, "workload of name '%s' is not a volume", mnt.Volume)
		}
	}

	//TODO: also validate the network
	//NOTE: we leave this out at the moment because network
	// still can be global per user.
	return nil
}

// Challenge implementation
func (c Container) Challenge(w io.Writer) error {
	encodeEnv := func(w io.Writer, env map[string]string) error {

		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if _, err := fmt.Fprintf(w, "%s=%s", k, env[k]); err != nil {
				return err
			}
		}

		return nil
	}

	if _, err := fmt.Fprintf(w, "%s", c.FList); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s", c.HubURL); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s", c.Entrypoint); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%t", c.Interactive); err != nil {
		return err
	}
	if err := encodeEnv(w, c.Env); err != nil {
		return err
	}

	for _, v := range c.Mounts {
		if err := v.Challenge(w); err != nil {
			return err
		}
	}

	if err := c.Network.Challenge(w); err != nil {
		return err
	}

	if err := c.ContainerCapacity.Challenge(w); err != nil {
		return err
	}

	return nil
}

// Capacity implementation
func (c Container) Capacity() (gridtypes.Capacity, error) {
	return c.ContainerCapacity.capacity()
}

// ContainerResult is the information return to the BCDB
// after deploying a container
type ContainerResult struct {
	ID    string `json:"id"`
	IPv6  string `json:"ipv6"`
	IPv4  string `json:"ipv4"`
	IPYgg string `json:"yggdrasil"`
}
