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
	if tmp.ReachabilityV4 != "" {
		n.ReachabilityV4, ok = _reachabilityV4FromStr[tmp.ReachabilityV4]
		if !ok {
			return fmt.Errorf("unsupported ReachabilityV4 %s", tmp.ReachabilityV4)
		}
	}
	if tmp.ReachabilityV6 != "" {
		n.ReachabilityV6, ok = _reachabilityV6FromStr[tmp.ReachabilityV6]
		if !ok {
			return fmt.Errorf("unsupported ReachabilityV4 %s", tmp.ReachabilityV4)
		}
	}

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (n *NodeID) MarshalJSON() ([]byte, error) {
	tmp := struct {
		ID             string `json:"id"`
		FarmerID       string `json:"farmer_id"`
		ReachabilityV4 string `json:"reachability_v4"`
		ReachabilityV6 string `json:"reachability_v6"`
	}{
		ID:             n.ID,
		FarmerID:       n.FarmerID,
		ReachabilityV4: _reachabilityV4ToStr[n.ReachabilityV4],
		ReachabilityV6: _reachabilityV6ToStr[n.ReachabilityV6],
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (p *Peer) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Type       string `json:"type"`
		Prefix     string `json:"prefix"`
		Connection struct {
			IP         string `json:"ip"`
			Port       uint16 `json:"port"`
			Key        string `json:"key"`
			PrivateKey string `json:"private_key"`
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
	if tmp.Prefix != "" {
		_, p.Prefix, err = net.ParseCIDR(tmp.Prefix)
		if err != nil {
			return err
		}
	}
	p.Connection.IP = net.ParseIP(tmp.Connection.IP)
	p.Connection.Port = tmp.Connection.Port
	p.Connection.Key = tmp.Connection.Key
	p.Connection.PrivateKey = tmp.Connection.PrivateKey

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (p *Peer) MarshalJSON() ([]byte, error) {

	if p.Type != ConnTypeWireguard {
		return nil, fmt.Errorf("unsupported connection type")
	}

	type Connection struct {
		IP         string `json:"ip"`
		Port       uint16 `json:"port"`
		Key        string `json:"key"`
		PrivateKey string `json:"private_key"`
	}
	tmp := struct {
		Type       string `json:"type"`
		Prefix     string `json:"prefix"`
		Connection Connection
	}{
		Type:   "wireguard",
		Prefix: p.Prefix.String(),
		Connection: Connection{
			IP:         p.Connection.IP.String(),
			Port:       p.Connection.Port,
			Key:        p.Connection.Key,
			PrivateKey: p.Connection.PrivateKey,
		},
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) UnmarshalJSON(b []byte) error {
	tmp := struct {
		NodeID    *NodeID `json:"node_id"`
		Prefix    string  `json:"prefix"`
		LinkLocal string  `json:"link_local"`
		Peers     []*Peer `json:"peers"`
		ExitPoint bool    `json:"exit_point"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*r = NetResource{}

	r.NodeID = tmp.NodeID
	r.Peers = tmp.Peers
	if tmp.Prefix != "" {
		_, r.Prefix, err = net.ParseCIDR(tmp.Prefix)
		if err != nil {
			return err
		}
	}
	if tmp.LinkLocal != "" {
		var ip net.IP
		ip, r.LinkLocal, err = net.ParseCIDR(tmp.LinkLocal)
		if err != nil {
			return err
		}
		r.LinkLocal.IP = ip
	}
	r.ExitPoint = tmp.ExitPoint

	return nil
}

// MarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) MarshalJSON() ([]byte, error) {
	tmp := struct {
		NodeID    *NodeID `json:"node_id"`
		Prefix    string  `json:"prefix"`
		LinkLocal string  `json:"link_local"`
		Peers     []*Peer `json:"peers"`
		ExitPoint bool    `json:"exit_point"`
	}{
		NodeID:    r.NodeID,
		Prefix:    r.Prefix.String(),
		LinkLocal: r.LinkLocal.String(),
		Peers:     r.Peers,
		ExitPoint: r.ExitPoint,
	}
	return json.Marshal(tmp)
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
	if tmp.Cidr != "" {
		_, i.CIDR, err = net.ParseCIDR(tmp.Cidr)
		if err != nil {
			return err
		}
	}
	i.Gateway = net.ParseIP(tmp.Gateway)
	i.Metric = tmp.Metric
	i.Iface = tmp.Iface
	i.EnableNAT = tmp.EnableNat

	return nil
}

// MarshalJSON implements encoding/json.Unmarshaler
func (i *Ipv4Conf) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Cidr      string `json:"cidr"`
		Gateway   string `json:"gateway"`
		Metric    uint32 `json:"metric"`
		Iface     string `json:"iface"`
		EnableNat bool   `json:"enable_nat"`
	}{
		Cidr:      i.CIDR.String(),
		Gateway:   i.Gateway.String(),
		Metric:    i.Metric,
		Iface:     i.Iface,
		EnableNat: i.EnableNAT,
	}

	return json.Marshal(tmp)
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
	if tmp.Addr != "" {
		ip, ipnet, err := net.ParseCIDR(tmp.Addr)
		if err != nil {
			return err
		}
		ipnet.IP = ip
		i.Addr = ipnet
	}
	i.Gateway = net.ParseIP(tmp.Gateway)
	i.Metric = tmp.Metric
	i.Iface = tmp.Iface
	return nil
}

// MarshalJSON implements encoding/json.Unmarshaler
func (i *Ipv6Conf) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Addr    string `json:"addr"`
		Gateway string `json:"gateway"`
		Metric  uint32 `json:"metric"`
		Iface   string `json:"iface"`
	}{
		Addr:    i.Addr.String(),
		Gateway: i.Gateway.String(),
		Metric:  i.Metric,
		Iface:   i.Iface,
	}

	return json.Marshal(tmp)
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

// MarshalJSON implements encoding/json.Unmarshaler
func (d *DNAT) MarshalJSON() ([]byte, error) {
	tmp := struct {
		InternalIP   string `json:"internal_ip"`
		InternalPort uint16 `json:"internal_port"`
		ExternalIP   string `json:"external_ip"`
		ExternalPort uint16 `json:"external_port"`
		Protocol     string `json:"protocol"`
	}{
		InternalIP:   d.InternalIP.String(),
		InternalPort: d.InternalPort,
		ExternalIP:   d.InternalIP.String(),
		ExternalPort: d.ExternalPort,
		Protocol:     d.Protocol,
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (e *ExitPoint) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Ipv4Conf  *Ipv4Conf `json:"ipv4_conf"`
		Ipv4Dnat  []*DNAT   `json:"ipv4_dnat"`
		Ipv6Conf  *Ipv6Conf `json:"ipv6_conf"`
		Ipv6Allow []net.IP  `json:"ipv6_allow"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*e = ExitPoint{}
	e.Ipv4Conf = tmp.Ipv4Conf
	e.Ipv4DNAT = tmp.Ipv4Dnat
	e.Ipv6Conf = tmp.Ipv6Conf
	e.Ipv6Allow = tmp.Ipv6Allow

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (e *ExitPoint) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Ipv4Conf  *Ipv4Conf `json:"ipv4_conf"`
		Ipv4Dnat  []*DNAT   `json:"ipv4_dnat"`
		Ipv6Conf  *Ipv6Conf `json:"ipv6_conf"`
		Ipv6Allow []string  `json:"ipv6_allow"`
	}{
		Ipv4Conf: e.Ipv4Conf,
		Ipv4Dnat: e.Ipv4DNAT,
		Ipv6Conf: e.Ipv6Conf,
	}
	ips := make([]string, 0, len(e.Ipv6Allow))
	for i, ip := range e.Ipv6Allow {
		ips[i] = ip.String()
	}
	tmp.Ipv6Allow = ips
	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (d *Network) UnmarshalJSON(b []byte) (err error) {
	tmp := struct {
		NetworkID    string         `json:"network_id"`
		Resources    []*NetResource `json:"resources"`
		PrefixZero   string         `json:"prefix_zero"`
		ExitPoint    *ExitPoint     `json:"exit_point"`
		AllocationNR int8           `json:"allocation_nr"`
		Version      uint32         `json:"version"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*d = Network{}
	d.NetID = NetID(tmp.NetworkID)
	d.Resources = tmp.Resources
	d.Exit = tmp.ExitPoint
	d.AllocationNR = tmp.AllocationNR
	if tmp.PrefixZero != "" {
		_, d.PrefixZero, err = net.ParseCIDR(tmp.PrefixZero)
		if err != nil {
			return err
		}
	}
	d.Version = tmp.Version

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (d *Network) MarshalJSON() ([]byte, error) {
	tmp := struct {
		NetworkID    string         `json:"network_id"`
		Resources    []*NetResource `json:"resources"`
		PrefixZero   string         `json:"prefix_zero"`
		ExitPoint    *ExitPoint     `json:"exit_point"`
		AllocationNR int8           `json:"allocation_nr"`
		Version      uint32         `json:"version"`
	}{
		NetworkID:    string(d.NetID),
		Resources:    d.Resources,
		ExitPoint:    d.Exit,
		AllocationNR: d.AllocationNR,
		Version:      d.Version,
	}
	if d.PrefixZero != nil {
		tmp.PrefixZero = d.PrefixZero.String()
	}
	return json.Marshal(tmp)
}
