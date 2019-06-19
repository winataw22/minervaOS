package network

import (
	"fmt"
	"net"
	"path/filepath"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"
	"github.com/vishvananda/netlink"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
)

type networker struct {
	storageDir  string
	netResAlloc NetResourceAllocator
}

// NewNetworker create a new modules.Networker that can be used with zbus
func NewNetworker(storageDir string, allocator NetResourceAllocator) modules.Networker {
	return &networker{
		storageDir:  storageDir,
		netResAlloc: allocator,
	}
}

var _ modules.Networker = (*networker)(nil)

// GetNetResource implements modules.Networker interface
func (n *networker) GetNetResource(id string) (modules.NetResource, error) {
	// TODO check signature
	return n.netResAlloc.Get(id)
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) ApplyNetResource(netID modules.NetID, resource modules.NetResource) error {
	return applyNetResource(n.storageDir, netID, resource)
}

func applyNetResource(storageDir string, netID modules.NetID, netRes modules.NetResource) error {
	if err := createNetworkResource(netID, netRes); err != nil {
		return err
	}

	if _, err := configureWG(storageDir, netRes); err != nil {
		return err
	}
	return nil
}

// createNetworkResource creates a network namespace and a bridge
// and a wireguard interface and then move it interface inside
// the net namespace
func createNetworkResource(netID modules.NetID, resource modules.NetResource) error {
	var (
		// prefix     = prefixStr(resource.Prefix)
		netnsName  = netnsName(resource.Prefix)
		bridgeName = bridgeName(resource.Prefix)
		wgName     = wgName(resource.Prefix)
		vethName   = vethName(resource.Prefix)
	)

	log.Info().Str("bridge", bridgeName).Msg("Create bridge")
	br, err := bridge.New(bridgeName)
	if err != nil {
		return err
	}

	log.Info().Str("namesapce", netnsName).Msg("Create namesapce")
	netns, err := namespace.Create(netnsName)
	if err != nil {
		return err
	}

	hostIface := &current.Interface{}
	var handler = func(hostNS ns.NetNS) error {
		log.Info().
			Str("namespace", netnsName).
			Str("veth", vethName).
			Msg("Create veth pair in net namespace")
		hostVeth, containerVeth, err := ip.SetupVeth(vethName, 1500, hostNS)
		if err != nil {
			return err
		}
		hostIface.Name = hostVeth.Name

		link, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}

		log.Info().Str("addr", resource.Prefix.String()).Msg("set address on veth interface")
		addr := &netlink.Addr{IPNet: &resource.Prefix, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil {
			return err
		}

		a, b := ipv4Nibble(resource.Prefix)
		ip, ipNet, err := net.ParseCIDR(fmt.Sprintf("10.%d.%d.1/24", a, b))
		if err != nil {
			return err
		}
		ipNet.IP = ip
		addr = &netlink.Addr{IPNet: ipNet, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil {
			return err
		}

		return nil
	}
	if err := netns.Do(handler); err != nil {
		return err
	}

	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return err
	}

	log.Info().
		Str("veth", vethName).
		Str("bridge", bridgeName).
		Msg("attach veth to bridge")
	if err := bridge.AttachNic(hostVeth, br); err != nil {
		return err
	}

	log.Info().Str("wg", wgName).Msg("create wireguard interface")
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

	return nil
}

func deleteNetworkResource(resource modules.NetResource) error {
	var (
		netnsName  = netnsName(resource.Prefix)
		bridgeName = bridgeName(resource.Prefix)
	)
	if err := bridge.Delete(bridgeName); err != nil {
		return err
	}
	return namespace.Delete(netnsName)
}

func configureWG(storageDir string, resource modules.NetResource) (wgtypes.Key, error) {
	var (
		netnsName   = netnsName(resource.Prefix)
		wgName      = wgName(resource.Prefix)
		storagePath = filepath.Join(storageDir, prefixStr(resource.Prefix))
		key         wgtypes.Key
		err         error
	)

	key, err = wireguard.LoadKey(storagePath)
	if err != nil {
		key, err = wireguard.GenerateKey(storagePath)
		if err != nil {
			return key, err
		}
	}

	// configure wg iface
	peers := make([]wireguard.Peer, len(resource.Connected))
	for i, peer := range resource.Connected {
		if peer.Type != modules.ConnTypeWireguard {
			continue
		}

		a, b := ipv4Nibble(peer.Prefix)
		peers[i] = wireguard.Peer{
			PublicKey: peer.Connection.Key,
			Endpoint:  endpoint(peer),
			AllowedIPs: []string{
				fmt.Sprintf("fe80::%s/128", prefixStr(peer.Prefix)),
				fmt.Sprintf("172.16.%d.%d/32", a, b),
			},
		}
	}

	netns, err := namespace.GetByName(netnsName)
	if err != nil {
		return key, err
	}

	var handler = func(_ ns.NetNS) error {

		wg, err := wireguard.GetByName(wgName)
		if err != nil {
			return err
		}

		log.Info().Msg("configure wireguard interface")
		if err = wg.Configure(resource.LinkLocal.String(), key.String(), peers); err != nil {
			return err
		}
		return nil
	}
	if err := netns.Do(handler); err != nil {
		return key, err
	}

	return key, nil
}

func endpoint(peer modules.Connected) string {
	var endpoint string
	if peer.Connection.IP.To16() != nil {
		endpoint = fmt.Sprintf("[%s]:%d", peer.Connection.IP.String(), peer.Connection.Port)
	} else {
		endpoint = fmt.Sprintf("%s:%d", peer.Connection.IP.String(), peer.Connection.Port)
	}
	return endpoint
}

func prefixStr(prefix net.IPNet) string {
	b := []byte(prefix.IP)[6:8]
	return fmt.Sprintf("%x", b)
}
func bridgeName(prefix net.IPNet) string {
	return fmt.Sprintf("br%s", prefixStr(prefix))
}
func wgName(prefix net.IPNet) string {
	return fmt.Sprintf("wg%s", prefixStr(prefix))
}
func netnsName(prefix net.IPNet) string {
	return fmt.Sprintf("ns%s", prefixStr(prefix))
}
func vethName(prefix net.IPNet) string {
	return fmt.Sprintf("veth%s", prefixStr(prefix))
}

func ipv4Nibble(prefix net.IPNet) (uint8, uint8) {
	x := []byte(prefix.IP)
	a := uint8(x[6])
	b := uint8(x[7])
	return a, b
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
