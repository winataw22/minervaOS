package yggdrasil

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"math"
	"net"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/latency"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

// List of port used by yggdrasil
const (
	YggListenTCP       = 9943
	YggListenTLS       = 9944
	YggListenLinkLocal = 9945

	YggIface = "ygg0"
)

// NodeConfig wrapper around yggdrasil node config
type NodeConfig config.NodeConfig

// Address gets the address from the config
func (n *NodeConfig) Address() (net.IP, error) {
	ip := make([]byte, net.IPv6len)
	pk, err := hex.DecodeString(n.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load public key")
	}
	copy(ip, address.AddrForKey(pk)[:])

	return ip, nil
}

func (n *NodeConfig) FindPeers(ctx context.Context, filter ...Filter) error {
	// fetching a peer list goes as this
	// - Always include the list of peers from
	zos, err := fetchZosYggList()
	if err != nil {
		return errors.Wrap(err, "failed to get zos public peer list")
	}

	zos, err = zos.Ups(filter...)
	if err != nil {
		return errors.Wrap(err, "failed to filter out peer list")
	}

	pub := fetchPubYggList()
	pub, err = pub.Ups(filter...)
	if err != nil {
		return errors.Wrap(err, "failed to get peers list")
	}

	log.Info().Int("count", len(pub)).Msg("found yggdrasil up peers")
	endpoints := make([]string, len(pub))
	for i, p := range pub {
		endpoints[i] = p.Endpoint
	}

	ls := latency.NewSorter(endpoints, 5)
	results := ls.Run(ctx)
	if len(results) == 0 {
		return fmt.Errorf("cannot find public yggdrasil peer to connect to")
	}

	// select the best 3 public peers
	var peers []string
	for _, peer := range zos {
		peers = append(peers, peer.Endpoint)
	}

	// take max of 3 from the results list
	to := math.Min(3, float64(len(results)))

	for i := 0; i < int(to); i++ {
		peers = append(peers, results[i].Endpoint)
		log.Info().Str("endpoint", results[i].Endpoint).Msg("yggdrasill public peer selected")
	}

	n.Peers = peers
	return nil
}

// GenerateConfig creates a new yggdrasil configuration and generate the
// box and signing key from the ed25519 Private key of the node
// this creates a mapping between a yggdrasil identity and the TFGrid identity
func GenerateConfig(privateKey ed25519.PrivateKey) (cfg NodeConfig) {
	cfg.IfMTU = 65535
	if privateKey != nil {
		cfg.PrivateKey = hex.EncodeToString(privateKey)

		signingPublicKey := privateKey.Public().(ed25519.PublicKey)
		cfg.PublicKey = hex.EncodeToString(signingPublicKey)

		cfg.NodeInfo = map[string]interface{}{
			"name": base58.Encode(signingPublicKey)[:6],
		}
	}

	cfg.MulticastInterfaces = []config.MulticastInterfaceConfig{
		{
			Regex:  ".*",
			Listen: true,
			Beacon: true,
			Port:   0,
		},
	}

	cfg.IfName = YggIface

	cfg.Listen = []string{
		fmt.Sprintf("tcp://[::]:%d", YggListenTCP),
		fmt.Sprintf("tls://[::]:%d", YggListenTLS),
	}

	return
}
