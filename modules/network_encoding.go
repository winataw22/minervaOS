package modules

import (
	"encoding/json"
	"fmt"
	"net"
)

var _reachabilityV4ToStr = map[ReachabilityV4]string{
	ReachabilityV4Public: "public",
	ReachabilityV4Hidden: "hidden",
}

var _reachabilityV4FromStr = map[string]ReachabilityV4{
	"public": ReachabilityV4Public,
	"hidden": ReachabilityV4Hidden,
}

var _reachabilityV6ToStr = map[ReachabilityV6]string{
	ReachabilityV6Public: "public",
	ReachabilityV6ULA:    "hidden",
}

var _reachabilityV6FromStr = map[string]ReachabilityV6{
	"public": ReachabilityV6Public,
	"hidden": ReachabilityV6ULA,
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (n *NodeID) UnmarshalJSON(b []byte) error {
	tmp := struct {
		ID             string `json:"id"`
		FarmerID       string `json:"farmer_id"`
		ReachabilityV4 string `json:"reachability_v4"`
		ReachabilityV6 string `json:"reachability_v6"`
	}{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	var ok bool

	*n = NodeID{}
	n.ID = tmp.ID
	n.FarmerID = tmp.FarmerID
	n.ReachabilityV4, ok = _reachabilityV4FromStr[tmp.ReachabilityV4]
	if !ok {
		return fmt.Errorf("unsupported ReachabilityV4 %s", tmp.ReachabilityV4)
	}
	n.ReachabilityV6, ok = _reachabilityV6FromStr[tmp.ReachabilityV6]
	if !ok {
		return fmt.Errorf("unsupported ReachabilityV4 %s", tmp.ReachabilityV4)
	}

	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (p *Peer) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Type       string `json:"type"`
		Prefix     string `json:"prefix"`
		Connection struct {
			IP   string `json:"ip"`
			Port uint16 `json:"port"`
			Key  string `json:"key"`
		}
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*p = Peer{}
	if tmp.Type != "wireguard" {
		return fmt.Errorf("unsupported peer connection type %s", tmp.Type)
	}
	p.Type = ConnTypeWireguard
	_, p.Prefix, err = net.ParseCIDR(tmp.Prefix)
	if err != nil {
		return err
	}
	p.Connection.IP = net.ParseIP(tmp.Connection.IP)
	p.Connection.Port = tmp.Connection.Port
	p.Connection.Key = tmp.Connection.Key

	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) UnmarshalJSON(b []byte) error {
	tmp := struct {
		NodeID    NodeID  `json:"node_id"`
		Prefix    string  `json:"prefix"`
		LinkLocal string  `json:"link_local"`
		Peers     []*Peer `json:"peers"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*r = NetResource{}

	r.NodeID = tmp.NodeID
	r.Peers = tmp.Peers
	_, r.Prefix, err = net.ParseCIDR(tmp.Prefix)
	if err != nil {
		return err
	}
	_, r.LinkLocal, err = net.ParseCIDR(tmp.LinkLocal)
	if err != nil {
		return err
	}

	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (i *Ipv4Conf) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Cidr      string `json:"cidr"`
		Gateway   string `json:"gateway"`
		Metric    uint32 `json:"metric"`
		Iface     string `json:"iface"`
		EnableNat bool   `json:"enable_nat"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*i = Ipv4Conf{}
	_, i.CIDR, err = net.ParseCIDR(tmp.Cidr)
	if err != nil {
		return err
	}
	i.Gateway = net.ParseIP(tmp.Gateway)
	i.Metric = tmp.Metric
	i.Iface = tmp.Iface
	i.EnableNAT = tmp.EnableNat

	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (i *Ipv6Conf) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Addr    string `json:"addr"`
		Gateway string `json:"gateway"`
		Metric  uint32 `json:"metric"`
		Iface   string `json:"iface"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*i = Ipv6Conf{}
	i.Addr = net.ParseIP(tmp.Addr)
	i.Gateway = net.ParseIP(tmp.Gateway)
	i.Metric = tmp.Metric
	i.Iface = tmp.Iface
	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (d *DNAT) UnmarshalJSON(b []byte) error {
	tmp := struct {
		InternalIP   string `json:"internal_ip"`
		InternalPort uint16 `json:"internal_port"`
		ExternalIP   string `json:"external_ip"`
		ExternalPort uint16 `json:"external_port"`
		Protocol     string `json:"protocol"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*d = DNAT{}
	d.InternalIP = net.ParseIP(tmp.InternalIP)
	d.InternalPort = tmp.InternalPort
	d.ExternalIP = net.ParseIP(tmp.ExternalIP)
	d.ExternalPort = tmp.ExternalPort
	d.Protocol = tmp.Protocol

	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (e *ExitPoint) UnmarshalJSON(b []byte) error {
	tmp := struct {
		NetResource *NetResource
		Ipv4Conf    Ipv4Conf `json:"ipv4_conf"`
		Ipv4Dnat    []DNAT   `json:"ipv4_dnat"`
		Ipv6Conf    Ipv6Conf `json:"ipv6_conf"`
		Ipv6Allow   []net.IP `json:"ipv6_allow`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*e = ExitPoint{}
	e.NetResource = tmp.NetResource
	e.Ipv4Conf = tmp.Ipv4Conf
	e.Ipv4DNAT = tmp.Ipv4Dnat
	e.Ipv6Conf = tmp.Ipv6Conf
	e.Ipv6Allow = tmp.Ipv6Allow

	return nil
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (d *Network) UnmarshalJSON(b []byte) error {
	tmp := struct {
		NetworkID    string         `json:"network_id"`
		Resources    []*NetResource `json:"resources"`
		ExitPoint    *ExitPoint     `json:"exit_point"`
		AllocationNR int8           `json:"allocation_nr"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*d = Network{}
	d.NetID = NetID(tmp.NetworkID)
	d.Resources = tmp.Resources
	d.Exit = tmp.ExitPoint
	d.AllocationNR = tmp.AllocationNR

	return nil
}
