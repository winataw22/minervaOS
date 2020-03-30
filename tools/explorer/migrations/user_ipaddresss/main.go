// This script is used to update the field from the User type from Ipaddr net.IP to Host string

package main

import (
	"context"
	"flag"
	"net"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/pkg/phonebook/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type oldType struct {
	ID          schema.ID `bson:"_id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Email       string    `bson:"email" json:"email"`
	Pubkey      string    `bson:"pubkey" json:"pubkey"`
	Ipaddr      net.IP    `bson:"ipaddr" json:"ipaddr"`
	Description string    `bson:"description" json:"description"`
	Signature   string    `bson:"signature" json:"signature"`
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
	col := db.Collection(types.UserCollection)

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

		new := types.User{
			ID:          old.ID,
			Name:        old.Name,
			Email:       old.Email,
			Pubkey:      old.Pubkey,
			Host:        old.Ipaddr.String(),
			Description: old.Description,
			Signature:   old.Signature,
		}

		f := types.UserFilter{}
		f = f.WithID(old.ID)
		if _, err := col.UpdateOne(ctx, f, bson.M{"$set": new}); err != nil {
			log.Fatal().Err(err).Msgf("failed to update %d", old.ID)
		}
		log.Info().Msgf("updated %d %s", old.ID, old.Name)
	}
}
