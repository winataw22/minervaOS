package yggdrasil

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

const (
	zinitService = "yggdrasil"
	confPath     = "/var/cache/modules/networkd/yggdrasil.conf"
)

// YggServer represent a yggdrasil server
type YggServer struct {
	cfg *NodeConfig
}

// NewYggServer create a new yggdrasil Server
func NewYggServer(cfg *NodeConfig) *YggServer {
	return &YggServer{
		cfg: cfg,
	}
}

func (s *YggServer) Restart(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGTERM)
}

func (s *YggServer) Reload(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGHUP)
}

// Start creates an yggdrasil zinit service and starts it
func (s *YggServer) Ensure(z *zinit.Client, ns string) error {
	if !namespace.Exists(ns) {
		return fmt.Errorf("invalid namespace '%s'", ns)
	}

	if err := writeConfig(confPath, s.cfg); err != nil {
		return err
	}

	// service found.
	// better if we just stop, forget and start over to make
	// sure we using the right exec params
	if _, err := z.Status(zinitService); err == nil {
		// not here we need to stop it
		if err := z.StopWait(5*time.Second, zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to stop yggdrasil service")
		}
		if err := z.Forget(zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to forget yggdrasil service")
		}
	}

	bin, err := exec.LookPath("yggdrasil")
	if err != nil {
		return err
	}

	cmd := `sh -c '
		ulimit -n 16384

		exec ip netns exec %s %s -loglevel error -useconffile %s
	'`

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: fmt.Sprintf(cmd, ns, bin, confPath),
		After: []string{
			"node-ready",
		},
		Test: "yggdrasilctl getself | grep -i coords",
	})

	if err != nil {
		return err
	}

	if err := z.Monitor(zinitService); err != nil && !errors.Is(err, zinit.ErrAlreadyMonitored) {
		return err
	}

	return z.StartWait(time.Second*20, zinitService)
}

// NodeID returns the yggdrasil node ID of s
func (s *YggServer) NodeID() (ed25519.PublicKey, error) {
	if s.cfg.PublicKey == "" {
		panic("EncryptionPublicKey empty")
	}

	return hex.DecodeString(s.cfg.PublicKey)
}

// Address return the address in the 200::/7 subnet allocated by yggdrasil
func (s *YggServer) Address() (net.IP, error) {
	nodeID, err := s.NodeID()
	if err != nil {
		return nil, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, address.AddrForKey(nodeID)[:])

	return ip, nil
}

// Subnet return the 300::/64 subnet allocated by yggdrasil
func (s *YggServer) Subnet() (net.IPNet, error) {
	nodeID, err := s.NodeID()
	if err != nil {
		return net.IPNet{}, err
	}

	snet := *address.SubnetForKey(nodeID)
	ipnet := net.IPNet{
		IP:   append(snet[:], 0, 0, 0, 0, 0, 0, 0, 0),
		Mask: net.CIDRMask(len(snet)*8, 128),
	}

	return ipnet, nil
}

// Gateway return the first IP of the 300::/64 subnet allocated by yggdrasil
func (s *YggServer) Gateway() (net.IPNet, error) {

	subnet, err := s.Subnet()
	if err != nil {
		return net.IPNet{}, err
	}
	subnet.IP[len(subnet.IP)-1] = 0x1

	return subnet, nil
}

// Tun return the name of the TUN interface created by yggdrasil
func (s *YggServer) Tun() string {
	return s.cfg.IfName
}

// SubnetFor return an IP address out of the node allocated subnet by hasing b and using it
// to generate the last 64 bits of the IPV6 address
func (s *YggServer) SubnetFor(b []byte) (net.IPNet, error) {
	subnet, err := s.Subnet()
	if err != nil {
		return net.IPNet{}, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, subnet.IP)

	subIP, err := subnetFor(ip, b)
	if err != nil {
		return net.IPNet{}, err
	}

	return net.IPNet{
		IP:   subIP,
		Mask: net.CIDRMask(64, 128),
	}, nil
}

func subnetFor(prefix net.IP, b []byte) (net.IP, error) {
	h := sha512.New()
	if _, err := h.Write(b); err != nil {
		return nil, err
	}
	digest := h.Sum(nil)
	copy(prefix[8:], digest[:8])
	return prefix, nil
}

func writeConfig(path string, cfg *NodeConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(cfg)
}
