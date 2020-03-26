package directory

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	directory "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

func (s *NodeAPI) registerNode(r *http.Request) (interface{}, mw.Response) {
	log.Info().Msg("node register request received")

	defer r.Body.Close()

	var n directory.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		return nil, mw.BadRequest(err)
	}

	//make sure node can not set public config
	n.PublicConfig = nil
	db := mw.Database(r)
	if _, err := s.Add(r.Context(), db, n); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, mw.NotFound(fmt.Errorf("farm with id:%d does not exists", n.FarmId))
		}
		return nil, mw.Error(err)
	}

	log.Info().Msgf("node registered: %+v\n", n)

	return nil, mw.Created()
}

func (s *NodeAPI) nodeDetail(r *http.Request) (interface{}, mw.Response) {
	nodeID := mux.Vars(r)["node_id"]
	q := nodeQuery{}
	if err := q.Parse(r); err != nil {
		return nil, err
	}
	db := mw.Database(r)

	node, err := s.Get(r.Context(), db, nodeID, q.Proofs)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return node, nil
}

func (s *NodeAPI) listNodes(r *http.Request) (interface{}, mw.Response) {
	q := nodeQuery{}
	if err := q.Parse(r); err != nil {
		return nil, err
	}

	db := mw.Database(r)
	pager := models.PageFromRequest(r)
	nodes, total, err := s.List(r.Context(), db, q, pager)
	if err != nil {
		return nil, mw.Error(err)
	}

	pages := fmt.Sprintf("%d", models.Pages(pager, total))
	return nodes, mw.Ok().WithHeader("Pages", pages)
}

func (s *NodeAPI) registerCapacity(r *http.Request) (interface{}, mw.Response) {
	x := struct {
		Capacity   generated.ResourceAmount `json:"capacity,omitempty"`
		DMI        dmi.DMI                  `json:"dmi,omitempty"`
		Disks      capacity.Disks           `json:"disks,omitempty"`
		Hypervisor []string                 `json:"hypervisor,omitempty"`
	}{}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]
	db := mw.Database(r)

	if err := s.updateTotalCapacity(r.Context(), db, nodeID, x.Capacity); err != nil {
		return nil, mw.NotFound(err)
	}

	if err := s.StoreProof(r.Context(), db, nodeID, x.DMI, x.Disks, x.Hypervisor); err != nil {
		return nil, mw.Error(err)
	}

	return nil, nil
}

func (s *NodeAPI) registerIfaces(r *http.Request) (interface{}, mw.Response) {
	log.Debug().Msg("network interfaces register request received")

	defer r.Body.Close()

	var input []generated.Iface
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]
	db := mw.Database(r)
	if err := s.SetInterfaces(r.Context(), db, nodeID, input); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}

func (s *NodeAPI) configurePublic(r *http.Request) (interface{}, mw.Response) {
	var iface generated.PublicIface

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&iface); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]
	db := mw.Database(r)
	if err := s.SetPublicConfig(r.Context(), db, nodeID, iface); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}

func (s *NodeAPI) registerPorts(r *http.Request) (interface{}, mw.Response) {

	defer r.Body.Close()

	input := struct {
		Ports []uint `json:"ports"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	log.Debug().Uints("ports", input.Ports).Msg("wireguard ports received")

	nodeID := mux.Vars(r)["node_id"]
	db := mw.Database(r)
	if err := s.SetWGPorts(r.Context(), db, nodeID, input.Ports); err != nil {
		return nil, mw.NotFound(err)
	}

	return nil, nil
}

func (s *NodeAPI) updateUptimeHandler(r *http.Request) (interface{}, mw.Response) {
	defer r.Body.Close()

	input := struct {
		Uptime uint64 `json:"uptime"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]
	db := mw.Database(r)
	log.Debug().Str("node", nodeID).Uint64("uptime", input.Uptime).Msg("node uptime received")

	if err := s.updateUptime(r.Context(), db, nodeID, int64(input.Uptime)); err != nil {
		return nil, mw.NotFound(err)
	}

	return nil, nil
}

func (s *NodeAPI) updateReservedResources(r *http.Request) (interface{}, mw.Response) {
	//return nil, mw.Error(fmt.Errorf("not implemented"))
	defer r.Body.Close()

	var resources generated.ResourceAmount

	if err := json.NewDecoder(r.Body).Decode(&resources); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]

	db := mw.Database(r)
	if err := s.updateReservedCapacity(r.Context(), db, nodeID, resources); err != nil {
		return nil, mw.NotFound(err)
	}

	return nil, nil
}
