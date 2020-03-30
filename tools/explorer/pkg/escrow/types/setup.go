package types

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Setup sets up indexes for types, must be called at least
// Onetime during the life time of the object
func Setup(ctx context.Context, db *mongo.Database) error {
	escrow := db.Collection(EscrowCollection)
	_, err := escrow.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.M{"_id": 1},
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize reservation payment index")
		return err
	}

	addresses := db.Collection(AddressCollection)
	_, err = addresses.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"customer_tid": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.M{"address": 1},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize reservation payment index")
	}

	return err
}
