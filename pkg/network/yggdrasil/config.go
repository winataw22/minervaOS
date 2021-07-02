package yggdrasil

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/jbenet/go-base58"

	zosCrypto "github.com/threefoldtech/zos/pkg/crypto"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/crypto"
)

// List of port used by yggdrasil
const (
	YggListenTCP       = 9943
	YggListenTLS       = 9944
	YggListenLinkLocal = 9945

	YggIface = "ygg0"
)

// NodeConfig type
type NodeConfig config.NodeConfig

// NodeID returns the yggdrasil node ID of s
func (s *NodeConfig) NodeID() (*crypto.NodeID, error) {
	if s.EncryptionPublicKey == "" {
		panic("EncryptionPublicKey empty")
	}

	pubkey, err := hex.DecodeString(s.EncryptionPublicKey)
	if err != nil {
		return nil, err
	}

	var box crypto.BoxPubKey
	copy(box[:], pubkey[:])
	return crypto.GetNodeID(&box), nil
}

// Address return the address in the 200::/7 subnet allocated by yggdrasil
func (s *NodeConfig) Address() (net.IP, error) {
	nodeID, err := s.NodeID()
	if err != nil {
		return nil, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, address.AddrForNodeID(nodeID)[:])

	return ip, nil
}

// GenerateConfig creates a new yggdrasil configuration and generate the
// box and signing key from the ed25519 Private key of the node
// this creates a mapping between a yggdrasil identity and the TFGrid identity
func GenerateConfig(privateKey ed25519.PrivateKey) NodeConfig {
	cfg := config.GenerateConfig()

	if privateKey != nil {
		cfg.SigningPrivateKey = hex.EncodeToString(privateKey)

		signingPublicKey := privateKey.Public().(ed25519.PublicKey)
		cfg.SigningPublicKey = hex.EncodeToString(signingPublicKey)

		encryptionPrivateKey := zosCrypto.PrivateKeyToCurve25519(privateKey)
		cfg.EncryptionPrivateKey = hex.EncodeToString(encryptionPrivateKey[:])

		encryptionPublicKey := zosCrypto.PublicKeyToCurve25519(signingPublicKey)
		cfg.EncryptionPublicKey = hex.EncodeToString(encryptionPublicKey[:])

		cfg.NodeInfo = map[string]interface{}{
			"name": base58.Encode(signingPublicKey)[:6],
		}
	}
	cfg.MulticastInterfaces = []string{"npub6", "npub4"}
	cfg.LinkLocalTCPPort = YggListenLinkLocal

	cfg.IfName = YggIface
	cfg.TunnelRouting.Enable = true
	cfg.SessionFirewall.Enable = false

	cfg.Listen = []string{
		fmt.Sprintf("tcp://[::]:%d", YggListenTCP),
		fmt.Sprintf("tls://[::]:%d", YggListenTLS),
	}

	return NodeConfig(*cfg)
}
