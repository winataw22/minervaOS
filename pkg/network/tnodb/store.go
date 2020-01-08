package tnodb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network/types"
)

type httpTNoDB struct {
	baseURL string
}

// NewHTTPTNoDB create an a client to a TNoDB reachable over HTTP
func NewHTTPTNoDB(url string) network.TNoDB {
	return &httpTNoDB{baseURL: url}
}

func (s *httpTNoDB) RegisterAllocation(farm pkg.Identifier, allocation *net.IPNet) error {
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

func (s *httpTNoDB) RequestAllocation(farm pkg.Identifier) (*net.IPNet, *net.IPNet, uint8, error) {
	url := fmt.Sprintf("%s/%s/%s", s.baseURL, "allocations", farm.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v", string(b))
		return nil, nil, 0, fmt.Errorf("wrong response status code received: %v", resp.Status)
	}

	data := struct {
		Alloc      string `json:"allocation"`
		FarmAlloc  string `json:"farm_alloc"`
		ExitNodeNr uint8  `json:"exit_node_nr"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil, 0, err
	}

	_, alloc, err := net.ParseCIDR(data.Alloc)
	if err != nil {
		return nil, nil, 0, errors.Wrap(err, "failed to parse network allocation")
	}
	_, farmAlloc, err := net.ParseCIDR(data.FarmAlloc)
	if err != nil {
		return nil, nil, 0, errors.Wrap(err, "failed to parse farm allocation")
	}

	return alloc, farmAlloc, data.ExitNodeNr, nil
}

func (s *httpTNoDB) GetFarm(farm pkg.Identifier) (*network.Farm, error) {
	var f network.Farm

	url := fmt.Sprintf("%s/farms/%s", s.baseURL, farm.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&f)

	return &f, err
}

func (s *httpTNoDB) GetNode(nodeID pkg.Identifier) (*types.Node, error) {

	url := fmt.Sprintf("%s/nodes/%s", s.baseURL, nodeID.Identity())

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("node %s not found", nodeID.Identity())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong response status: %v", resp.Status)
	}

	node := directory.TfgridNode2{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, err
	}

	return types.NewNodeFromSchema(node), nil
}

func (s *httpTNoDB) PublishInterfaces(local pkg.Identifier, ifaces []types.IfaceInfo) error {
	url := fmt.Sprintf("%s/nodes/%s/interfaces", s.baseURL, local.Identity())
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(ifaces); err != nil {
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

func (s *httpTNoDB) PublishWGPort(nodeID pkg.Identifier, ports []uint) error {
	url := fmt.Sprintf("%s/nodes/%s/ports", s.baseURL, nodeID.Identity())

	output := struct {
		Ports []uint `json:"ports"`
	}{
		Ports: ports,
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(output); err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong response status received: %s", resp.Status)
	}

	return nil
}

func (s *httpTNoDB) SetPublicIface(node pkg.Identifier, pub *types.PubIface) error {
	url := fmt.Sprintf("%s/nodes/%s/configure_public", s.baseURL, node.Identity())

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(pub); err != nil {
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

func (s *httpTNoDB) SelectExitNode(node pkg.Identifier) error {
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

func (s *httpTNoDB) GetPubIface(node pkg.Identifier) (*types.PubIface, error) {

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

	iface := struct {
		PublicConfig *directory.TfgridNodePublicIface1 `json:"public_config,omitempty"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&iface); err != nil {
		return nil, err
	}

	if iface.PublicConfig == nil {
		return nil, network.ErrNoPubIface
	}

	return &types.PubIface{
		Master: iface.PublicConfig.Master,
		// Vlan:   iface.PublicConfig.Vlan, // doesn't seems to exists in gedis types
		Type:    types.IfaceType(iface.PublicConfig.Type.String()),
		GW4:     iface.PublicConfig.Gw4,
		GW6:     iface.PublicConfig.Gw6,
		IPv4:    types.NewIPNetFromSchema(iface.PublicConfig.Ipv4),
		IPv6:    types.NewIPNetFromSchema(iface.PublicConfig.Ipv6),
		Version: int(iface.PublicConfig.Version),
	}, nil
}

func (s *httpTNoDB) GetNetwork(netid pkg.NetID) (*pkg.Network, error) {
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

	network := &pkg.Network{}
	if err := json.NewDecoder(resp.Body).Decode(network); err != nil {
		return nil, err
	}
	return network, nil
}

func (s *httpTNoDB) GetNetworksVersion(nodeID pkg.Identifier) (map[pkg.NetID]uint32, error) {
	url := fmt.Sprintf("%s/networks/%s/versions", s.baseURL, nodeID.Identity())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	versions := make(map[pkg.NetID]uint32)
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}
