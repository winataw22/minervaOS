package networkd

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// ErrNoPubInterface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubInterface = errors.New("no public interface configured for this node")

func getExitInterface(dir client.Directory, nodeID string) (*types.PubIface, error) {
	schemaNode, err := dir.NodeGet(nodeID, false)
	if err != nil {
		return nil, err
	}

	node := types.NewNodeFromSchema(schemaNode)
	if node.PublicConfig == nil {
		return nil, ErrNoPubInterface
	}

	return node.PublicConfig, nil
}
