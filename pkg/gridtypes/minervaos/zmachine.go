package zos

import (
	"fmt"
	"io"
	"net"
	"sort"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// MachineInterface structure
type MachineInterface struct {
	// Network name (znet name) to join
	Network gridtypes.Name `json:"network"`
	// IP of the zmachine on this network must be a valid Ip in the
	// selected network
	IP net.IP `json:"ip"`
}

// MachineNetwork structure
type MachineNetwork struct {
	// PublicIP optional public IP attached to this machine. If set
	// it must be a valid name of a PublicIP workload in the same deployment
	PublicIP gridtypes.Name `json:"public_ip"`
	// Planetary support planetary network
	Planetary bool `json:"planetary"`
	// Interfaces list of user znets to join
	Interfaces []MachineInterface `json:"interfaces"`
}

// Challenge builder
func (n *MachineNetwork) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", n.PublicIP); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", n.Planetary); err != nil {
		return err
	}

	for _, inf := range n.Interfaces {
		if _, err := fmt.Fprintf(w, "%s", inf.Network); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s", inf.IP.String()); err != nil {
			return err
		}
	}

	return nil
}

// MachineCapacity structure
type MachineCapacity struct {
	CPU    uint8          `json:"cpu"`
	Memory gridtypes.Unit `json:"memory"`
}

// Challenge builder
func (c *MachineCapacity) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", c.CPU); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", c.Memory); err != nil {
		return err
	}

	return nil
}

// MachineMount structure
type MachineMount struct {
	// Name is name of a zmount. The name must be a valid zmount
	// in the same deployment as the zmachine
	Name gridtypes.Name `json:"name"`
	// Mountpoint inside the container. Not used if the zmachine
	// is running in a vm mode.
	Mountpoint string `json:"mountpoint"`
}

// Challenge builder
func (m *MachineMount) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", m.Name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", m.Mountpoint); err != nil {
		return err
	}

	return nil
}

// ZMachine reservation data
type ZMachine struct {
	// Flist of the zmachine, must be a valid url to an flist.
	FList string `json:"flist"`
	// Network configuration for machine network
	Network MachineNetwork `json:"network"`
	// Size of zmachine (deprecated)
	Size uint8 `json:"size"` // deprecated, use compute_capacity instead
	// ComputeCapacity configuration for machine cpu+memory
	ComputeCapacity MachineCapacity `json:"compute_capacity"`
	// Mounts configure mounts/disks attachments to this machine
	Mounts []MachineMount `json:"mounts"`

	// following items are only available in container mode. if FList is for a container
	// not a VM.

	// Entrypoint entrypoint of the container, if not set the configured one from the flist
	// is going to be used
	Entrypoint string `json:"entrypoint"`
	// Env variables available for a container
	Env map[string]string `json:"env"`
}

// Valid implementation
func (v ZMachine) Valid(getter gridtypes.WorkloadGetter) error {
	if len(v.Network.Interfaces) != 1 {
		return fmt.Errorf("only one network private network is supported at the moment")
	}

	for _, inf := range v.Network.Interfaces {
		if inf.IP.To4() == nil && inf.IP.To16() == nil {
			return fmt.Errorf("invalid IP")
		}
	}

	if v.Size == 0 {
		if v.ComputeCapacity.CPU == 0 {
			return fmt.Errorf("cpu capcity can't be 0")
		}
		if v.ComputeCapacity.Memory < 250*gridtypes.Megabyte {
			return fmt.Errorf("mem capacity can't be less that 250M")
		}
	} else if v.Size < 1 || v.Size > 18 {
		return fmt.Errorf("unsupported vm size %d, only size 1 to 18 are supported", v.Size)
	}
	if !v.Network.PublicIP.IsEmpty() {
		wl, err := getter.Get(v.Network.PublicIP)
		if err != nil {
			return fmt.Errorf("public ip is not found")
		}

		if wl.Type != PublicIPType {
			return errors.Wrapf(err, "workload of name '%s' is not a public ip", v.Network.PublicIP)
		}
	}

	for _, ifc := range v.Network.Interfaces {
		if ifc.Network == "ygg" { //reserved temporary
			return fmt.Errorf("ygg is not a valid network name")
		}
	}

	return nil
}

// Capacity implementation
func (v ZMachine) Capacity() (gridtypes.Capacity, error) {
	if v.Size > 0 {
		rsu, ok := vmSize[v.Size]
		if !ok {
			return gridtypes.Capacity{}, fmt.Errorf("VM size %d is not supported", v.Size)
		}
		return rsu, nil
	}

	// otherwise
	return gridtypes.Capacity{
		CRU: uint64(v.ComputeCapacity.CPU),
		MRU: v.ComputeCapacity.Memory,
	}, nil
}

// Challenge creates signature challenge
func (v ZMachine) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%s", v.FList); err != nil {
		return err
	}

	if err := v.Network.Challenge(b); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%d", v.Size); err != nil {
		return err
	}

	if err := v.ComputeCapacity.Challenge(b); err != nil {
		return err
	}

	for _, mnt := range v.Mounts {
		if err := mnt.Challenge(b); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(b, "%s", v.Entrypoint); err != nil {
		return err
	}

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

	if err := encodeEnv(b, v.Env); err != nil {
		return err
	}

	return nil
}

// ZMachineResult result returned by VM reservation
type ZMachineResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}
