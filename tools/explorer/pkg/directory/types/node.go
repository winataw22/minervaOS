package types

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	generated "github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// NodeCollection db collection name
	NodeCollection = "node"
)

// Node model
type Node generated.Node

// Validate node
func (n *Node) Validate() error {
	if len(n.NodeId) == 0 {
		return fmt.Errorf("node_is is required")
	}

	if n.FarmId == 0 {
		return fmt.Errorf("farm_id is required")
	}

	if len(n.OsVersion) == 0 {
		return fmt.Errorf("os_version is required")
	}

	if len(n.PublicKeyHex) == 0 {
		return fmt.Errorf("public_key_hex is required")
	}

	pk, err := hex.DecodeString(n.PublicKeyHex)
	if err != nil {
		return errors.Wrap(err, "fail to decode public key")
	}

	if n.NodeId != base58.Encode(pk) {
		return fmt.Errorf("nodeID and public key does not match")
	}

	// Unfortunately, jsx schema does not support nil types
	// so this is the only way to check if values are not set
	empty := generated.Location{}
	if n.Location == empty {
		return fmt.Errorf("location is required")
	}

	return nil
}

// NodeFilter type
type NodeFilter bson.D

// WithID filter node with ID
func (f NodeFilter) WithID(id schema.ID) NodeFilter {
	return append(f, bson.E{Key: "_id", Value: id})
}

// WithNodeID search nodes with this node id
func (f NodeFilter) WithNodeID(id string) NodeFilter {
	return append(f, bson.E{Key: "node_id", Value: id})
}

// WithFarmID search nodes with given farmID
func (f NodeFilter) WithFarmID(id schema.ID) NodeFilter {
	return append(f, bson.E{Key: "farm_id", Value: id})
}

// WithTotalCap filter with total cap only units that > 0 are used
// in the query
func (f NodeFilter) WithTotalCap(cru, mru, hru, sru int64) NodeFilter {
	for k, v := range map[string]int64{
		"total_resources.cru": cru,
		"total_resources.mru": mru,
		"total_resources.hru": hru,
		"total_resources.sru": sru} {
		if v > 0 {
			f = append(f, bson.E{Key: k, Value: bson.M{"$gte": v}})
		}
	}

	return f
}

// WithLocation search the nodes that are located in country and or city
func (f NodeFilter) WithLocation(country, city string) NodeFilter {
	if country != "" {
		f = append(f, bson.E{Key: "location.country", Value: country})
	}
	if city != "" {
		f = append(f, bson.E{Key: "location.city", Value: city})
	}

	return f
}

// Find run the filter and return a cursor result
func (f NodeFilter) Find(ctx context.Context, db *mongo.Database, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	col := db.Collection(NodeCollection)
	if f == nil {
		f = NodeFilter{}
	}

	return col.Find(ctx, f, opts...)
}

// Get one farm that matches the filter
func (f NodeFilter) Get(ctx context.Context, db *mongo.Database, includeproofs bool) (node Node, err error) {
	if f == nil {
		f = NodeFilter{}
	}

	col := db.Collection(NodeCollection)

	var projection bson.D
	if !includeproofs {
		projection = bson.D{
			{Key: "proofs", Value: 0},
		}
	} else {
		projection = bson.D{}
	}
	result := col.FindOne(ctx, f, options.FindOne().SetProjection(projection))

	err = result.Err()
	if err != nil {
		return
	}

	err = result.Decode(&node)
	return
}

// Count number of documents matching
func (f NodeFilter) Count(ctx context.Context, db *mongo.Database) (int64, error) {
	col := db.Collection(NodeCollection)
	if f == nil {
		f = NodeFilter{}
	}

	return col.CountDocuments(ctx, f)
}

// Delete deletes a node by ID
func (f NodeFilter) Delete(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(NodeCollection)
	if f == nil {
		f = NodeFilter{}
	}
	_, err := col.DeleteOne(ctx, f, options.Delete())
	return err
}

// NodeCreate creates a new farm
func NodeCreate(ctx context.Context, db *mongo.Database, node Node) (schema.ID, error) {
	if err := node.Validate(); err != nil {
		return 0, err
	}

	var farmFilter FarmFilter
	farmFilter = farmFilter.WithID(schema.ID(node.FarmId))
	_, err := farmFilter.Get(ctx, db)
	if err != nil {
		return 0, errors.Wrap(err, "unknown farm id")
	}

	var filter NodeFilter
	filter = filter.WithNodeID(node.NodeId)
	var id schema.ID
	current, err := filter.Get(ctx, db, false)
	if err != nil {
		//TODO: check that this is a NOT FOUND error
		id, err = models.NextID(ctx, db, NodeCollection)
		if err != nil {
			return id, err
		}
		node.Created = schema.Date{Time: time.Now()}
	} else {
		id = current.ID
		// make sure we do NOT overwrite these field
		node.Created = current.Created
		node.FreeToUse = current.FreeToUse
	}

	node.ID = id
	if node.Proofs == nil {
		node.Proofs = make([]generated.Proof, 0)
	}

	node.Updated = schema.Date{Time: time.Now()}
	col := db.Collection(NodeCollection)
	_, err = col.UpdateOne(ctx, filter, bson.M{"$set": node}, options.Update().SetUpsert(true))
	return id, err
}

func nodeUpdate(ctx context.Context, db *mongo.Database, nodeID string, value interface{}) error {
	if nodeID == "" {
		return fmt.Errorf("invalid node id")
	}

	col := db.Collection(NodeCollection)
	var filter NodeFilter
	filter = filter.WithNodeID(nodeID)
	_, err := col.UpdateOne(ctx, filter, bson.M{
		"$set": value,
	})

	return err
}

// NodeUpdateTotalResources sets the node total resources
func NodeUpdateTotalResources(ctx context.Context, db *mongo.Database, nodeID string, capacity generated.ResourceAmount) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{"total_resources": capacity})
}

// NodeUpdateReservedResources sets the node reserved resources
func NodeUpdateReservedResources(ctx context.Context, db *mongo.Database, nodeID string, capacity generated.ResourceAmount) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{"reserved_resources": capacity})
}

// NodeUpdateUsedResources sets the node total resources
func NodeUpdateUsedResources(ctx context.Context, db *mongo.Database, nodeID string, capacity generated.ResourceAmount) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{"used_resources": capacity})
}

// NodeUpdateWorkloadsAmount sets the node reserved resources
func NodeUpdateWorkloadsAmount(ctx context.Context, db *mongo.Database, nodeID string, workloads generated.WorkloadAmount) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{"workloads": workloads})
}

// NodeUpdateUptime updates node uptime
func NodeUpdateUptime(ctx context.Context, db *mongo.Database, nodeID string, uptime int64) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{
		"uptime":  uptime,
		"updated": schema.Date{Time: time.Now()},
	})
}

// NodeSetInterfaces updates node interfaces
func NodeSetInterfaces(ctx context.Context, db *mongo.Database, nodeID string, ifaces []generated.Iface) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{
		"ifaces": ifaces,
	})
}

// NodeSetPublicConfig sets node public config
func NodeSetPublicConfig(ctx context.Context, db *mongo.Database, nodeID string, cfg generated.PublicIface) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{
		"public_config": cfg,
	})
}

// NodeUpdateFreeToUse sets node free to use flag
func NodeUpdateFreeToUse(ctx context.Context, db *mongo.Database, nodeID string, freeToUse bool) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{
		"free_to_use": freeToUse,
	})
}

// NodeSetWGPorts update wireguard ports
func NodeSetWGPorts(ctx context.Context, db *mongo.Database, nodeID string, ports []uint) error {
	return nodeUpdate(ctx, db, nodeID, bson.M{
		"wg_ports": ports,
	})
}

// NodePushProof push proof to node
func NodePushProof(ctx context.Context, db *mongo.Database, nodeID string, proof generated.Proof) error {
	if nodeID == "" {
		return fmt.Errorf("invalid node id")
	}

	col := db.Collection(NodeCollection)
	var filter NodeFilter
	filter = filter.WithNodeID(nodeID)
	_, err := col.UpdateOne(ctx, filter, bson.M{
		"$addToSet": bson.M{
			"proofs": proof,
		},
	})

	return err
}

// FarmsForNodes return farm objects given node ids
func FarmsForNodes(ctx context.Context, db *mongo.Database, nodeID ...string) (farms []directory.Farm, err error) {
	// this pipline finds all farms from a list of nodes and return the list
	if len(nodeID) == 0 {
		return
	}

	pipeline := mongo.Pipeline{
		{{
			Key: "$match",
			Value: bson.M{
				"node_id": bson.M{
					"$in": nodeID,
				},
			},
		}},
		{{
			Key: "$group",
			Value: bson.M{
				"_id": "$farm_id",
			},
		}},
		{{
			Key: "$lookup",
			Value: bson.M{
				"from":         FarmCollection,
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "farm"},
		}},
		{{
			Key: "$replaceRoot",
			Value: bson.M{
				"newRoot": bson.M{
					"$arrayElemAt": []interface{}{"$farm", 0},
				},
			},
		}},
	}

	cur, err := db.Collection(NodeCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	err = cur.All(ctx, &farms)
	return
}
