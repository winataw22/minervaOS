package directory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	directory "github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"

	"github.com/gorilla/mux"
)

func (s *FarmAPI) registerFarm(r *http.Request) (interface{}, mw.Response) {
	log.Info().Msg("farm register request received")

	db := mw.Database(r)
	defer r.Body.Close()

	var info directory.Farm
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := info.Validate(); err != nil {
		return nil, mw.BadRequest(err)
	}

	id, err := s.Add(r.Context(), db, info)
	if err != nil {
		return nil, mw.Error(err)
	}

	return struct {
		ID schema.ID `json:"id"`
	}{
		id,
	}, mw.Created()
}

func (s *FarmAPI) listFarm(r *http.Request) (interface{}, mw.Response) {
	q := directory.FarmQuery{}
	if err := q.Parse(r); err != nil {
		return nil, err
	}
	var filter directory.FarmFilter
	filter = filter.WithFarmQuery(q)
	db := mw.Database(r)

	pager := models.PageFromRequest(r)
	farms, total, err := s.List(r.Context(), db, filter, pager)
	if err != nil {
		return nil, mw.Error(err)
	}

	pages := fmt.Sprintf("%d", models.Pages(pager, total))
	return farms, mw.Ok().WithHeader("Pages", pages)
}

func (s *FarmAPI) getFarm(r *http.Request) (interface{}, mw.Response) {
	sid := mux.Vars(r)["farm_id"]

	id, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		return nil, mw.BadRequest(err)
	}

	db := mw.Database(r)

	farm, err := s.GetByID(r.Context(), db, id)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return farm, nil
}
