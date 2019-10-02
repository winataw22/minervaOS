package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net"

// 	mapset "github.com/deckarep/golang-set"
// 	"github.com/pkg/errors"

// 	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

// 	"github.com/google/uuid"
// 	"github.com/threefoldtech/zos/pkg"
// 	"github.com/threefoldtech/zos/pkg/network/types"
// 	"github.com/threefoldtech/zos/pkg/provision"
// 	"github.com/threefoldtech/zos/pkg/tno"
// )

// func createNetwork(nodeID string) (*pkg.Network, error) {
// 	if nodeID == "" {
// 		return nil, fmt.Errorf("exit node ID must be specified")
// 	}

// 	node, err := db.GetNode(pkg.StrIdentifier(nodeID))
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to get node detail from BCDB")
// 	}

// 	if node.ExitNode <= 0 {
// 		return nil, fmt.Errorf("node %s cannot be used as exit node", nodeID)
// 	}

// 	publicIP, err := selectPublicIP(node)
// 	if err != nil {
// 		return nil, err
// 	}

// 	allocation, farmAlloc, exitNodeNr, err := db.RequestAllocation(pkg.StrIdentifier(node.NodeID))
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "failed to request a new allocation")
// 	}

// 	key, err := wgtypes.GeneratePrivateKey()
// 	if err != nil {
// 		return nil, err
// 	}

// 	network := &pkg.Network{}
// 	err = tno.Configure(network, []tno.Opts{
// 		tno.GenerateID(),
// 		tno.ConfigurePrefixZero(farmAlloc),
// 		tno.ConfigureExitResource(node.NodeID, allocation, publicIP, key, exitNodeNr),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	network.Resources[0].NodeID.FarmerID = node.FarmID

// 	return network, nil
// }

// func selectPublicIP(node *types.Node) (net.IP, error) {
// 	if node.PublicConfig != nil && node.PublicConfig.IPv6 != nil {
// 		return node.PublicConfig.IPv6.IP, nil
// 	}

// 	for _, iface := range node.Ifaces {
// 		for _, addr := range iface.Addrs {
// 			if addr.IP.IsGlobalUnicast() {
// 				return addr.IP, nil
// 			}
// 		}
// 	}

// 	return nil, fmt.Errorf("no public address found")
// }

// func addNode(nw *pkg.Network, nodeID string) (*pkg.Network, error) {
// 	if len(nw.Resources) <= 0 {
// 		return nil, fmt.Errorf("cannot add a node to network without exit node")
// 	}

// 	farm, err := db.GetNode(pkg.StrIdentifier(nodeID))
// 	if err != nil {
// 		return nil, err
// 	}

// 	allocation, _, _, err := db.RequestAllocation(nw.Resources[0].NodeID)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "failed to request a new allocation")
// 	}

// 	var (
// 		pubIface *types.PubIface
// 		ip       net.IP
// 	)
// 	pubIface, err = db.ReadPubIface(pkg.StrIdentifier(nodeID))
// 	if err != nil {
// 		ip = nil
// 	} else {
// 		ip = pubIface.IPv6.IP
// 	}

// 	key, err := wgtypes.GeneratePrivateKey()
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = tno.Configure(nw, []tno.Opts{
// 		tno.AddNode(nodeID, farm.FarmID, allocation, key, ip),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	nw.Version++

// 	return nw, nil
// }

// func addUser(network *pkg.Network, userID string) (*pkg.Network, string, error) {
// 	if len(network.Resources) <= 0 {
// 		return nil, "", fmt.Errorf("cannot add a node to network without exit node")
// 	}

// 	farmID := network.Resources[0].NodeID.FarmerID
// 	allocation, _, _, err := db.RequestAllocation(pkg.StrIdentifier(farmID))
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	key, err := wgtypes.GeneratePrivateKey()
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	err = tno.Configure(network, []tno.Opts{
// 		tno.AddUser(userID, allocation, key),
// 	})
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	network.Version++

// 	return network, key.String(), nil
// }

// func removeNode(network *pkg.Network, nodeID string) (*pkg.Network, error) {
// 	err := tno.Configure(network, []tno.Opts{
// 		tno.RemoveNode(nodeID),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	network.Version++
// 	return network, nil
// }

// func reserveNetwork(network *pkg.Network) error {
// 	raw, err := json.Marshal(network)
// 	if err != nil {
// 		return err
// 	}

// 	id, err := uuid.NewRandom()
// 	if err != nil {
// 		return err
// 	}
// 	r := &provision.Reservation{
// 		ID:   id.String(),
// 		Type: provision.NetworkReservation,
// 		Data: raw,
// 	}

// 	nodes := mapset.NewSet()
// 	for _, r := range network.Resources {
// 		nodes.Add(r.NodeID.ID)
// 	}

// 	for nodeID := range nodes.Iter() {
// 		nodeID := nodeID.(string)
// 		id, err := store.Reserve(r, pkg.StrIdentifier(nodeID))
// 		if err != nil {
// 			return err
// 		}

// 		fmt.Printf("network reservation sent for node ID %s (%v)\n", nodeID, id)
// 	}

// 	return nil
// }
