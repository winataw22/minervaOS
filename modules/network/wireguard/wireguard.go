package wireguard

import (
	"net"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Wireguard is a netlink.Link of type wireguard
type Wireguard struct {
	attrs *netlink.LinkAttrs
}

func New(name string) (*Wireguard, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	attrs.MTU = 1420

	wg := &Wireguard{attrs: &attrs}
	if err := netlink.LinkAdd(wg); err != nil {
		return nil, err
	}
	return wg, nil
}

// Type implements the netlink.Link interface
func (w *Wireguard) Type() string {
	return "wireguard"
}

// Attrs implements the netlink.Link interface
func (w *Wireguard) Attrs() *netlink.LinkAttrs {
	return w.attrs
}

func (w *Wireguard) SetAddr(cidr string) error {
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		return err
	}

	if err := netlink.AddrAdd(w, addr); err != nil {
		return err
	}
	return nil
}

type Peer struct {
	PublicKey  string
	Endpoint   string
	AllowedIPs []string
}

func (w *Wireguard) Configure(addr, privateKey string, peers []Peer) error {

	if err := netlink.LinkSetDown(w); err != nil {
		return err
	}

	if err := w.SetAddr(addr); err != nil {
		return err
	}

	wc, err := wgctrl.New()
	if err != nil {
		return err
	}
	defer wc.Close()

	peersConfig := make([]wgtypes.PeerConfig, len(peers))
	for i, peer := range peers {
		p, err := newPeer(peer.PublicKey, peer.Endpoint, peer.AllowedIPs)
		if err != nil {
			return err
		}
		peersConfig[i] = p
	}

	key, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return err
	}

	config := wgtypes.Config{
		PrivateKey: &key,
		Peers:      peersConfig,
	}
	log.Info().Msg("configure wg device")

	if err := wc.ConfigureDevice(w.attrs.Name, config); err != nil {
		return err
	}

	return netlink.LinkSetUp(w)
}

func newPeer(pubkey, endpoint string, allowedIPs []string) (wgtypes.PeerConfig, error) {
	peer := wgtypes.PeerConfig{}
	var err error

	duration := time.Second * 10
	peer.PersistentKeepaliveInterval = &duration

	peer.PublicKey, err = wgtypes.ParseKey(pubkey)
	if err != nil {
		return peer, err
	}

	host, p, err := net.SplitHostPort(endpoint)
	if err != nil {
		return peer, err
	}

	port, err := strconv.Atoi(p)
	if err != nil {
		return peer, err
	}

	peer.Endpoint = &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	if err != nil {
		return peer, err
	}
	for _, allowedIP := range allowedIPs {
		_, ip, err := net.ParseCIDR(allowedIP)
		if err != nil {
			return peer, err
		}
		peer.AllowedIPs = append(peer.AllowedIPs, *ip)
	}

	return peer, nil
}
