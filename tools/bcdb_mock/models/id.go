package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/threefoldtech/zos/pkg/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	//Counters is the counters collection in mongo
	Counters = "counters"
)

var (
	//ErrFailedToGetID is base error for generation failure
	ErrFailedToGetID = errors.New("failed to generate new id")
)

// NextID for a collection
func NextID(ctx context.Context, db *mongo.Database, collection string) (schema.ID, error) {
	result := db.Collection(Counters).FindOneAndUpdate(
		ctx,
		bson.M{"_id": collection},
		bson.M{"$inc": bson.M{"sequence": 1}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)

	if result.Err() != nil {
		return 0, result.Err()
	}
	var value struct {
		Sequence schema.ID `bson:"sequence"`
	}
	err := result.Decode(&value)
	return value.Sequence, err
}

//MustID must get next available ID, or panic with an error that has error.Is(err, ErrFailedToGetID) == true
func MustID(ctx context.Context, db *mongo.Database, collection string) schema.ID {
	id, err := NextID(ctx, db, collection)
	if err != nil {
		panic(fmt.Errorf("%w: %s", ErrFailedToGetID, err.Error()))
	}

	return id
}
