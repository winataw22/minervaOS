package wireguard

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Wireguard is a netlink.Link of type wireguard
type Wireguard struct {
	attrs *netlink.LinkAttrs
}

// New create a new wireguard interface
func New(name string) (*Wireguard, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	attrs.MTU = 1420

	wg := &Wireguard{attrs: &attrs}
	if err := netlink.LinkAdd(wg); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return wg, nil
}

// GetByName return a wireguard object by its name
func GetByName(name string) (*Wireguard, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	if link.Type() != "wireguard" {
		return nil, fmt.Errorf("link %s is not of type wireguard", name)
	}
	wg := &Wireguard{
		attrs: link.Attrs(),
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

// Device returns the detail of the configuration of the
// wireguard interface
func (w *Wireguard) Device() (*wgtypes.Device, error) {
	wg, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	defer wg.Close()

	return wg.Device(w.attrs.Name)
}

// SetAddr sets an IP address on the interface
func (w *Wireguard) SetAddr(cidr string) error {
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		return err
	}

	if err := netlink.AddrAdd(w, addr); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// UnsetAddr removes an IP address from the interface
func (w *Wireguard) UnsetAddr(cidr string) error {
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		return err
	}

	if err := netlink.AddrDel(w, addr); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Peer represent a peer in a wireguard configuration
type Peer struct {
	PublicKey  string
	Endpoint   string
	AllowedIPs []string
}

// Configure configures the wiregard configuration
func (w *Wireguard) Configure(privateKey string, listentPort int, peers []*Peer) error {

	if err := netlink.LinkSetDown(w); err != nil {
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

	for _, peer := range peersConfig {
		fmt.Printf("%+v\n", peer)
	}

	config := wgtypes.Config{
		PrivateKey:   &key,
		Peers:        peersConfig,
		ListenPort:   &listentPort,
		ReplacePeers: true,
	}
	log.Info().Msg("configure wg device")

	if err := wc.ConfigureDevice(w.attrs.Name, config); err != nil {
		return errors.Wrap(err, "failed to configure wireguard interface")
	}

	if err := netlink.LinkSetUp(w); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "failed to bring wireguard interface %s up", w.Attrs().Name)
	}
	return nil
}

func newPeer(pubkey, endpoint string, allowedIPs []string) (wgtypes.PeerConfig, error) {
	peer := wgtypes.PeerConfig{
		ReplaceAllowedIPs: true,
	}
	var err error

	duration := time.Second * 20
	peer.PersistentKeepaliveInterval = &duration

	peer.PublicKey, err = wgtypes.ParseKey(pubkey)
	if err != nil {
		return peer, err
	}

	if endpoint != "" {
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
	}

	for _, allowedIP := range allowedIPs {
		ip, ipNet, err := net.ParseCIDR(allowedIP)
		if err != nil {
			return peer, err
		}
		ipNet.IP = ip
		peer.AllowedIPs = append(peer.AllowedIPs, *ipNet)
	}

	return peer, nil
}

// GenerateKey generates a new private key. If key already exists
// in that location, that key is returned instead.
func GenerateKey(dir string) (wgtypes.Key, error) {
	path := filepath.Join(dir, "key.priv")
	data, err := ioutil.ReadFile(path)
	if err == nil {
		//key already exists
		return wgtypes.ParseKey(string(data))
	} else if !os.IsNotExist(err) {
		//another error than not exist
		return wgtypes.Key{}, err
	}

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return wgtypes.Key{}, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return wgtypes.Key{}, err
	}

	if err := ioutil.WriteFile(path, []byte(key.String()), 0400); err != nil {
		return wgtypes.Key{}, err
	}
	return key, nil
}

// LoadKey tries to read a private key from disk
func LoadKey(dir string) (wgtypes.Key, error) {
	path := filepath.Join(dir, "key.priv")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.ParseKey(string(b))
}
