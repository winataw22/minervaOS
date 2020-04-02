package types

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/config"
	"github.com/threefoldtech/zos/tools/explorer/models"
	generated "github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
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

	found := false
	validator := stellar.NewAddressValidator(config.Config.Network, config.Config.Asset)
	for _, a := range f.WalletAddresses {
		if a.Asset != config.Config.Asset {
			continue
		}

		found = true
		if err := validator.Valid(a.Address); err != nil {
			return err
		}
	}

	if !found {
		return fmt.Errorf("no wallet address found for asset %s", config.Config.Asset)
	}

	return nil
}

// FarmQuery helper to parse query string
type FarmQuery struct {
	FarmName string
	OwnerID  int64
}

// Parse querystring from request
func (f *FarmQuery) Parse(r *http.Request) mw.Response {
	var err error
	f.OwnerID, err = models.QueryInt(r, "owner")
	if err != nil {
		return mw.BadRequest(errors.Wrap(err, "owner should be a integer"))
	}
	f.FarmName = r.FormValue("name")
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

// WithFarmQuery filter based on FarmQuery
func (f FarmFilter) WithFarmQuery(q FarmQuery) FarmFilter {
	if len(q.FarmName) != 0 {
		f = f.WithName(q.FarmName)
	}
	if q.OwnerID != 0 {
		f = f.WithOwner(q.OwnerID)
	}
	return f

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

// Delete deletes one farm that match the filter
func (f FarmFilter) Delete(ctx context.Context, db *mongo.Database) (err error) {
	if f == nil {
		f = FarmFilter{}
	}
	col := db.Collection(FarmCollection)
	_, err = col.DeleteOne(ctx, f, options.Delete())
	return err
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

// FarmUpdate update an existing farm
func FarmUpdate(ctx context.Context, db *mongo.Database, id schema.ID, farm Farm) error {
	farm.ID = id

	if err := farm.Validate(); err != nil {
		return err
	}

	col := db.Collection(FarmCollection)
	f := FarmFilter{}.WithID(id)
	_, err := col.UpdateOne(ctx, f, bson.M{"$set": farm})
	return err
}
