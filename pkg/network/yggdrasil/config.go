package yggdrasil

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/jbenet/go-base58"

	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

// List of port used by yggdrasil
const (
	YggListenTCP       = 9943
	YggListenTLS       = 9944
	YggListenLinkLocal = 9945

	YggIface = "ygg0"
)

// GenerateConfig creates a new yggdrasil configuration and generate the
// box and signing key from the ed25519 Private key of the node
// this creates a mapping between a yggdrasil identity and the TFGrid identity
func GenerateConfig(privateKey ed25519.PrivateKey) config.NodeConfig {
	cfg := config.GenerateConfig()

	if privateKey != nil {
		cfg.SigningPrivateKey = hex.EncodeToString(privateKey)

		signingPublicKey := privateKey.Public().(ed25519.PublicKey)
		cfg.SigningPublicKey = hex.EncodeToString(signingPublicKey)

		encryptionPrivateKey := crypto.PrivateKeyToCurve25519(privateKey)
		cfg.EncryptionPrivateKey = hex.EncodeToString(encryptionPrivateKey[:])

		encryptionPublicKey := crypto.PublicKeyToCurve25519(signingPublicKey)
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

	return *cfg
}
