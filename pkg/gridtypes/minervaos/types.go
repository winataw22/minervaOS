package zos

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	// ZMountType type
	ZMountType gridtypes.WorkloadType = "zmount"
	// NetworkType type
	NetworkType gridtypes.WorkloadType = "network"
	// ZDBType type
	ZDBType gridtypes.WorkloadType = "zdb"
	// ZMachineType type
	ZMachineType gridtypes.WorkloadType = "zmachine"
	//PublicIPType type
	PublicIPType gridtypes.WorkloadType = "ipv4"
	// GatewayNameProxyType type
	GatewayNameProxyType gridtypes.WorkloadType = "gateway-name-proxy"
	// GatewayFQDNProxyType type
	GatewayFQDNProxyType gridtypes.WorkloadType = "gateway-fqdn-proxy"
)

func init() {
	gridtypes.RegisterType(ZMountType, ZMount{})
	gridtypes.RegisterType(NetworkType, Network{})
	gridtypes.RegisterType(ZDBType, ZDB{})
	gridtypes.RegisterType(ZMachineType, ZMachine{})
	gridtypes.RegisterType(PublicIPType, PublicIP{})
	gridtypes.RegisterType(GatewayNameProxyType, GatewayNameProxy{})
	gridtypes.RegisterType(GatewayFQDNProxyType, GatewayFQDNProxy{})
}

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice DeviceType = "ssd"
	HDDDevice DeviceType = "hdd"
)

func (d DeviceType) String() string {
	return string(d)
}

// Valid validates device type
func (d DeviceType) Valid() error {
	if d != SSDDevice && d != HDDDevice {
		return fmt.Errorf("invalid device type")
	}
	return nil
}
