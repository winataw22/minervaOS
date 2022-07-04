// Package client provides a simple RMB interface to work with the node.
//
// Requirements
//
// 1. A msgbusd instance must be running on the node. this client uses RMB (message bus)
// to send messages to nodes, and get the repspons.
// 2. A valid ed25519 key pair. this key is used to sign deployments and MUST be the same
// key used to configure the local twin on substrate.
//
// Simple deployment
//
// create an instance from the default rmb client.
// ```
// cl, err := rmb.Default()
// if err != nil {
// 	panic(err)
// }
// ```
// then create an instance of the node client
// ```
// node := client.NewNodeClient(NodeTwinID, cl)
// ```
// define your deployment object
// ```
// dl := gridtypes.Deployment{
// 	Version: Version,
// 	TwinID:  Twin, //LocalTwin,
// 	// this contract id must match the one on substrate
// 	Workloads: []gridtypes.Workload{
// 		network(), // network workload definition
// 		zmount(), // zmount workload definition
// 		publicip(), // public ip definition
// 		zmachine(), // zmachine definition
// 	},
// 	SignatureRequirement: gridtypes.SignatureRequirement{
// 		WeightRequired: 1,
// 		Requests: []gridtypes.SignatureRequest{
// 			{
// 				TwinID: Twin,
// 				Weight: 1,
// 			},
// 		},
// 	},
// }
// ```
// compute hash
// ```
// hash, err := dl.ChallengeHash()
// if err != nil {
// 	panic("failed to create hash")
// }
// fmt.Printf("Hash: %x\n", hash)
// ```
// create the contract and ge the contract id
// then
// ``
// dl.ContractID = 11 // from substrate
// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// defer cancel()
// err = node.DeploymentDeploy(ctx, dl)
// if err != nil {
// 	panic(err)
// }
// ```
package client

import (
	"context"
	"net"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// NodeClient struct
type NodeClient struct {
	nodeTwin uint32
	bus      rmb.Client
}

type Version struct {
	ZOS   string `json:"zos"`
	ZInit string `json:"zinit"`
}

type Interface struct {
	IPs []string `json:"ips"`
	Mac string   `json:"mac"`
}

type ExitDevice struct {
	// IsSingle is set to true if br-pub
	// is connected to zos bridge
	IsSingle bool `json:"is_single"`
	// IsDual is set to true if br-pub is
	// connected to a physical nic
	IsDual bool `json:"is_dual"`
	// AsDualInterface is set to the physical
	// interface name if IsDual is true
	AsDualInterface string `json:"dual_interface"`
}

type args map[string]interface{}

// NewNodeClient creates a new node RMB client. This client then can be used to
// communicate with the node over RMB.
func NewNodeClient(nodeTwin uint32, bus rmb.Client) *NodeClient {
	return &NodeClient{nodeTwin, bus}
}

// DeploymentDeploy sends the deployment to the node for processing.
func (n *NodeClient) DeploymentDeploy(ctx context.Context, dl gridtypes.Deployment) error {
	const cmd = "zos.deployment.deploy"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

// DeploymentUpdate update the given deployment. deployment must be a valid update for
// a deployment that has been already created via DeploymentDeploy
func (n *NodeClient) DeploymentUpdate(ctx context.Context, dl gridtypes.Deployment) error {
	const cmd = "zos.deployment.update"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

// DeploymentGet gets a deployment via contract ID
func (n *NodeClient) DeploymentGet(ctx context.Context, contractID uint64) (dl gridtypes.Deployment, err error) {
	const cmd = "zos.deployment.get"
	in := args{
		"contract_id": contractID,
	}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, in, &dl); err != nil {
		return dl, err
	}

	return dl, nil
}

// DeploymentGet gets a deployment via contract ID
func (n *NodeClient) DeploymentChanges(ctx context.Context, contractID uint64) (changes []gridtypes.Workload, err error) {
	const cmd = "zos.deployment.changes"
	in := args{
		"contract_id": contractID,
	}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, in, &changes); err != nil {
		return changes, err
	}

	return changes, nil
}

// DeploymentDelete deletes a deployment, the node will make sure to decomission all deployments
// and set all workloads to deleted. A call to Get after delete is valid
func (n *NodeClient) DeploymentDelete(ctx context.Context, contractID uint64) error {
	const cmd = "zos.deployment.delete"
	in := args{
		"contract_id": contractID,
	}

	return n.bus.Call(ctx, n.nodeTwin, cmd, in, nil)
}

// Counters returns some node statistics. Including total and available cpu, memory, storage, etc...
func (n *NodeClient) Counters(ctx context.Context) (total gridtypes.Capacity, used gridtypes.Capacity, err error) {
	const cmd = "zos.statistics.get"
	var result struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}
	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return result.Total, result.Used, nil
}

// NetworkListWGPorts return a list of all "taken" ports on the node. A new deployment
// should be careful to use a free port for its network setup.
func (n *NodeClient) NetworkListWGPorts(ctx context.Context) ([]uint16, error) {
	const cmd = "zos.network.list_wg_ports"
	var result []uint16

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (n *NodeClient) HasPublicIPv6(ctx context.Context) (bool, error) {
	const cmd = "zos.network.has_ipv6"
	var result bool

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return false, err
	}

	return result, nil
}

func (n *NodeClient) NetworkListInterfaces(ctx context.Context) (result map[string][]net.IP, err error) {
	const cmd = "zos.network.interfaces"

	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result)

	return
}

// NetworkListAllInterfaces return all physical devices on a node
func (n *NodeClient) NetworkListAllInterfaces(ctx context.Context) (result map[string]Interface, err error) {
	const cmd = "zos.network.admin.interfaces"

	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result)

	return

}

// NetworkSetPublicExitDevice select which physical interface to use as an exit device
// setting `iface` to `zos` will then make node run in a single nic setup.
func (n *NodeClient) NetworkSetPublicExitDevice(ctx context.Context, iface string) error {
	const cmd = "zos.network.admin.set_public_nic"

	return n.bus.Call(ctx, n.nodeTwin, cmd, iface, nil)
}

// NetworkGetPublicExitDevice gets the current dual nic setup of the node.
func (n *NodeClient) NetworkGetPublicExitDevice(ctx context.Context) (exit ExitDevice, err error) {
	const cmd = "zos.network.admin.get_public_nic"

	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &exit)
	return
}

// NetworkListPublicIPs list taken public IPs on the node
func (n *NodeClient) NetworkListPublicIPs(ctx context.Context) ([]string, error) {
	const cmd = "zos.network.list_public_ips"
	var result []string

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NetworkGetPublicConfig returns the current public node network configuration. A node with a
// public config can be used as an access node for wireguard.
func (n *NodeClient) NetworkGetPublicConfig(ctx context.Context) (cfg pkg.PublicConfig, err error) {
	const cmd = "zos.network.public_config_get"

	err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &cfg)
	return
}

func (n *NodeClient) SystemVersion(ctx context.Context) (ver Version, err error) {
	const cmd = "zos.system.version"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &ver); err != nil {
		return
	}

	return
}

func (n *NodeClient) SystemDMI(ctx context.Context) (result dmi.DMI, err error) {
	const cmd = "zos.system.dmi"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return
}

func (n *NodeClient) SystemHypervisor(ctx context.Context) (result string, err error) {
	const cmd = "zos.system.hypervisor"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return
}
