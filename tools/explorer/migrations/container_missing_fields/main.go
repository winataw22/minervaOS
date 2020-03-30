// This script is used to update the missing field from the ContainerCapacity type
// It fields the empty value with the value from the JSON field on the reservation type

package main

import (
	"context"
	"encoding/json"
	"flag"
	"reflect"

	generated "github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/tools/explorer/pkg/workloads/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
	col := db.Collection(types.ReservationCollection)

	cur, err := col.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		r := types.Reservation{}
		if err := cur.Decode(&r); err != nil {
			log.Error().Err(err).Send()
			continue
		}

		var data generated.ReservationData

		if err := json.Unmarshal([]byte(r.Json), &data); err != nil {
			log.Fatal().Err(err).Msg("invalid json data on reservation")
		}

		if !reflect.DeepEqual(r.DataReservation, data) {
			log.Info().Msgf("start update %d", r.ID)

			for i := range r.DataReservation.Containers {
				r.DataReservation.Containers[i].Capacity.Cpu = data.Containers[i].Capacity.Cpu
				r.DataReservation.Containers[i].Capacity.Memory = data.Containers[i].Capacity.Memory
				for y := range r.DataReservation.Containers[i].NetworkConnection {
					r.DataReservation.Containers[i].NetworkConnection[y].PublicIp6 = data.Containers[i].NetworkConnection[y].PublicIp6
				}
			}

			if !reflect.DeepEqual(r.DataReservation, data) {
				log.Error().Msgf("\n%+v\n%+v", r.DataReservation, data)
				log.Fatal().Msg("json data does not match the reservation data")
			}

			filter := bson.D{}
			filter = append(filter, bson.E{Key: "_id", Value: r.ID})

			res, err := col.UpdateOne(ctx, filter, bson.M{"$set": r})
			if err != nil {
				log.Fatal().Err(err).Msgf("failed to update %d", r.ID)
			}
			if res.ModifiedCount == 1 {
				log.Info().Msgf("updated %d", r.ID)
			} else {
				log.Error().Msgf("no document updated %d", r.ID)
			}

		}

	}
}
