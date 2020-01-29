package pkg

import (
	"fmt"
	"net"
)

//go:generate zbusc -module vmd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+VMModule stubs/vmd_stub.go

// VMNetworkInfo structure
type VMNetworkInfo struct {
	// Tap device name
	Tap string
	// Mac address of the device
	MAC string
	// Address of the device in the form of cidr
	AddressCIDR net.IPNet
	// Gateway gateway address
	GatewayIP net.IP
	// Nameservers dns servers
	Nameservers []net.IP
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
	Exists(id string) bool
}
