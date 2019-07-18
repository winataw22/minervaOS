package tnodb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/vishvananda/netlink"
)

type httpTNoDB struct {
	baseURL string
}

// NewHTTPHTTPTNoDB create an a client to a TNoDB reachable over HTTP
func NewHTTPHTTPTNoDB(url string) network.TNoDB {
	return &httpTNoDB{baseURL: url}
}

func (s *httpTNoDB) RegisterAllocation(farm identity.Identifier, allocation *net.IPNet) error {
	req := struct {
		FarmerID string `json:"farmer_id"`
		Alloc    string `json:"allocation"`
	}{
		FarmerID: farm.Identity(),
		Alloc:    allocation.String(),
	}
	buf := bytes.Buffer{}
	err := json.NewEncoder(&buf).Encode(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(s.baseURL+"/allocations", "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v", string(b))
		return fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	return nil
}

func (s *httpTNoDB) RequestAllocation(farm identity.Identifier) (*net.IPNet, error) {
	url := fmt.Sprintf("%s/%s/%s", s.baseURL, "allocations", farm.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v", string(b))
		return nil, fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	data := struct {
		Alloc string `json:"allocation"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	_, alloc, err := net.ParseCIDR(data.Alloc)
	if err != nil {
		return nil, err
	}

	return alloc, nil
}

func (s *httpTNoDB) PublishInterfaces() error {
	type ifaceInfo struct {
		Name    string   `json:"name"`
		Addrs   []string `json:"addrs"`
		Gateway []string `json:"gateway"`
	}
	output := []ifaceInfo{}

	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return err
	}

	for _, link := range ifaceutil.LinkFilter(links, []string{"device", "bridge"}) {
		if !ifaceutil.IsPlugged(link.Attrs().Name) {
			log.Info().Msgf("not plugged %s", link.Attrs().Name)
			continue
		}

		_, gw, err := ifaceutil.HasDefaultGW(link)
		if err != nil {
			return err
		}

		addrs, err := linkAddrs(link)
		if err != nil {
			return err
		}

		info := ifaceInfo{
			Name:  link.Attrs().Name,
			Addrs: addrs,
		}
		if gw != nil {
			info.Gateway = []string{gw.String()}
		}
		output = append(output, info)
	}

	log.Info().Msgf("%+v", output)

	nodeID, err := identity.LocalNodeID()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/nodes/%s/interfaces", s.baseURL, nodeID.Identity())
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(output); err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status received: %s", resp.Status)
	}

	return nil
}

func (s *httpTNoDB) ConfigurePublicIface(node identity.Identifier, ips []*net.IPNet, gws []net.IP, iface string) error {
	output := struct {
		Iface string   `json:"iface"`
		IPs   []string `json:"ips"`
		GWs   []string `json:"gateways"`
		// Type todo allow to chose type of connection
	}{
		Iface: iface,
		IPs:   make([]string, len(ips)),
		GWs:   make([]string, len(gws)),
	}

	for i := range ips {
		output.IPs[i] = ips[i].String()
		output.GWs[i] = gws[i].String()
	}

	url := fmt.Sprintf("%s/nodes/%s/configure_public", s.baseURL, node.Identity())

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(output); err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status received: %s", resp.Status)
	}
	return nil
}

func (s *httpTNoDB) SelectExitNode(node identity.Identifier) error {
	url := fmt.Sprintf("%s/nodes/%s/select_exit", s.baseURL, node.Identity())

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status received: %s", resp.Status)
	}
	return nil
}

func (s *httpTNoDB) ReadPubIface(node identity.Identifier) (*network.PubIface, error) {

	input := struct {
		Master  string
		IPv4    string
		IPv6    string
		GW4     string
		GW6     string
		Version int
	}{}

	url := fmt.Sprintf("%s/nodes/%s", s.baseURL, node.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, network.ErrNoPubIface
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong response status: %v", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&input); err != nil {
		return nil, err
	}

	exitIface := &network.PubIface{}
	exitIface.Version = input.Version
	exitIface.Master = input.Master
	exitIface.Type = network.MacVlanIface
	if input.IPv4 != "" {
		ip, ipnet, err := net.ParseCIDR(input.IPv4)
		if err != nil {
			return nil, err
		}
		ipnet.IP = ip
		exitIface.IPv4 = ipnet
	}
	if input.IPv6 != "" {
		ip, ipnet, err := net.ParseCIDR(input.IPv6)
		if err != nil {
			return nil, err
		}
		ipnet.IP = ip
		exitIface.IPv6 = ipnet
	}
	if input.GW4 != "" {
		gw := net.ParseIP(input.GW4)
		exitIface.GW4 = gw
	}
	if input.GW6 != "" {
		gw := net.ParseIP(input.GW6)
		exitIface.GW6 = gw
	}

	return exitIface, nil
}

func (s *httpTNoDB) PublishWireguarKey(key string, nodeID string, netID modules.NetID) error {
	url := fmt.Sprintf("%s/networks/%s/%s/wgkeys", s.baseURL, netID, nodeID)
	buf := &bytes.Buffer{}

	output := struct {
		Key string `json:"key"`
	}{Key: key}

	if err := json.NewEncoder(buf).Encode(output); err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("wrong response status received: %s", resp.Status)
	}

	return nil
}

func (s *httpTNoDB) GetNetwork(netid modules.NetID) (*modules.Network, error) {
	url := fmt.Sprintf("%s/networks/%s", s.baseURL, string(netid))
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	network := &modules.Network{}
	if err := json.NewDecoder(resp.Body).Decode(network); err != nil {
		return nil, err
	}
	return network, nil
}

func (s *httpTNoDB) CreateNetwork(farmID string) (*modules.Network, error) {
	networkReq := struct {
		ExitFarm string `json:"exit_farm"`
	}{ExitFarm: farmID}

	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(networkReq); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/networks", s.baseURL)
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("wrong response status received: %s %s", resp.Status, string(body))
	}

	network := &modules.Network{}
	if err := json.NewDecoder(resp.Body).Decode(network); err != nil {
		log.Error().Err(err).Msg("failed to decode network json")
		return nil, err
	}

	return network, nil
}

func (s *httpTNoDB) JoinNetwork(nodeID identity.Identifier, id modules.NetID, WGPort uint16, WGPubKey string) (*modules.Network, error) {

	req := struct {
		WGPort   uint16 `json:"wg_port"`
		WGPubKey string `json:"wg_public_key"`
		NodeID   string `json:"node_id"`
	}{
		WGPort:   WGPort,
		WGPubKey: WGPubKey,
		NodeID:   nodeID.Identity(),
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(req); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/networks/%s", s.baseURL, id)
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("wrong response status received: %s %s", resp.Status, string(body))
	}

	network := &modules.Network{}
	if err := json.NewDecoder(resp.Body).Decode(network); err != nil {
		log.Error().Err(err).Msg("failed to decode network json")
		return nil, err
	}

	return network, nil
}

func (s *httpTNoDB) AddUser(user identity.Identifier, id modules.NetID, WGPubKey string) (*modules.Network, error) {

	req := struct {
		WGPubKey string `json:"wg_public_key"`
		UserID   string `json:"user_id"`
	}{
		WGPubKey: WGPubKey,
		UserID:   user.Identity(),
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(req); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/networks/%s/user", s.baseURL, id)
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("wrong response status received: %s %s", resp.Status, string(body))
	}

	network := &modules.Network{}
	if err := json.NewDecoder(resp.Body).Decode(network); err != nil {
		log.Error().Err(err).Msg("failed to decode network json")
		return nil, err
	}

	return network, nil
}

func (s *httpTNoDB) GetNetworksVersion(nodeID identity.Identifier) (map[modules.NetID]uint32, error) {
	url := fmt.Sprintf("%s/networks/%s/versions", s.baseURL, nodeID.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	versions := make(map[modules.NetID]uint32)
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}

func linkAddrs(l netlink.Link) ([]string, error) {
	addrs, err := netlink.AddrList(l, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}
	output := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		output = append(output, addr.IPNet.String())
	}
	return output, nil
}
