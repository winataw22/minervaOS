package types

import (
	"context"
	"fmt"
	"regexp"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	farmNamePattern = regexp.MustCompile("^[a-zA-Z0-9-_]+$")
)

const (
	// FarmCollection db collection name
	FarmCollection = "farm"
)

//Farm mongo db wrapper for generated TfgridDirectoryFarm
type Farm generated.Farm

// Validate validates farm object
func (f *Farm) Validate() error {
	if !farmNamePattern.MatchString(f.Name) {
		return fmt.Errorf("invalid farm name. name can only contain alphanumeric characters dash (-) or underscore (_)")
	}

	if f.ThreebotId == 0 {
		return fmt.Errorf("threebot_id is required")
	}

	if len(f.WalletAddresses) == 0 {
		return fmt.Errorf("invalid wallet_addresses, is required")
	}

	return nil
}

// FarmFilter type
type FarmFilter bson.D

// WithID filter farm with ID
func (f FarmFilter) WithID(id schema.ID) FarmFilter {
	return append(f, bson.E{Key: "_id", Value: id})
}

// WithName filter farm with name
func (f FarmFilter) WithName(name string) FarmFilter {
	return append(f, bson.E{Key: "name", Value: name})
}

// WithOwner filter farm by owner ID
func (f FarmFilter) WithOwner(tid int64) FarmFilter {
	return append(f, bson.E{Key: "threebot_id", Value: tid})
}

// Find run the filter and return a cursor result
func (f FarmFilter) Find(ctx context.Context, db *mongo.Database, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	col := db.Collection(FarmCollection)
	if f == nil {
		f = FarmFilter{}
	}
	return col.Find(ctx, f, opts...)
}

// Count number of documents matching
func (f FarmFilter) Count(ctx context.Context, db *mongo.Database) (int64, error) {
	col := db.Collection(NodeCollection)
	if f == nil {
		f = FarmFilter{}
	}

	return col.CountDocuments(ctx, f)
}

// Get one farm that matches the filter
func (f FarmFilter) Get(ctx context.Context, db *mongo.Database) (farm Farm, err error) {
	if f == nil {
		f = FarmFilter{}
	}
	col := db.Collection(FarmCollection)
	result := col.FindOne(ctx, f, options.FindOne())

	err = result.Err()
	if err != nil {
		return
	}

	err = result.Decode(&farm)
	return
}

// FarmCreate creates a new farm
func FarmCreate(ctx context.Context, db *mongo.Database, farm Farm) (schema.ID, error) {
	if err := farm.Validate(); err != nil {
		return 0, err
	}

	col := db.Collection(FarmCollection)
	id, err := models.NextID(ctx, db, FarmCollection)
	if err != nil {
		return id, err
	}

	farm.ID = id
	_, err = col.InsertOne(ctx, farm)
	return id, err
}
