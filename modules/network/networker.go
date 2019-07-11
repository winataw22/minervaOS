package network

import (
	"fmt"
	"path/filepath"

	"github.com/threefoldtech/zosv2/modules/identity"

	"github.com/threefoldtech/zosv2/modules/network/wireguard"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

type networker struct {
	nodeID     identity.Identifier
	storageDir string
	tnodb      TNoDB
}

// NewNetworker create a new modules.Networker that can be used over zbus
func NewNetworker(nodeID identity.Identifier, tnodb TNoDB, storageDir string) modules.Networker {
	return &networker{
		nodeID:     nodeID,
		storageDir: storageDir,
		tnodb:      tnodb,
	}
}

var _ modules.Networker = (*networker)(nil)

func validateNetwork(n *modules.Network) error {
	if n.NetID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}

	if n.PrefixZero == nil {
		return fmt.Errorf("PrefixZero cannot be empty")
	}

	if len(n.Resources) < 1 {
		return fmt.Errorf("Network needs at least one network ressource")
	}
	// TODO validate each resource

	if n.Exit == nil {
		return fmt.Errorf("Exit point cannot be empty")
	}

	if n.AllocationNR < 0 {
		return fmt.Errorf("AllocationNR cannot be negative")
	}

	return nil
}

// GetNetwork implements modules.Networker interface
func (n *networker) GetNetwork(id modules.NetID) (net modules.Network, err error) {
	no, err := n.tnodb.GetNetwork(id)
	if err != nil {
		return net, err
	}

	return *no, nil
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) ApplyNetResource(network *modules.Network) (err error) {

	if err := validateNetwork(network); err != nil {
		log.Error().Err(err).Msg("network object format invalid")
		return err
	}


	log.Info().Msg("apply netresource")

	path := filepath.Join(n.storageDir, string(network.NetID))
	wgKey, err := wireguard.LoadKey(path)
	if err != nil {
		log.Error().
			Err(err).
			Str("network", string(network.NetID)).
			Str("directory", path).
			Msg("failed to load wireguard keys. Generate the keys before trying to configure the network")
		return err
	}

	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return fmt.Errorf("not network resource for this node: %s", n.nodeID.Identity())
	}
	exitNetRes, err := exitResource(network.Resources)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if err := n.DeleteNetResource(network); err != nil {
				log.Error().Err(err).Msg("during during deletion of network resource after failed deployment")
			}
		}
	}()

	log.Info().Msg("create net resource namespace")
	err = createNetworkResource(localResource, &network)
	if err != nil {
		return err
	}

	log.Info().Msg("Generate wireguard config for all peers")
	peers, routes, err := genWireguardPeers(localResource, &network)
	if err != nil {
		return err
	}

	// if we are not the exit node, then add the default route to the exit node
	if localResource.Prefix.String() != exitNetRes.Prefix.String() {
		log.Info().Msg("Generate wireguard config to the exit node")
		exitPeers, exitRoutes, err := genWireguardExitPeers(localResource, &network)
		if err != nil {
			return err
		}
		peers = append(peers, exitPeers...)
		routes = append(routes, exitRoutes...)
	} else { // we are the exit node
		log.Info().Msg("Configure network resource as exit point")
		err := configNetResAsExitPoint(exitNetRes, network.Exit, network.PrefixZero)
		if err != nil {
			return err
		}
	}

	log.Info().
		Int("number of peers", len(peers)).
		Msg("configure wg")
	err = configWG(localResource, &network, peers, routes, wgKey)
	if err != nil {
		return err
	}

	return nil
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) DeleteNetResource(network modules.Network) error {
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
func (n *networker) PublishWGPubKey(key string, netID modules.NetID) error {
	return n.tnodb.PublishWireguarKey(key, n.nodeID.Identity(), netID)
}
