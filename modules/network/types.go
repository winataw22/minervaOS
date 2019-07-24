package network

import (
	"encoding/json"
	"fmt"
	"net"
)

// IfaceType define the different public interface supported
type IfaceType string

const (
	//VlanIface means we use vlan for the public interface
	VlanIface IfaceType = "vlan"
	//MacVlanIface means we use macvlan for the public interface
	MacVlanIface IfaceType = "macvlan"
)

// IfaceInfo is the information about network interfaces
// that the node will publish publicly
// this is used to be able to configure public side of a node
type IfaceInfo struct {
	Name    string       `json:"name"`
	Addrs   []*net.IPNet `json:"addrs"`
	Gateway []net.IP     `json:"gateway"`
}

// MarshalJSON implements encoding/json.Unmarshaler
func (i *IfaceInfo) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Name    string   `json:"name"`
		Addrs   []string `json:"addrs"`
		Gateway []string `json:"gateway"`
	}{
		Name:    i.Name,
		Addrs:   make([]string, 0, len(i.Addrs)),
		Gateway: make([]string, 0, len(i.Gateway)),
	}
	for _, addr := range i.Addrs {
		if addr == nil {
			continue
		}
		tmp.Addrs = append(tmp.Addrs, addr.String())
	}
	for _, gw := range i.Gateway {
		if gw == nil {
			continue
		}
		tmp.Gateway = append(tmp.Gateway, gw.String())
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (i *IfaceInfo) UnmarshalJSON(b []byte) (err error) {
	tmp := struct {
		Name    string   `json:"name"`
		Addrs   []string `json:"addrs"`
		Gateway []string `json:"gateway"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*i = IfaceInfo{
		Name:    tmp.Name,
		Addrs:   make([]*net.IPNet, 0, len(tmp.Addrs)),
		Gateway: make([]net.IP, 0, len(tmp.Gateway)),
	}
	i.Name = tmp.Name
	for _, addr := range tmp.Addrs {
		if addr == "" {
			continue
		}
		ip, ipnet, err := net.ParseCIDR(addr)
		if err != nil {
			return err
		}
		ipnet.IP = ip
		i.Addrs = append(i.Addrs, ipnet)
	}
	for _, gw := range tmp.Gateway {
		if gw == "" {
			continue
		}
		ip := net.ParseIP(gw)
		i.Gateway = append(i.Gateway, ip)
	}

	return nil
}

// DefaultIP return the IP address of the interface that has a default gateway configured
// this function currently only check IPv6 addresses
func (i *IfaceInfo) DefaultIP() (net.IP, error) {
	if len(i.Gateway) <= 0 {
		return nil, fmt.Errorf("interface has not gateway")
	}

	for _, addr := range i.Addrs {
		if addr.IP.IsLinkLocalUnicast() ||
			addr.IP.IsLinkLocalMulticast() ||
			addr.IP.To4() != nil {
			continue
		}

		if addr.IP.To16() != nil {
			return addr.IP, nil
		}
	}
	return nil, fmt.Errorf("no ipv6 address with default gateway")
}

// PubIface is the configuration of the interface
// that is connected to the public internet
type PubIface struct {
	Master string `json:"master"`
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType `json:"iface_type"`
	Vlan int16     `json:"vlan"`
	// Macvlan net.HardwareAddr

	IPv4 *net.IPNet `json:"ip_v4"`
	IPv6 *net.IPNet `json:"ip_v6"`

	GW4 net.IP `json:"gw4"`
	GW6 net.IP `json:"gw6"`

	Version int `json:"version"`
}

// MarshalJSON implements encoding/json.Unmarshaler
func (p *PubIface) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Master  string `json:"master"`
		Type    string `json:"iface_type"`
		Vlan    int16  `json:"vlan"`
		IPv4    string `json:"ip_v4"`
		IPv6    string `json:"ip_v6"`
		GW4     string `json:"gw4"`
		GW6     string `json:"gw6"`
		Version int    `json:"version"`
	}{
		Master:  p.Master,
		Type:    string(p.Type),
		Vlan:    p.Vlan,
		Version: p.Version,
	}
	if p.IPv4 != nil {
		tmp.IPv4 = p.IPv4.String()
	}
	if p.IPv6 != nil {
		tmp.IPv6 = p.IPv6.String()
	}
	if p.GW4 != nil {
		tmp.GW4 = p.GW4.String()
	}
	if p.GW6 != nil {
		tmp.GW6 = p.GW6.String()
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (p *PubIface) UnmarshalJSON(b []byte) (err error) {
	tmp := struct {
		Master  string `json:"master"`
		Type    string `json:"iface_type"`
		Vlan    int16  `json:"vlan"`
		IPv4    string `json:"ip_v4"`
		IPv6    string `json:"ip_v6"`
		GW4     string `json:"gw4"`
		GW6     string `json:"gw6"`
		Version int    `json:"version"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*p = PubIface{}
	p.Master = tmp.Master
	p.Type = IfaceType(tmp.Type)
	p.Vlan = tmp.Vlan
	if tmp.IPv4 != "" {
		ip, ipnet, err := net.ParseCIDR(tmp.IPv4)
		if err != nil {
			return err
		}
		ipnet.IP = ip
		p.IPv4 = ipnet
	}
	if tmp.IPv6 != "" {
		ip, ipnet, err := net.ParseCIDR(tmp.IPv6)
		if err != nil {
			return err
		}
		ipnet.IP = ip
		p.IPv6 = ipnet
	}
	p.GW4 = net.ParseIP(tmp.GW4)
	p.GW6 = net.ParseIP(tmp.GW6)
	p.Version = tmp.Version
	return nil
}

// Node is the public information about a node
type Node struct {
	NodeID string `json:"node_id"`
	FarmID string `json:"farm_id"`

	Ifaces []*IfaceInfo `json:"ifaces"`

	PublicConfig *PubIface `json:"public_config"`
	ExitNode     bool      `json:"exit_node"`
}
