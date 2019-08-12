package main

import (
	"encoding/json"
	"fmt"
	"net"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/google/uuid"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/threefoldtech/zosv2/modules/tno"
)

func createNetwork(nodeID string) (*modules.Network, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("exit node ID must be specified")
	}

	node, err := db.GetNode(modules.StrIdentifier(nodeID))
	if err != nil {
		return nil, err
	}

	if !node.ExitNode {
		return nil, fmt.Errorf("node %s cannot be used as exit node", nodeID)
	}

	pubIface, err := db.ReadPubIface(modules.StrIdentifier(node.NodeID))
	if err != nil {
		return nil, errors.Wrap(err, "fail to read public interface config")
	}

	allocation, farmAlloc, err := db.RequestAllocation(modules.StrIdentifier(node.FarmID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to request a new allocation")
	}

	_, farmAllocSize := farmAlloc.Mask.Size()

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	network := &modules.Network{}
	err = tno.Configure(network, []tno.Opts{
		tno.GenerateID(),
		tno.ConfigurePrefixZero(farmAlloc),
		tno.ConfigureExitResource(node.NodeID, allocation, pubIface.IPv6.IP, key, farmAllocSize),
	})
	if err != nil {
		return nil, err
	}
	network.Resources[0].NodeID.FarmerID = node.FarmID

	return network, nil
}

func addNode(nw *modules.Network, nodeID string) (*modules.Network, error) {
	if len(nw.Resources) <= 0 {
		return nil, fmt.Errorf("cannot add a node to network without exit node")
	}

	farmID := nw.Resources[0].NodeID.FarmerID

	allocation, _, err := db.RequestAllocation(modules.StrIdentifier(farmID))
	if err != nil {
		return nil, err
	}

	var (
		pubIface *network.PubIface
		ip       net.IP
	)
	pubIface, err = db.ReadPubIface(modules.StrIdentifier(nodeID))
	if err != nil {
		ip = nil
	} else {
		ip = pubIface.IPv6.IP
	}

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	err = tno.Configure(nw, []tno.Opts{
		tno.AddNode(nodeID, farmID, allocation, key, ip),
	})
	if err != nil {
		return nil, err
	}

	nw.Version++

	return nw, nil
}

func addUser(network *modules.Network, userID string) (*modules.Network, string, error) {
	if len(network.Resources) <= 0 {
		return nil, "", fmt.Errorf("cannot add a node to network without exit node")
	}

	farmID := network.Resources[0].NodeID.FarmerID
	allocation, _, err := db.RequestAllocation(modules.StrIdentifier(farmID))
	if err != nil {
		return nil, "", err
	}

	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, "", err
	}
	// todo serialize the key somewhere

	err = tno.Configure(network, []tno.Opts{
		tno.AddUser(userID, allocation, key),
	})
	if err != nil {
		return nil, "", err
	}

	network.Version++

	return network, key.String(), nil
}

func removeNode(network *modules.Network, nodeID string) (*modules.Network, error) {
	err := tno.Configure(network, []tno.Opts{
		tno.RemoveNode(nodeID),
	})
	if err != nil {
		return nil, err
	}

	network.Version++
	return network, nil
}

func reserveNetwork(network *modules.Network) error {
	raw, err := json.Marshal(network)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := &provision.Reservation{
		ID:   id.String(),
		Type: provision.NetworkReservation,
		Data: raw,
	}

	nodes := mapset.NewSet()
	for _, r := range network.Resources {
		nodes.Add(r.NodeID.ID)
	}

	for nodeID := range nodes.Iter() {
		nodeID := nodeID.(string)
		if err := store.Reserve(r, modules.StrIdentifier(nodeID)); err != nil {
			return err
		}
		fmt.Printf("network reservation sent for node ID %s\n", nodeID)
	}

	return nil
}
