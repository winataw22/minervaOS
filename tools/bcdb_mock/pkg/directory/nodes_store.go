package directory

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	directory "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// NodeAPI holds api for nodes
type NodeAPI struct{}

// List farms
// TODO: add paging arguments
func (s *NodeAPI) List(ctx context.Context, db *mongo.Database, farm schema.ID) ([]directory.Node, error) {
	var filter directory.NodeFilter
	if farm > 0 {
		filter = filter.WithFarmID(farm)
	}

	cur, err := filter.Find(ctx, db, models.Page(0))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}
	defer cur.Close(ctx)
	var out []directory.Node
	if err := cur.All(ctx, &out); err != nil {
		return nil, errors.Wrap(err, "failed to load node list")
	}

	return out, nil
}

// Get a single node
func (s *NodeAPI) Get(ctx context.Context, db *mongo.Database, nodeID string) (directory.Node, error) {
	var filter directory.NodeFilter
	filter = filter.WithNodeID(nodeID)
	return filter.Get(ctx, db)
}

// Exists tests if node exists
func (s *NodeAPI) Exists(ctx context.Context, db *mongo.Database, nodeID string) (bool, error) {
	var filter directory.NodeFilter
	filter = filter.WithNodeID(nodeID)

	count, err := filter.Count(ctx, db)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Add a node to the store
func (s *NodeAPI) Add(ctx context.Context, db *mongo.Database, node directory.Node) (schema.ID, error) {
	return directory.NodeCreate(ctx, db, node)
}

func (s *NodeAPI) updateTotalCapacity(ctx context.Context, db *mongo.Database, nodeID string, capacity generated.TfgridDirectoryNodeResourceAmount1) error {
	return directory.NodeUpdateTotalResources(ctx, db, nodeID, capacity)
}

func (s *NodeAPI) updateReservedCapacity(ctx context.Context, db *mongo.Database, nodeID string, capacity generated.TfgridDirectoryNodeResourceAmount1) error {
	return directory.NodeUpdateUsedResources(ctx, db, nodeID, capacity)
}

func (s *NodeAPI) updateUptime(ctx context.Context, db *mongo.Database, nodeID string, uptime int64) error {
	return directory.NodeUpdateUptime(ctx, db, nodeID, uptime)
}

// StoreProof stores node hardware proof
func (s *NodeAPI) StoreProof(ctx context.Context, db *mongo.Database, nodeID string, dmi dmi.DMI, disks capacity.Disks, hypervisor []string) error {
	var err error
	proof := generated.TfgridDirectoryNodeProof1{
		Created:    schema.Date{Time: time.Now()},
		Hypervisor: hypervisor,
	}

	proof.Hardware = map[string]interface{}{
		"sections": dmi.Sections,
		"tooling":  dmi.Tooling,
	}
	proof.HardwareHash, err = hashProof(proof.Hardware)
	if err != nil {
		return err
	}

	proof.Disks = map[string]interface{}{
		"aggregator":  disks.Aggregator,
		"environment": disks.Environment,
		"devices":     disks.Devices,
		"tool":        disks.Tool,
	}
	proof.DiskHash, err = hashProof(proof.Disks)
	if err != nil {
		return err
	}

	return directory.NodePushProof(ctx, db, nodeID, proof)
}

// SetInterfaces updates node interfaces
func (s *NodeAPI) SetInterfaces(ctx context.Context, db *mongo.Database, nodeID string, ifaces []generated.TfgridDirectoryNodeIface1) error {
	return directory.NodeSetInterfaces(ctx, db, nodeID, ifaces)
}

// SetPublicConfig sets node public config
func (s *NodeAPI) SetPublicConfig(ctx context.Context, db *mongo.Database, nodeID string, cfg generated.TfgridDirectoryNodePublicIface1) error {
	node, err := s.Get(ctx, db, nodeID)
	if err != nil {
		return err
	}

	if node.PublicConfig == nil {
		cfg.Version = 1
	} else {
		cfg.Version = node.PublicConfig.Version + 1
	}

	return directory.NodeSetPublicConfig(ctx, db, nodeID, cfg)
}

// SetWGPorts sets node gateway ports
func (s *NodeAPI) SetWGPorts(ctx context.Context, db *mongo.Database, nodeID string, ports []uint) error {
	return directory.NodeSetWGPorts(ctx, db, nodeID, ports)
}

// Requires is a wrapper that makes sure node with that case exists before
// running the handler
func (s *NodeAPI) Requires(key string, handler mw.Action) mw.Action {
	return func(r *http.Request) (interface{}, mw.Response) {
		nodeID, ok := mux.Vars(r)[key]
		if !ok {
			// programming error, we should panic in this case
			panic("invalid node-id key")
		}

		db := mw.Database(r)

		exists, err := s.Exists(r.Context(), db, nodeID)
		if err != nil {
			return nil, mw.Error(err)
		} else if !exists {
			return nil, mw.NotFound(fmt.Errorf("node '%s' not found", nodeID))
		}

		return handler(r)
	}
}

// hashProof return the hex encoded md5 hash of the json encoded version of p
func hashProof(p map[string]interface{}) (string, error) {

	// we are trying to have always produce same hash for same content of p
	// so we convert the map into a list so we can sort
	// the key and workaround the fact that maps are not sorted

	type kv struct {
		k string
		v interface{}
	}

	kvs := make([]kv, len(p))
	for k, v := range p {
		kvs = append(kvs, kv{k: k, v: v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })

	b, err := json.Marshal(kvs)
	if err != nil {
		return "", err
	}
	h := md5.New()
	bh := h.Sum(b)
	return fmt.Sprintf("%x", bh), nil
}
