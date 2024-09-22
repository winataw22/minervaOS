package mycelium

import (
	"context"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oasisprotocol/curve25519-voi/primitives/x25519"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	tunName      = "my0"
	myceliumBin  = "mycelium"
	zinitService = "mycelium"
	confPath     = "/tmp/mycelium_priv_key.bin"
	MyListenTCP  = 9651
)

// MyceliumServer represent a mycelium server
type MyceliumServer struct {
	cfg *NodeConfig
	ns  string
}

type MyceliumInspection struct {
	PublicKey string `json:"publicKey"`
	Address   net.IP `json:"address"`
}

// NewMyceliumServer create a new mycelium Server
func NewMyceliumServer(cfg *NodeConfig) *MyceliumServer {
	return &MyceliumServer{
		cfg: cfg,
	}
}

func (s *MyceliumServer) Restart(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGTERM)
}

func (s *MyceliumServer) Reload(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGHUP)
}

// Start creates a mycelium zinit service and starts it
func (s *MyceliumServer) Ensure(z *zinit.Client, ns string) error {
	if !namespace.Exists(ns) {
		return fmt.Errorf("invalid namespace '%s'", ns)
	}
	s.ns = ns

	if err := writeKey(confPath, s.cfg.privateKey); err != nil {
		return err
	}

	// service found.
	// better if we just stop, forget and start over to make
	// sure we using the right exec params
	if _, err := z.Status(zinitService); err == nil {
		// not here we need to stop it
		if err := z.StopWait(5*time.Second, zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to stop mycelium service")
		}
		if err := z.Forget(zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to forget mycelium service")
		}
	}

	bin, err := exec.LookPath(myceliumBin)
	if err != nil {
		return err
	}

	args := []string{
		"ip", "netns", "exec", ns,
		bin,
		"--key-file", confPath,
		"--tun-name", s.cfg.TunName,
		"--peers",
	}

	args = append(args, s.cfg.Peers...)

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: strings.Join(args, " "),
		After: []string{
			"node-ready",
		},
	})
	if err != nil {
		return err
	}

	if err := z.Monitor(zinitService); err != nil && !errors.Is(err, zinit.ErrAlreadyMonitored) {
		return err
	}

	return z.StartWait(time.Second*20, zinitService)
}

func EnsureMycelium(ctx context.Context, privateKey ed25519.PrivateKey, ns MyceliumNamespace) (*MyceliumServer, error) {
	// Filter out all the nodes from the same
	// segment so we do not just connect locally
	ips, err := ns.GetIPs() // returns ipv6 only
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ndmz public ipv6")
	}

	var ranges Ranges
	for _, ip := range ips {
		if ip.IP.IsGlobalUnicast() {
			ranges = append(ranges, ip)
		}
	}

	log.Info().Msgf("filtering out peers from ranges: %s", ranges)
	filter := Exclude(ranges)
	z := zinit.Default()

	cfg := GenerateConfig(privateKey)
	if err := cfg.FindPeers(ctx, filter); err != nil {
		return nil, err
	}

	server := NewMyceliumServer(&cfg)
	if err := server.Ensure(z, ns.Name()); err != nil {
		return nil, err
	}

	myInspec, err := server.InspectMycelium()
	if err != nil {
		return nil, err
	}

	gw, err := myInspec.Gateway()
	if err != nil {
		return nil, errors.Wrap(err, "fail read mycelium subnet")
	}

	if err := ns.SetMyIP(gw, nil); err != nil {
		return nil, errors.Wrap(err, "fail to configure mycelium subnet gateway IP")
	}

	return server, nil
}

func (s *MyceliumServer) InspectMycelium() (inspection MyceliumInspection, err error) {
	// we check if the file exists before we do inspect because mycelium will create a random seed
	// file if file does not exist
	_, err = os.Stat(confPath)
	if err != nil {
		return inspection, err
	}

	bin, err := exec.LookPath(myceliumBin)
	if err != nil {
		return inspection, err
	}

	output, err := exec.Command("ip", "netns", "exec", s.ns, bin, "inspect", "--json", "--key-file", confPath).Output()
	if err != nil {
		return inspection, errors.Wrap(err, "failed to inspect mycelium ip")
	}

	if err := json.Unmarshal(output, &inspection); err != nil {
		return inspection, errors.Wrap(err, "failed to load mycelium information from key")
	}

	return
}

// IP return the address in the 400::/7 subnet allocated by mycelium
func (m *MyceliumInspection) IP() net.IP {
	return m.Address
}

// Subnet return the 400::/64 subnet allocated by mycelium
func (m *MyceliumInspection) Subnet() (subnet net.IPNet, err error) {
	ipv6 := m.Address.To16()
	if ipv6 == nil {
		return subnet, errors.Errorf("invalid mycelium ip")
	}

	ip := make(net.IP, net.IPv6len)
	copy(ip[0:8], ipv6[0:8])

	subnet = net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}

	return
}

// Gateway derive the gateway IP from the mycelium IP in the /64 range.
func (m *MyceliumInspection) Gateway() (gw net.IPNet, err error) {
	subnet, err := m.Subnet()
	if err != nil {
		return gw, err
	}

	ip := subnet.IP
	ip[net.IPv6len-1] = 1

	gw = net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}

	return
}

// Tun return the name of the TUN interface created by mycelium
func (s *MyceliumServer) Tun() string {
	return s.cfg.TunName
}

// IPFor return an IP address out of the node allocated subnet by hasing b and using it
// to generate the last 64 bits of the IPV6 address
func (m *MyceliumInspection) IPFor(b []byte) (net.IPNet, error) {
	subnet, err := m.Subnet()
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

func writeKey(path string, privateKey x25519.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return err
	}

	return os.WriteFile(path, privateKey[:], 0666)
}
