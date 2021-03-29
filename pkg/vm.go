package pkg

import (
	"fmt"
	"net"
)

//go:generate zbusc -module vmd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+VMModule stubs/vmd_stub.go

// VMIface structure
type VMIface struct {
	// Tap device name
	Tap string
	// Mac address of the device
	MAC string
	// Address of the device in the form of cidr for ipv4
	IP4AddressCIDR net.IPNet
	// Gateway address for ipv4
	IP4GatewayIP net.IP
	// Full subnet for the IP4 resource. This allows configuration of networking for
	// non local subnets (i.e. NR on other nodes).
	// Does not need to be set for public ifaces
	IP4Net net.IPNet
	// Address of the device in the form of cidr for ipv6
	IP6AddressCIDR net.IPNet
	// Gateway address for ipv6
	IP6GatewayIP net.IP
	// Private or public network
	Public bool
}

// VMNetworkInfo structure
type VMNetworkInfo struct {
	// Interfaces for the vm network
	Ifaces []VMIface
	// Nameservers dns servers
	Nameservers []net.IP
	// NewStyle version of the network. This specifically differentiates between
	// the old way of statically setting the IP via the `IP` kernel parameter,
	// or using a boot script. The new way allows multiple interfaces and more
	// configuration
	NewStyle bool
}

// VMDisk specifies vm disk params
type VMDisk struct {
	// Size is disk size in Mib
	Path     string
	ReadOnly bool
	Root     bool
}

// VM config structure
type VM struct {
	// virtual machine name, or ID
	Name string
	// CPU is number of cores assigned to the VM
	CPU uint8
	// Memory size in Mib
	Memory int64
	// Network is network info
	Network VMNetworkInfo
	// KernelImage path to uncompressed linux kernel ELF
	KernelImage string
	// InitrdImage (optiona) path to initrd disk
	InitrdImage string
	// KernelArgs to override the default kernel arguments. (default: "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules")
	KernelArgs string
	// Disks are a list of disks that are going to
	// be auto allocated on the provided storage path
	Disks []VMDisk
	// If this flag is set, the VM module will not auto start
	// this machine hence, also no auto clean up when it exits
	// it's up to the caller to check for the machine status
	// and do clean up (module.Delete(vm)) when needed
	NoKeepAlive bool
}

// Validate vm data
func (vm *VM) Validate() error {
	missing := func(s string) bool {
		return len(s) == 0
	}

	if missing(vm.Name) {
		return fmt.Errorf("name is required")
	}

	if missing(vm.KernelImage) {
		return fmt.Errorf("kernel-image is required")
	}

	if vm.Memory < 512 {
		return fmt.Errorf("invalid memory must not be less than 512M")
	}

	if vm.CPU == 0 || vm.CPU > 32 {
		return fmt.Errorf("invalid cpu must be between 1 and 32")
	}

	return nil
}

// VMInfo returned by the inspect method
type VMInfo struct {
	// Flag for enabling/disabling Hyperthreading
	// Required: true
	HtEnabled bool

	// Memory size of VM
	// Required: true
	Memory int64

	// Number of vCPUs (either 1 or an even number)
	CPU int64
}

// VMModule defines the virtual machine module interface
type VMModule interface {
	Run(vm VM) error
	Inspect(name string) (VMInfo, error)
	Delete(name string) error
	Exists(name string) bool
	Logs(name string) (string, error)
	List() ([]string, error)
}
