package network

import (
	"fmt"
	"net"
	"path/filepath"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
)

type networker struct {
	storageDir  string
	netResAlloc NetResourceAllocator
}

func NewNetworker(storageDir string, allocator NetResourceAllocator) modules.Networker {
	return &networker{
		storageDir:  storageDir,
		netResAlloc: allocator,
	}
}

var _ modules.Networker = (*networker)(nil)

func (n *networker) GetNetResource(id string) (modules.NetResource, error) {
	// TODO check signature
	return n.netResAlloc.Get(id)
}

func (n *networker) ApplyNetResource(netID modules.NetID, resource modules.NetResource) error {
	return applyNetResource(n.storageDir, netID, resource)
}

func applyNetResource(storageDir string, netID modules.NetID, netRes modules.NetResource) error {
	network := string(netID)
	if err := createNetwork(network); err != nil {
		return err
	}

	storage := filepath.Join(storageDir, network)
	if err := configureWG(storage, network, netRes); err != nil {
		return err
	}
	return nil
}

// createNetwork creates a network namespace and a bridge
// and a wireguard interface and then move it interface inside
// the net namespace
func createNetwork(name string) error {
	var (
		netnsName  = netnsName(name)
		bridgeName = bridgeName(name)
		wgName     = wgName(name)
	)

	if !namespace.Exists(netnsName) {
		log.Info().Str("name", netnsName).Msg("create network namespace")
		_, err := namespace.Create(netnsName)
		if err != nil {
			return err
		}

		log.Info().Str("name", wgName).Msg("create wireguard interface")
		wg, err := wireguard.New(wgName)
		if err != nil {
			return err
		}

		log.Info().
			Str("wg", wgName).
			Str("namespace", netnsName).
			Msg("move wireguard into network namespace")
		if err := namespace.SetLink(wg, netnsName); err != nil {
			return err
		}
	}

	if !bridge.Exists(bridgeName) {
		if _, err := bridge.New(bridgeName); err != nil {
			return err
		}
	}

	return nil
}

func deleteNetwork(network string) error {
	if err := bridge.Delete(bridgeName(network)); err != nil {
		return err
	}
	return namespace.Delete(netnsName(network))
}

func configureWG(storageDir, network string, netRes modules.NetResource) error {
	var (
		netnsName   = netnsName(network)
		wgName      = wgName(network)
		storagePath = filepath.Join(storageDir, network)
		key         wgtypes.Key
		err         error
	)

	key, err = wireguard.LoadKey(storagePath)
	if err != nil {
		key, err = wireguard.GenerateKey(storagePath)
		if err != nil {
			return err
		}
	}

	// enter container net ns
	nsCtx := namespace.NSContext{}
	if err := nsCtx.Enter(netnsName); err != nil {
		return err
	}

	wg, err := wireguard.GetByName(wgName)
	if err != nil {
		return err
	}

	// configure wg iface
	peers := make([]wireguard.Peer, len(netRes.Connected))
	for i, connected := range netRes.Connected {
		if connected.Type != modules.ConnTypeWireguard {
			continue
		}

		conn := connected.Connection

		var endpoint string
		if conn.Peer.To16() != nil {
			endpoint = fmt.Sprintf("[%s]:%d", conn.Peer.String(), conn.PeerPort)
		} else {
			endpoint = fmt.Sprintf("%s:%d", conn.Peer.String(), conn.PeerPort)
		}

		peers[i] = wireguard.Peer{
			PublicKey:  conn.Key,
			Endpoint:   endpoint,
			AllowedIPs: []string{"0.0.0.0/0"},
		}
	}

	wgIPNet, err := wgIP(netRes.Prefix)
	if err != nil {
		return err
	}

	log.Info().Msg("configure wireguard interface")

	err = wg.Configure(wgIPNet.String(), key.String(), peers)
	if err != nil {
		nsCtx.Exit()
		return err
	}

	// exit containe net ns
	if err := nsCtx.Exit(); err != nil {
		return err
	}

	return nil
}

func bridgeName(name string) string {
	return fmt.Sprintf("br%s", name)
}
func wgName(name string) string {
	return fmt.Sprintf("wg%s", name)
}
func netnsName(name string) string {
	return fmt.Sprintf("ns%s", name)
}

func wgIP(prefix net.IPNet) (*net.IPNet, error) {
	prefixIP := []byte(prefix.IP.To16())
	id := prefixIP[6:8]
	_, ipnet, err := net.ParseCIDR(fmt.Sprintf("fe80::%x/64", id))
	if err != nil {
		return nil, err
	}
	return ipnet, nil
}

type NetResourceAllocator interface {
	Get(txID string) (modules.NetResource, error)
}

// type httpNetResourceAllocator struct {
// 	baseURL string
// }

// func NewHTTPNetResourceAllocator(url string) *httpNetResourceAllocator {
// 	return &httpNetResourceAllocator{url}
// }

// func (a *httpNetResourceAllocator) Get(txID string) (modules.NetResource, error) {
// 	netRes := modules.NetResource{}

// 	url, err := joinURL(a.baseURL, txID)

// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return netRes, err
// 	}
// 	defer resp.Body.Close()

// 	if err := json.NewDecoder(resp.Body).Decode(&netRes); err != nil {
// 		return netRes, err
// 	}

// 	return netRes, nil
// }

// func joinURL(base, path string) (string, error) {
// 	u, err := url.Parse(base)
// 	if err != nil {
// 		return "nil", err
// 	}
// 	u.Path = filepath.Join(u.Path, path)
// 	return u.String(), nil
// }

type TestNetResourceAllocator struct {
	Resource modules.NetResource
}

func NewTestNetResourceAllocator() NetResourceAllocator {
	return &TestNetResourceAllocator{
		Resource: modules.NetResource{
			NodeID: modules.NodeID("supernode"),
			Prefix: MustParseCIDR("2a02:1802:5e:f002::/64"),
			Connected: []modules.Connected{
				{
					Type:   modules.ConnTypeWireguard,
					Prefix: MustParseCIDR("2a02:1802:5e:cc02::/64"),
					Connection: modules.Wireguard{
						NICName: "cc02",
						Peer:    net.ParseIP("2001::1"),
						Key:     "",
						// LinkLocal: net.
					},
				},
			},
		},
	}
}

func (a *TestNetResourceAllocator) Get(txID string) (modules.NetResource, error) {
	return a.Resource, nil
}

func MustParseCIDR(cidr string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return *ipnet
}
