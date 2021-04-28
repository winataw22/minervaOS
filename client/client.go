package client

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jbenet/go-base58"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	zoscrypt "github.com/threefoldtech/zos/pkg/crypto"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/crypto"
)

// Client struct
type Client struct {
	id uint32
	sk ed25519.PrivateKey
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

	return &Client{
		id: id,
		sk: sk,
	}, nil
}

func (c *Client) getAuthHeader() (string, error) {
	token := jwt.New()
	token.Set(jwt.IssuerKey, fmt.Sprint(c.id))
	token.Set(jwt.AudienceKey, "zos")
	now := time.Now()
	token.Set(jwt.IssuedAtKey, now.Unix())
	token.Set(jwt.ExpirationKey, now.Add(1*time.Minute).Unix())

	//jwt.ParseHeader(hdr http.Header, name string, options ...jwt.ParseOption)
	j, err := jwt.Sign(token, jwa.EdDSA, c.sk)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", string(j)), nil
}

func (c *Client) authorize(r *http.Request) error {
	token, err := c.getAuthHeader()
	if err != nil {
		return err
	}

	r.Header.Set("authorization", token)
	return nil
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
