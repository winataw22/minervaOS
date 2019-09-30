package modules

import (
	"encoding/json"
	"net"
)

// UnmarshalJSON implements encoding/json.Unmarshaler
func (p *Peer) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Subnet      string   `json:"subnet"`
		WGPublicKey string   `json:"wg_public_key"`
		AllowedIPs  []string `json:"allowed_ips"`
		Endpoint    string   `json:"endpoint"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*p = Peer{}

	if tmp.Subnet != "" {
		_, p.Subnet, err = net.ParseCIDR(tmp.Subnet)
		if err != nil {
			return err
		}
	}
	p.WGPublicKey = tmp.WGPublicKey
	p.WGPublicKey = tmp.WGPublicKey
	p.Endpoint = tmp.Endpoint
	p.AllowedIPs = make([]net.IPNet, len(tmp.AllowedIPs))
	for i, ip := range tmp.AllowedIPs {
		ip, ipnet, err := net.ParseCIDR(ip)
		if err != nil {
			return err
		}
		ipnet.IP = ip
		p.AllowedIPs[i] = *ipnet
	}

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (p *Peer) MarshalJSON() ([]byte, error) {

	allowedIPs := make([]string, len(p.AllowedIPs))
	for i, ip := range p.AllowedIPs {
		allowedIPs[i] = ip.String()
	}
	tmp := struct {
		Subnet      string   `json:"subnet"`
		WGPublicKey string   `json:"wg_public_key"`
		AllowedIPs  []string `json:"allowed_ips"`
		Endpoint    string   `json:"endpoint"`
	}{
		Subnet:      p.Subnet.String(),
		WGPublicKey: p.WGPublicKey,
		AllowedIPs:  allowedIPs,
		Endpoint:    p.Endpoint,
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) UnmarshalJSON(b []byte) error {
	tmp := struct {
		NodeID       string `json:"node_id"`
		Subnet       string `json:"subnet"`
		WGPrivateKey string `json:"wg_private_key"`
		WGPublicKey  string `json:"wg_public_key"`
		WGListenPort uint16 `json:"wg_listen_port"`
		Peers        []Peer `json:"peers"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*r = NetResource{}

	r.NodeID = tmp.NodeID
	r.Peers = tmp.Peers
	if tmp.Subnet != "" {
		_, r.Subnet, err = net.ParseCIDR(tmp.Subnet)
		if err != nil {
			return err
		}
	}
	r.WGPrivateKey = tmp.WGPrivateKey
	r.WGPublicKey = tmp.WGPublicKey
	r.WGListenPort = tmp.WGListenPort

	return nil
}

// MarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) MarshalJSON() ([]byte, error) {
	tmp := struct {
		NodeID       string `json:"node_id"`
		Subnet       string `json:"subnet"`
		Peers        []Peer `json:"peers"`
		WGPrivateKey string `json:"wg_private_key"`
		WGPublicKey  string `json:"wg_public_key"`
		WGListenPort uint16 `json:"wg_listen_port"`
	}{
		NodeID:       r.NodeID,
		Subnet:       r.Subnet.String(),
		Peers:        r.Peers,
		WGPrivateKey: r.WGPrivateKey,
		WGPublicKey:  r.WGPublicKey,
		WGListenPort: r.WGListenPort,
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (d *Network) UnmarshalJSON(b []byte) (err error) {
	tmp := struct {
		Name         string        `json:"name,omitempty"`
		NetID        string        `json:"net_id,omitempty"`
		NetResources []NetResource `json:"net_resources,omitempty"`
		IPRange      string        `json:"ip_range,omitempty"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*d = Network{}
	d.NetID = NetID(tmp.NetID)
	d.NetResources = tmp.NetResources
	if tmp.IPRange != "" {
		_, d.IPRange, err = net.ParseCIDR(tmp.IPRange)
		if err != nil {
			return err
		}
	}

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (d *Network) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Name         string        `json:"name,omitempty"`
		NetID        string        `json:"net_id,omitempty"`
		NetResources []NetResource `json:"net_resources,omitempty"`
		IPRange      string        `json:"ip_range,omitempty"`
	}{
		NetID:        string(d.NetID),
		NetResources: d.NetResources,
	}
	if d.IPRange != nil {
		tmp.IPRange = d.IPRange.String()
	}
	return json.Marshal(tmp)
}
