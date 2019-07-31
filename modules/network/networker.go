package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules/network/ip"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network/bridge"

	"github.com/threefoldtech/zosv2/modules/network/namespace"

	"github.com/threefoldtech/zosv2/modules"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
)

type networker struct {
	identity   modules.IdentityManager
	storageDir string
	tnodb      TNoDB
}

// NewNetworker create a new modules.Networker that can be used over zbus
func NewNetworker(identity modules.IdentityManager, tnodb TNoDB, storageDir string) modules.Networker {
	nw := &networker{
		identity:   identity,
		storageDir: storageDir,
		tnodb:      tnodb,
	}

	return nw
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

func (n networker) Namespace(id modules.NetID) (string, error) {
	b, err := ioutil.ReadFile(filepath.Join(n.storageDir, string(id)))
	if err != nil {
		return "", err
	}
	return string(b), err
}

// ApplyNetResource implements modules.Networker interface
func (n *networker) ApplyNetResource(network modules.Network) (string, error) {
	var err error

	if err := validateNetwork(&network); err != nil {
		log.Error().Err(err).Msg("network object format invalid")
		return "", err
	}

	localResource := n.localResource(network.Resources)
	if localResource == nil {
		return "", fmt.Errorf("not network resource for this node: %s", n.identity.NodeID())
	}
	exitNetRes, err := exitResource(network.Resources)
	if err != nil {
		return "", err
	}
	nibble := ip.NewNibble(localResource.Prefix, network.AllocationNR)

	wgKey, err := n.extractPrivateKey(localResource)
	if err != nil {
		return "", errors.Wrap(err, "failed to extract private key from network object")
	}

	// the flow is a bit different if the network namespace already exist or not
	// if it already exists, we skip the all network resource creation
	// and only do the wireguard configuration
	// so any new updated wireguard peer will be updated
	creation := !namespace.Exists(nibble.NetworkName())

	if creation {
		log.Info().Msg("create new network resource")
	} else {
		log.Info().Msg("update existing network resource")
	}

	defer func() {
		if err != nil {
			if err := n.DeleteNetResource(network); err != nil {
				log.Error().Err(err).Msg("during during deletion of network resource after failed deployment")
			}
		}
	}()

	if creation {
		log.Info().Msg("create network resource namespace")
		err = createNetworkResource(localResource, &network)
		if err != nil {
			return "", err
		}
	}

	log.Info().Msg("Generate wireguard config for all peers")
	peers, routes, err := genWireguardPeers(localResource, &network)
	if err != nil {
		return "", err
	}

	log.Debug().
		Str("local prefix", localResource.Prefix.String()).
		Str("exit prefix", exitNetRes.Prefix.String()).
		Msg("configure wireguard exit node")

	// if we are not the exit node, then add the default route to the exit node
	if localResource.Prefix.String() != exitNetRes.Prefix.String() {
		log.Info().Msg("Generate wireguard config to the exit node")
		exitPeers, exitRoutes, err := genWireguardExitPeers(localResource, &network)
		if err != nil {
			return "", err
		}
		peers = append(peers, exitPeers...)
		routes = append(routes, exitRoutes...)
	} else if creation { // we are the exit node and this network resource is being creating
		log.Info().Msg("Configure network resource as exit point")
		err := configNetResAsExitPoint(exitNetRes, network.Exit, network.PrefixZero)
		if err != nil {
			return "", err
		}
	}

	log.Info().
		Int("number of peers", len(peers)).
		Msg("configure wg")

	err = configWG(localResource, &network, peers, routes, wgKey)
	if err != nil {
		return "", err
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	if err := ioutil.WriteFile(path, []byte(nibble.NetworkName()), 0660); err != nil {
		return "", errors.Wrap(err, "fail to write file that maps network ID to network namespace")
	}

	return nibble.NetworkName(), nil
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

	// map the network ID to the network namespace
	path := filepath.Join(n.storageDir, string(network.NetID))
	if err := os.Remove(path); err != nil {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) extractPrivateKey(r *modules.NetResource) (wgtypes.Key, error) {
	key := wgtypes.Key{}

	peer, err := getPeer(r.Prefix.String(), r.Peers)
	if err != nil {
		return key, err
	}
	if peer.Connection.PrivateKey == "" {
		return key, fmt.Errorf("wireguard private key is empty")
	}

	// private key is hex encoded in the network object
	sk := ""
	_, err = fmt.Sscanf(peer.Connection.PrivateKey, "%x", &sk)
	if err != nil {
		return key, err
	}

	decoded, err := n.identity.Decrypt([]byte(sk))
	if err != nil {
		return key, err
	}

	return wgtypes.ParseKey(string(decoded))
}
