package directory

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/schema"
	directory "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FarmAPI holds farm releated handlers
type FarmAPI struct{}

// List farms
// TODO: add paging arguments
func (s *FarmAPI) List(ctx context.Context, db *mongo.Database, tid int64, opts ...*options.FindOptions) ([]directory.Farm, int64, error) {
	var filter directory.FarmFilter

	if tid != 0 {
		log.Info().Msgf("with owner id %d", tid)
		filter = filter.WithOwner(tid)
	}

	cur, err := filter.Find(ctx, db, opts...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list farms")
	}
	defer cur.Close(ctx)
	out := []directory.Farm{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, errors.Wrap(err, "failed to load farm list")
	}

	count, err := filter.Count(ctx, db)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count entries in farms collection")
	}

	return out, count, nil
}

// GetByName gets a farm by name
func (s *FarmAPI) GetByName(ctx context.Context, db *mongo.Database, name string) (directory.Farm, error) {
	var filter directory.FarmFilter
	filter = filter.WithName(name)

	return filter.Get(ctx, db)
}

// GetByID gets a farm by ID
func (s *FarmAPI) GetByID(ctx context.Context, db *mongo.Database, id int64) (directory.Farm, error) {
	var filter directory.FarmFilter
	filter = filter.WithID(schema.ID(id))

	return filter.Get(ctx, db)
}

// Add add farm to store
// TODO: support update farm information ?
func (s *FarmAPI) Add(ctx context.Context, db *mongo.Database, farm directory.Farm) (schema.ID, error) {
	return directory.FarmCreate(ctx, db, farm)
}
