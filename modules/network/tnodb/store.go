package tnodb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network"

	"github.com/rs/zerolog/log"

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

func (s *httpTNoDB) RegisterAllocation(farm modules.Identifier, allocation *net.IPNet) error {
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

func (s *httpTNoDB) RequestAllocation(farm modules.Identifier) (*net.IPNet, *net.IPNet, error) {
	url := fmt.Sprintf("%s/%s/%s", s.baseURL, "allocations", farm.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v", string(b))
		return nil, nil, fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	data := struct {
		Alloc     string `json:"allocation"`
		FarmAlloc string `json:"farm_allocation"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil, err
	}

	_, alloc, err := net.ParseCIDR(data.Alloc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse network allocation")
	}
	_, farmAlloc, err := net.ParseCIDR(data.FarmAlloc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse farm allocation")
	}

	return alloc, farmAlloc, nil
}

func (s *httpTNoDB) GetFarm(farm modules.Identifier) (network.Farm, error) {
	f := network.Farm{}

	url := fmt.Sprintf("%s/farms/%s", s.baseURL, farm.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return f, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&f)

	return f, err
}

func (s *httpTNoDB) GetNode(nodeID modules.Identifier) (*network.Node, error) {

	url := fmt.Sprintf("%s/nodes/%s", s.baseURL, nodeID.Identity())

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("node %s node found", nodeID.Identity())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong response status: %v", resp.Status)
	}

	node := &network.Node{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, err
	}

	return node, nil
}

func (s *httpTNoDB) PublishInterfaces(local modules.Identifier) error {
	output := []*network.IfaceInfo{}

	links, err := netlink.LinkList()
	if err != nil {
		log.Error().Err(err).Msgf("failed to list interfaces")
		return err
	}

	for _, link := range ifaceutil.LinkFilter(links, []string{"device", "bridge"}) {

		// TODO: see if we need to set the if down
		if err := netlink.LinkSetUp(link); err != nil {
			log.Info().Str("interface", link.Attrs().Name).Msg("failed to bring interface up")
			continue
		}

		if !ifaceutil.IsVirtEth(link.Attrs().Name) && !ifaceutil.IsPluggedTimeout(link.Attrs().Name, time.Second*5) {
			log.Info().Str("interface", link.Attrs().Name).Msg("interface is not plugged in, skipping")
			continue
		}

		_, gw, err := ifaceutil.HasDefaultGW(link)
		if err != nil {
			return err
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return err
		}

		info := &network.IfaceInfo{
			Name:  link.Attrs().Name,
			Addrs: make([]*net.IPNet, len(addrs)),
		}
		for i, addr := range addrs {
			info.Addrs[i] = addr.IPNet
		}

		if gw != nil {
			info.Gateway = append(info.Gateway, gw)
		}

		output = append(output, info)
	}

	url := fmt.Sprintf("%s/nodes/%s/interfaces", s.baseURL, local.Identity())
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

func (s *httpTNoDB) ConfigurePublicIface(node modules.Identifier, ips []*net.IPNet, gws []net.IP, iface string) error {
	output := struct {
		Iface string            `json:"iface"`
		IPs   []string          `json:"ips"`
		GWs   []string          `json:"gateways"`
		Type  network.IfaceType `json:"iface_type"`
	}{
		Iface: iface,
		IPs:   make([]string, len(ips)),
		GWs:   make([]string, len(gws)),
		Type:  network.MacVlanIface, //TODO: allow to chose type of connection
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

func (s *httpTNoDB) SelectExitNode(node modules.Identifier) error {
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

func (s *httpTNoDB) ReadPubIface(node modules.Identifier) (*network.PubIface, error) {

	iface := &struct {
		PublicConfig *network.PubIface `json:"public_config"`
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

	if err := json.NewDecoder(resp.Body).Decode(&iface); err != nil {
		return nil, err
	}

	if iface.PublicConfig == nil {
		return nil, network.ErrNoPubIface
	}

	return iface.PublicConfig, nil
}

func (s *httpTNoDB) GetNetwork(netid modules.NetID) (*modules.Network, error) {
	url := fmt.Sprintf("%s/networks/%s", s.baseURL, string(netid))
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("network not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong response status code %s", resp.Status)
	}

	network := &modules.Network{}
	if err := json.NewDecoder(resp.Body).Decode(network); err != nil {
		return nil, err
	}
	return network, nil
}

func (s *httpTNoDB) GetNetworksVersion(nodeID modules.Identifier) (map[modules.NetID]uint32, error) {
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
