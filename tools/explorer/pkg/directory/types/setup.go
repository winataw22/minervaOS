package types

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Setup sets up indexes for types, must be called at least
// Onetime during the life time of the object
func Setup(ctx context.Context, db *mongo.Database) error {
	farm := db.Collection(FarmCollection)
	_, err := farm.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"name": 1},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize farm index")
	}

	node := db.Collection(NodeCollection)

	nodeIdexes := []mongo.IndexModel{
		{
			Keys:    bson.M{"node_id": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.M{"farm_id": 1},
		},
	}

	for _, x := range []string{"total_resources", "user_resources", "reserved_resources"} {
		for _, y := range []string{"cru", "mru", "hru", "sru"} {
			nodeIdexes = append(nodeIdexes, mongo.IndexModel{
				Keys: bson.M{fmt.Sprintf("%s.%s", x, y): 1},
			})
		}
	}

	_, err = node.Indexes().CreateMany(ctx, nodeIdexes)

	if err != nil {
		log.Error().Err(err).Msg("failed to initialize node index")
	}

	return err
}
