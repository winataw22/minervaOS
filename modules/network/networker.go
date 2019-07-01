package network

import (
	"fmt"
	"path/filepath"

	"github.com/threefoldtech/zosv2/modules/network/wireguard"

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
	tnodb       TNoDB
}

// NewNetworker create a new modules.Networker that can be used over zbus
func NewNetworker(storageDir string, allocator NetResourceAllocator, nodeID modules.NodeID) modules.Networker {
	return &networker{
		nodeID:      nodeID,
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
func (n *networker) ApplyNetResource(network *modules.Network) (err error) {
	log.Info().Msg("apply netresource")
	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node: %s", n.nodeID.ID)
	}

	defer func() {
		if err != nil {
			if err := n.DeleteNetResource(network); err != nil {
				log.Error().Err(err).Msg("during during deletion of network resource after failed deployment")
			}
		}
	}()

	log.Info().Msg("create net resource namespace")
	err = createNetworkResource(localResource, network)
	if err != nil {
		return err
	}

	log.Info().Msg("Generate wireguard config for all peers")
	peers, routes, err := genWireguardPeers(localResource, network)
	if err != nil {
		return err
	}

	// if we are not the exit node, then add the default route to the exit node
	if localResource.Prefix.String() != network.Exit.Prefix.String() {
		log.Info().Msg("Generate wireguard config to the exit node")
		exitPeers, exitRoutes, err := genWireguardExitPeers(localResource, network)
		if err != nil {
			return err
		}
		peers = append(peers, exitPeers...)
		routes = append(routes, exitRoutes...)
	}

	log.Info().
		Int("number of peers", len(peers)).
		Msg("configure wg")
	err = configWG(localResource, network, peers, routes, n.storageDir)
	if err != nil {
		return err
	}

	return nil
}

// ApplyNetResource implements modules.Networker interface
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

// GenerateWireguarKeyPair generate a pair of keys for a specific network
// and return the base64 encode version of the public key
func (n *networker) GenerateWireguarKeyPair(netID modules.NetID) (string, error) {
	path := filepath.Join(n.storageDir, string(netID))
	key, err := wireguard.GenerateKey(path)
	if err != nil {
		return "", err
	}
	return key.PublicKey().String(), nil
}
func (n *networker) PublishWireguarKeyPair(key string, nodeID modules.NodeID, netID modules.NetID) error {
	return n.tnodb.PublishWireguarKey(key, nodeID, netID)
}
