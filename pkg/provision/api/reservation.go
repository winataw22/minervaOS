package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

func (a *Workloads) createOrUpdate(request *http.Request) (interface{}, mw.Response) {
	var deployment gridtypes.Deployment
	if err := json.NewDecoder(request.Body).Decode(&deployment); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := deployment.Valid(); err != nil {
		return nil, mw.BadRequest(err)
	}

	twinID := mw.TwinID(request.Context())
	if deployment.TwinID != twinID {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid user id in request body doesn't match http signature"))
	}

	if err := deployment.Verify(a.engine.Twins()); err != nil {
		return nil, mw.UnAuthorized(err)
	}

	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	action := a.engine.Provision
	if request.Method == http.MethodPut {
		action = a.engine.Update
	}

	err := action(ctx, deployment)

	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if errors.Is(err, provision.ErrDeploymentExists) {
		return nil, mw.Conflict(err)
	} else if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(err)
	} else if errors.Is(err, provision.ErrDeploymentUpgradeValidationError) {
		return nil, mw.BadRequest(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (a *Workloads) parseIDs(request *http.Request) (twin uint32, id uint64, err error) {
	twinVar := mux.Vars(request)["twin"]
	if len(twinVar) == 0 {
		return twin, id, fmt.Errorf("invalid twin id format")
	}
	idVar := mux.Vars(request)["id"]
	if len(idVar) == 0 {
		return twin, id, fmt.Errorf("invalid id format")
	}

	twinU, err := strconv.ParseUint(twinVar, 10, 32)
	if err != nil {
		return twin, id, err
	}

	idU, err := strconv.ParseUint(idVar, 10, 64)
	if err != nil {
		return twin, id, err
	}

	return uint32(twinU), idU, nil
}

func (a *Workloads) delete(request *http.Request) (interface{}, mw.Response) {

	twin, id, err := a.parseIDs(request)
	if err != nil {
		return nil, mw.BadRequest(err)
	}
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	twinID := mw.TwinID(request.Context())
	if twin != twinID {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid twin id in request url doesn't match http signature"))
	}

	err = a.engine.Deprovision(ctx, twin, id, "requested by user")
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (a *Workloads) get(request *http.Request) (interface{}, mw.Response) {
	twin, id, err := a.parseIDs(request)
	if err != nil {
		return nil, mw.BadRequest(err)
	}

	twinID := mw.TwinID(request.Context())
	if twin != twinID {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid twin id in request url doesn't match http signature"))
	}

	deployment, err := a.engine.Storage().Get(twin, id)
	if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(fmt.Errorf("workload not found"))
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return deployment, nil
}
