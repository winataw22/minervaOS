// This script is used to update the WalletAddresses field from the Farm type

package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/schema"
	generated "github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type oldType struct {
	ID              schema.ID                     `bson:"_id" json:"id"`
	ThreebotId      int64                         `bson:"threebot_id" json:"threebot_id"`
	IyoOrganization string                        `bson:"iyo_organization" json:"iyo_organization"`
	Name            string                        `bson:"name" json:"name"`
	WalletAddresses []string                      `bson:"wallet_addresses" json:"wallet_addresses"`
	Location        generated.Location            `bson:"location" json:"location"`
	Email           schema.Email                  `bson:"email" json:"email"`
	ResourcePrices  []generated.NodeResourcePrice `bson:"resource_prices" json:"resource_prices"`
	PrefixZero      schema.IPRange                `bson:"prefix_zero" json:"prefix_zero"`
}

func connectDB(ctx context.Context, connectionURI string) (*mongo.Client, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		return nil, err
	}

	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

func main() {
	app.Initialize()

	var (
		dbConf string
		name   string
	)

	flag.StringVar(&dbConf, "mongo", "mongodb://localhost:27017", "connection string to mongo database")
	flag.StringVar(&name, "name", "explorer", "database name")
	flag.Parse()

	ctx := context.TODO()

	client, err := connectDB(ctx, dbConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer client.Disconnect(ctx)

	db := client.Database(name, nil)
	col := db.Collection(types.FarmCollection)

	cur, err := col.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		old := oldType{}
		if err := cur.Decode(&old); err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msgf("start update %d %s", old.ID, old.Name)

		new := types.Farm{
			ID:              old.ID,
			ThreebotId:      old.ThreebotId,
			IyoOrganization: old.IyoOrganization,
			Name:            old.Name,
			WalletAddresses: make([]generated.WalletAddress, len(old.WalletAddresses)),
			Location:        old.Location,
			Email:           old.Email,
			ResourcePrices:  old.ResourcePrices,
			PrefixZero:      old.PrefixZero,
		}
		for i := range old.WalletAddresses {
			new.WalletAddresses[i] = generated.WalletAddress{
				Asset:   "TFChain",
				Address: old.WalletAddresses[i],
			}
		}

		f := types.FarmFilter{}
		f = f.WithID(old.ID)
		if _, err := col.UpdateOne(ctx, f, bson.M{"$set": new}); err != nil {
			log.Fatal().Err(err).Msgf("failed to update %d", old.ID)
		}
		log.Info().Msgf("updated %d %s", old.ID, old.Name)
	}
}
