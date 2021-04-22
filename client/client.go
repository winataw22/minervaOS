package client

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	zoscrypt "github.com/threefoldtech/zos/pkg/crypto"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/crypto"
	"github.com/zaibon/httpsig"
)

// Client struct
type Client struct {
	id     uint32
	sk     ed25519.PrivateKey
	signer *httpsig.Signer
}

// NewClient creates a new instance of client
func NewClient(id uint32, seed string) (*Client, error) {
	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		return nil, err
	}

	if len(seedBytes) != ed25519.SeedSize {
		return nil, fmt.Errorf("invlaid seed, wrong seed size")
	}

	sk := ed25519.NewKeyFromSeed(seedBytes)
	idStr := fmt.Sprint(id)
	signer := httpsig.NewSigner(idStr, sk, httpsig.Ed25519, []string{"(created)", "date"})

	return &Client{
		id:     id,
		sk:     sk,
		signer: signer,
	}, nil
}

// Node gets a client to node given its id
func (c *Client) Node(nodeID string) (*NodeClient, error) {
	ip, err := c.AddressOf(nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node address")
	}

	log.Debug().Str("node-id", nodeID).Str("ip", ip.String()).Msg("found node ip")

	return &NodeClient{
		client: c,
		ip:     ip,
	}, nil

}

// NodeID returns the yggdrasil node ID of s
func (c *Client) nodeID(id string) *crypto.NodeID {
	pubkey := base58.Decode(id)

	curvePubkey := zoscrypt.PublicKeyToCurve25519(pubkey)
	var box crypto.BoxPubKey
	copy(box[:], curvePubkey[:])
	return crypto.GetNodeID(&box)
}

// AddressOf return the yggdrasil node address given it's node id
func (c *Client) AddressOf(nodeID string) (net.IP, error) {
	id := c.nodeID(nodeID)

	ip := make([]byte, net.IPv6len)
	copy(ip, address.AddrForNodeID(id)[:])

	return ip, nil
}
