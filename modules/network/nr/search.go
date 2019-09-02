package nr

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"
)

// ResourceByNodeID return the net resource associated with a nodeID
func ResourceByNodeID(nodeID string, resources []*modules.NetResource) (*modules.NetResource, error) {
	for _, resource := range resources {
		if resource.NodeID.ID == nodeID {
			return resource, nil
		}
	}
	return nil, fmt.Errorf("not network resource for this node: %s", nodeID)
}

// PeerByPrefix return a wireguard peer from a list of peers by its prefix
func PeerByPrefix(prefix string, peers []*modules.Peer) (*modules.Peer, error) {
	for _, peer := range peers {
		if peer.Prefix.String() == prefix {
			return peer, nil
		}
	}
	return nil, fmt.Errorf("peer not found")
}
