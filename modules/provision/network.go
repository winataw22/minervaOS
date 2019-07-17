package provision

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/identity"

	"github.com/threefoldtech/zosv2/modules/stubs"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/ip"
)

func networkProvision(ctx context.Context, netID modules.NetID) (namespace string, err error) {

	mgr := stubs.NewNetworkerStub(GetZBus(ctx))

	wgK, err := mgr.GenerateWireguarKeyPair(netID)
	if err != nil {
		return "", err
	}

	nodeID, err := identity.LocalNodeID()
	if err != nil {
		return "", err
	}

	db := GetTnoDB(ctx)
	network, err := db.GetNetwork(netID)
	if err != nil {
		return "", err
	}

	if !isInNetwork(network, nodeID) {
		nw, err := mgr.JoinNetwork(network.NetID, 1610, wgK)
		if err != nil {
			return "", errors.Wrapf(err, "failed to join network %s", network.NetID)
		}
		network = &nw
	} else {
		// TODO check if we really need to publish it
		if err := mgr.PublishWGPubKey(wgK, netID); err != nil {
			return "", errors.Wrap(err, "failed to publish wireguard key")
		}
	}
	log.Debug().Msgf("network %+v", network)

	if err := mgr.ApplyNetResource(*network); err != nil {
		return "", err
	}

	namespace, err = networkGetNamespace(network, nodeID)
	if err != nil {
		return "", err
	}

	return namespace, err
}

// isInNetwork checks if nodeID already has a network resource in the
// network object
func isInNetwork(network *modules.Network, nodeID identity.Identifier) bool {
	id := nodeID.Identity()
	for _, nr := range network.Resources {
		if nr.NodeID.ID == id {
			return true
		}
	}
	return false
}

func networkGetNamespace(network *modules.Network, nodeID identity.Identifier) (string, error) {
	var res *modules.NetResource
	for _, r := range network.Resources {
		if r.NodeID.ID == nodeID.Identity() {
			res = r
			break
		}
	}
	if res == nil {
		return "", fmt.Errorf("no network resource find for this node")
	}

	nib := ip.NewNibble(res.Prefix, network.AllocationNR)
	return nib.NetworkName(), nil
}

// NetworkProvision is entry point to provision a network
func NetworkProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	x := struct {
		NetID modules.NetID `json:"network_id"`
	}{}
	if err := json.Unmarshal(reservation.Data, &x); err != nil {
		return nil, err
	}

	_, err := networkProvision(ctx, x.NetID)
	return nil, err
}
