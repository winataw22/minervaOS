package substrate

import (
	"crypto/ed25519"
	"net"

	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	ygg "github.com/yggdrasil-network/yggdrasil-go/src/crypto"
)

// YggdrasilIdentity is a utility function to get the address
// of an yggdrasil server from a public ed25519 key
type YggdrasilIdentity struct {
	pk [32]byte
}

//NewYggdrasilIdentity creates a new instance of YggdrasilIdentity
func NewYggdrasilIdentity(ed ed25519.PublicKey) *YggdrasilIdentity {
	pk := crypto.PublicKeyToCurve25519(ed)
	return &YggdrasilIdentity{pk}
}

// NodeID returns the yggdrasil node ID of s
func (y *YggdrasilIdentity) NodeID() (*ygg.NodeID, error) {

	var box ygg.BoxPubKey
	copy(box[:], y.pk[:])
	return ygg.GetNodeID(&box), nil
}

// Address return the address in the 200::/7 subnet allocated by yggdrasil
func (y *YggdrasilIdentity) Address() (net.IP, error) {
	nodeID, err := y.NodeID()
	if err != nil {
		return nil, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, address.AddrForNodeID(nodeID)[:])

	return ip, nil
}
