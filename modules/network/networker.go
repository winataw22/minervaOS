package network

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

type networker struct {
	nodeID      modules.NodeID
	storageDir  string
	netResAlloc NetResourceAllocator
}

// NewNetworker create a new modules.Networker that can be used with zbus
func NewNetworker(storageDir string, allocator NetResourceAllocator) modules.Networker {
	return &networker{
		storageDir:  storageDir,
		netResAlloc: allocator,
	}
}

var _ modules.Networker = (*networker)(nil)

// GetNetwork implements modules.Networker interface
func (n *networker) GetNetwork(id string) (*modules.Network, error) {
	// TODO check signature
	return n.netResAlloc.Get(id)
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) ApplyNetResource(network *modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node: %s", n.nodeID.ID)
	}

	if err := createNetworkResource(localResource, network); err != nil {
		return err
	}

	peers, routes, err := prepareHidden(localResource, network)
	if err != nil {
		return err
	}

	if isPublic(localResource.NodeID) {
		pubPeers, pubRoutes, err := preparePublic(localResource, network)
		if err != nil {
			return err
		}
		peers = append(peers, pubPeers...)
		routes = append(routes, pubRoutes...)
	}

	exitPeers, exitRoutes, err := prepareNonExitNode(localResource, network)
	if err != nil {
		return err
	}

	peers = append(peers, exitPeers...)
	routes = append(routes, exitRoutes...)

	if err := configWG(localResource, network, peers, routes, n.storageDir); err != nil {
		return err
	}
	return nil
}

func (n *networker) DeleteNetResource(network *modules.Network) error {
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node")
	}
	var (
		nibble     = zosip.NewNibble(localResource.Prefix, network.AllocationNR)
		netnsName  = nibble.NetworkName()
		bridgeName = nibble.BridgeName()
	)
	if err := bridge.Delete(bridgeName); err != nil {
		log.Error().
			Err(err).
			Str("bridge", bridgeName).
			Msg("failed to delete network resource bridge")
		return err
	}

	netResNS, err := namespace.GetByName(netnsName)
	if err != nil {
		return err
	}
	// don't explicitly netResNS.Close() the netResNS here, namespace.Delete will take care of it
	if err := namespace.Delete(netResNS); err != nil {
		log.Error().
			Err(err).
			Str("namespace", netnsName).
			Msg("failed to delete network resource namespace")
	}
	return nil
}
