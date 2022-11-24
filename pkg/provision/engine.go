package provision

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/joncrlsn/dque"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/rs/zerolog/log"
)

// EngineOption interface
type EngineOption interface {
	apply(e *NativeEngine)
}

// WithTwins sets the user key getter on the
// engine
func WithTwins(g Twins) EngineOption {
	return &withUserKeyGetter{g}
}

// WithAdmins sets the admins key getter on the
// engine
func WithAdmins(g Twins) EngineOption {
	return &withAdminsKeyGetter{g}
}

// WithStartupOrder forces a specific startup order of types
// any type that is not listed in this list, will get started
// in an nondeterministic order
func WithStartupOrder(t ...gridtypes.WorkloadType) EngineOption {
	return &withStartupOrder{t}
}

// WithSubstrate sets the substrate client. If set it will
// be used by the engine to fetch (and validate) the deployment contract
// then contract with be available on the deployment context
func WithSubstrate(node uint32, sub substrate.Manager) EngineOption {
	return &withSubstrate{node, sub}
}

// WithRerunAll if set forces the engine to re-run all reservations
// on engine start.
func WithRerunAll(t bool) EngineOption {
	return &withRerunAll{t}
}

type Callback func(twin uint32, contract uint64, delete bool)

// WithCallback sets a callback that is called when a deployment is being Created, Updated, Or Deleted
// The handler then can use the id to get current "state" of the deployment from storage and
// take proper action. A callback must not block otherwise the engine operation will get blocked
func WithCallback(cb Callback) EngineOption {
	return &withCallback{cb}
}

type jobOperation int

const (
	// opProvision installs a deployment
	opProvision jobOperation = iota
	// removes a deployment
	opDeprovision
	// deletes a deployment
	opUpdate
	// opProvisionNoValidation is used to reinstall
	// a deployment on node reboot without validating
	// against the chain again because 1) validation
	// has already been done on first installation
	// 2) hash is not granteed to match because of the
	// order of the workloads doesn't have to match
	// the one sent by the user
	opProvisionNoValidation
	// opPause, pauses a deployment
	opPause
	// opResume resumes a deployment
	opResume
)

// engineJob is a persisted job instance that is
// stored in a queue. the queue uses a GOB encoder
// so please make sure that edits to this struct is
// ONLY by adding new fields or deleting older fields
// but never rename or change the type of a field.
type engineJob struct {
	Op      jobOperation
	Target  gridtypes.Deployment
	Source  *gridtypes.Deployment
	Message string
}

// NativeEngine is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type NativeEngine struct {
	storage     Storage
	provisioner Provisioner

	queue *dque.DQue

	//options
	// janitor Janitor
	twins     Twins
	admins    Twins
	order     []gridtypes.WorkloadType
	typeIndex map[gridtypes.WorkloadType]int
	rerunAll  bool
	//substrate specific attributes
	nodeID   uint32
	sub      substrate.Manager
	callback Callback
}

var _ Engine = (*NativeEngine)(nil)
var _ pkg.Provision = (*NativeEngine)(nil)

type withUserKeyGetter struct {
	g Twins
}

func (o *withUserKeyGetter) apply(e *NativeEngine) {
	e.twins = o.g
}

type withAdminsKeyGetter struct {
	g Twins
}

func (o *withAdminsKeyGetter) apply(e *NativeEngine) {
	e.admins = o.g
}

type withSubstrate struct {
	nodeID uint32
	sub    substrate.Manager
}

func (o *withSubstrate) apply(e *NativeEngine) {
	e.nodeID = o.nodeID
	e.sub = o.sub
}

type withStartupOrder struct {
	o []gridtypes.WorkloadType
}

func (w *withStartupOrder) apply(e *NativeEngine) {
	all := make(map[gridtypes.WorkloadType]struct{})
	for _, typ := range e.order {
		all[typ] = struct{}{}
	}
	ordered := make([]gridtypes.WorkloadType, 0, len(all))
	for _, typ := range w.o {
		if _, ok := all[typ]; !ok {
			panic(fmt.Sprintf("type '%s' is not registered", typ))
		}
		delete(all, typ)
		ordered = append(ordered, typ)
		e.typeIndex[typ] = len(ordered)
	}
	// now move everything else
	for typ := range all {
		ordered = append(ordered, typ)
		e.typeIndex[typ] = len(ordered)
	}

	e.order = ordered
}

type withRerunAll struct {
	t bool
}

func (w *withRerunAll) apply(e *NativeEngine) {
	e.rerunAll = w.t
}

type withCallback struct {
	cb Callback
}

func (w *withCallback) apply(e *NativeEngine) {
	e.callback = w.cb
}

type nullKeyGetter struct{}

func (n *nullKeyGetter) GetKey(id uint32) ([]byte, error) {
	return nil, fmt.Errorf("null user key getter")
}

type engineKey struct{}
type deploymentKey struct{}
type deploymentValue struct {
	twin       uint32
	deployment uint64
}
type contractKey struct{}
type substrateKey struct{}

// GetEngine gets engine from context
func GetEngine(ctx context.Context) Engine {
	return ctx.Value(engineKey{}).(Engine)
}

// GetDeploymentID gets twin and deployment ID for current deployment
func GetDeploymentID(ctx context.Context) (twin uint32, deployment uint64) {
	values := ctx.Value(deploymentKey{}).(deploymentValue)
	return values.twin, values.deployment
}

// GetDeployment gets a copy of the current deployment with latest state
func GetDeployment(ctx context.Context) (gridtypes.Deployment, error) {
	// we store the pointer on the context so changed to deployment object
	// actually reflect into the value.
	engine := GetEngine(ctx)
	twin, deployment := GetDeploymentID(ctx)

	// BUT we always return a copy so caller of GetDeployment can NOT manipulate
	// other attributed on the object.
	return engine.Storage().Get(twin, deployment)
}

// GetWorkload get the last state of the workload for the current deployment
func GetWorkload(ctx context.Context, name gridtypes.Name) (gridtypes.WorkloadWithID, error) {
	// we store the pointer on the context so changed to deployment object
	// actually reflect into the value.
	engine := GetEngine(ctx)
	twin, deployment := GetDeploymentID(ctx)

	// BUT we always return a copy so caller of GetDeployment can NOT manipulate
	// other attributed on the object.
	wl, err := engine.Storage().Current(twin, deployment, name)
	if err != nil {
		return gridtypes.WorkloadWithID{}, err
	}

	return gridtypes.WorkloadWithID{
		Workload: &wl,
		ID:       gridtypes.NewUncheckedWorkloadID(twin, deployment, name),
	}, nil
}

func withDeployment(ctx context.Context, twin uint32, deployment uint64) context.Context {
	return context.WithValue(ctx, deploymentKey{}, deploymentValue{twin, deployment})
}

// GetContract of deployment. panics if engine has no substrate set.
func GetContract(ctx context.Context) substrate.NodeContract {
	return ctx.Value(contractKey{}).(substrate.NodeContract)
}

func withContract(ctx context.Context, contract substrate.NodeContract) context.Context {
	return context.WithValue(ctx, contractKey{}, contract)
}

// GetSubstrate if engine has substrate set, panics otherwise
func GetSubstrate(ctx context.Context) substrate.Manager {
	return ctx.Value(substrateKey{}).(substrate.Manager)
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(storage Storage, provisioner Provisioner, root string, opts ...EngineOption) (*NativeEngine, error) {
	e := &NativeEngine{
		storage:     storage,
		provisioner: provisioner,
		twins:       &nullKeyGetter{},
		admins:      &nullKeyGetter{},
		order:       gridtypes.Types(),
		typeIndex:   make(map[gridtypes.WorkloadType]int),
	}

	for _, opt := range opts {
		opt.apply(e)
	}

	if e.rerunAll {
		os.RemoveAll(filepath.Join(root, "jobs"))
	}

	queue, err := dque.NewOrOpen("jobs", root, 512, func() interface{} { return &engineJob{} })
	if err != nil {
		// if this happens it means data types has been changed in that case we need
		// to clean up the queue and start over. unfortunately any un applied changes
		os.RemoveAll(filepath.Join(root, "jobs"))
		return nil, errors.Wrap(err, "failed to create job queue")
	}

	e.queue = queue
	return e, nil
}

// Storage returns
func (e *NativeEngine) Storage() Storage {
	return e.storage
}

// Twins returns twins db
func (e *NativeEngine) Twins() Twins {
	return e.twins
}

// Admins returns admins db
func (e *NativeEngine) Admins() Twins {
	return e.admins
}

// Provision workload
func (e *NativeEngine) Provision(ctx context.Context, deployment gridtypes.Deployment) error {
	if deployment.Version != 0 {
		return errors.Wrap(ErrInvalidVersion, "expected version to be 0 on deployment creation")
	}

	if err := e.storage.Create(deployment); err != nil {
		return err
	}

	job := engineJob{
		Target: deployment,
		Op:     opProvision,
	}

	return e.queue.Enqueue(&job)
}

// Pause deployment
func (e *NativeEngine) Pause(ctx context.Context, twin uint32, id uint64) error {
	deployment, err := e.storage.Get(twin, id)
	if err != nil {
		return err
	}

	log.Info().
		Uint32("twin", deployment.TwinID).
		Uint64("contract", deployment.ContractID).
		Msg("schedule for pausing")

	job := engineJob{
		Target: deployment,
		Op:     opPause,
	}

	return e.queue.Enqueue(&job)
}

// Resume deployment
func (e *NativeEngine) Resume(ctx context.Context, twin uint32, id uint64) error {
	deployment, err := e.storage.Get(twin, id)
	if err != nil {
		return err
	}

	log.Info().
		Uint32("twin", deployment.TwinID).
		Uint64("contract", deployment.ContractID).
		Msg("schedule for resuming")

	job := engineJob{
		Target: deployment,
		Op:     opResume,
	}

	return e.queue.Enqueue(&job)
}

// Deprovision workload
func (e *NativeEngine) Deprovision(ctx context.Context, twin uint32, id uint64, reason string) error {
	deployment, err := e.storage.Get(twin, id)
	if err != nil {
		return err
	}

	log.Info().
		Uint32("twin", deployment.TwinID).
		Uint64("contract", deployment.ContractID).
		Str("reason", reason).
		Msg("schedule for deprovision")

	job := engineJob{
		Target:  deployment,
		Op:      opDeprovision,
		Message: reason,
	}

	return e.queue.Enqueue(&job)
}

// Update workloads
func (e *NativeEngine) Update(ctx context.Context, update gridtypes.Deployment) error {
	deployment, err := e.storage.Get(update.TwinID, update.ContractID)
	if err != nil {
		return err
	}

	// this will just calculate the update
	// steps we run it here as a sort of validation
	// that this update is acceptable.
	upgrades, err := deployment.Upgrade(&update)
	if err != nil {
		return errors.Wrap(ErrDeploymentUpgradeValidationError, err.Error())
	}

	for _, op := range upgrades {
		if op.Op == gridtypes.OpUpdate {
			if !e.provisioner.CanUpdate(ctx, op.WlID.Type) {
				return errors.Wrapf(
					ErrDeploymentUpgradeValidationError,
					"workload '%s' does not support upgrade",
					op.WlID.Type.String())
			}
		}
	}

	// fields to update in storage
	fields := []Field{
		VersionField{update.Version},
		SignatureRequirementField{update.SignatureRequirement},
	}

	if deployment.Description != update.Description {
		fields = append(fields, DescriptionField{update.Description})
	}
	if deployment.Metadata != update.Metadata {
		fields = append(fields, MetadataField{update.Metadata})
	}
	// update deployment fields, workloads will then can get updated separately
	if err := e.storage.Update(update.TwinID, update.ContractID, fields...); err != nil {
		return errors.Wrap(err, "failed to update deployment data")
	}
	// all is okay we can push the job
	job := engineJob{
		Op:     opUpdate,
		Target: update,
		Source: &deployment,
	}

	return e.queue.Enqueue(&job)
}

// Run starts reader reservation from the Source and handle them
func (e *NativeEngine) Run(root context.Context) error {
	defer e.queue.Close()

	root = context.WithValue(root, engineKey{}, e)

	if e.rerunAll {
		if err := e.boot(root); err != nil {
			log.Error().Err(err).Msg("error while setting up")
		}
	}

	for {
		obj, err := e.queue.PeekBlock()
		if err != nil {
			log.Error().Err(err).Msg("failed to check job queue")
			<-time.After(2 * time.Second)
			continue
		}

		job := obj.(*engineJob)
		ctx := withDeployment(root, job.Target.TwinID, job.Target.ContractID)
		l := log.With().
			Uint32("twin", job.Target.TwinID).
			Uint64("contract", job.Target.ContractID).
			Logger()

		// contract validation
		// this should ONLY be done on provosion and update operation
		if job.Op == opProvision ||
			job.Op == opUpdate ||
			job.Op == opProvisionNoValidation {
			// otherwise, contract validation is needed
			ctx, err = e.contract(ctx, &job.Target, job.Op == opProvisionNoValidation)
			if err != nil {
				l.Error().Err(err).Msg("contact validation fails")
				//job.Target.SetError(err)
				if err := e.storage.Error(job.Target.TwinID, job.Target.ContractID, err); err != nil {
					l.Error().Err(err).Msg("failed to set deployment global error")
				}
				_, _ = e.queue.Dequeue()

				continue
			}

			l.Debug().Msg("contact validation pass")
		}

		switch job.Op {
		case opProvisionNoValidation:
			fallthrough
		case opProvision:
			e.installDeployment(ctx, &job.Target)
		case opDeprovision:
			e.uninstallDeployment(ctx, &job.Target, job.Message)
		case opPause:
			e.lockDeployment(ctx, &job.Target)
		case opResume:
			e.unlockDeployment(ctx, &job.Target)
		case opUpdate:
			// update is tricky because we need to work against
			// 2 versions of the object. Once that reflects the current state
			// and the new one that is the target state but it does not know
			// the current state of already deployed workloads
			// so (1st) we need to get the difference
			// this call will return 3 lists
			// - things to remove
			// - things to add
			// - things to update (not supported atm)
			// - things that is not in any of the 3 lists are basically stay as is
			// the call will also make sure the Result of those workload in both the (did not change)
			// and update to reflect the current result on those workloads.
			update, err := job.Source.Upgrade(&job.Target)
			if err != nil {
				l.Error().Err(err).Msg("failed to get update procedure")
				break
			}
			e.updateDeployment(ctx, update)
		}

		_, err = e.queue.Dequeue()
		if err != nil {
			l.Error().Err(err).Msg("failed to dequeue job")
		}

		e.safeCallback(&job.Target, job.Op == opDeprovision)
	}
}

func (e *NativeEngine) safeCallback(d *gridtypes.Deployment, delete bool) {
	if e.callback == nil {
		return
	}
	// in case callback panics we don't want to kill the engine
	defer func() {
		if err := recover(); err != nil {
			log.Error().Msgf("panic while processing callback: %v", err)
		}
	}()

	e.callback(d.TwinID, d.ContractID, delete)
}

// contract validates and injects the deployment contracts is substrate is configured
// for this instance of the provision engine. If noValidation is set contracts checks is skipped
func (e *NativeEngine) contract(ctx context.Context, dl *gridtypes.Deployment, noValidation bool) (context.Context, error) {
	if e.sub == nil {
		return ctx, fmt.Errorf("substrate is not configured in engine")
	}

	ctx = context.WithValue(ctx, substrateKey{}, e.sub)
	// if substrate is set. we need to get contract
	sub, err := e.sub.Substrate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to chain")
	}
	defer sub.Close()
	contract, err := sub.GetContract(uint64(dl.ContractID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deployment contract")
	}

	if !contract.ContractType.IsNodeContract {
		return nil, fmt.Errorf("invalid contract type, expecting node contract")
	}
	ctx = withContract(ctx, contract.ContractType.NodeContract)

	if noValidation {
		return ctx, nil
	}

	if uint32(contract.ContractType.NodeContract.Node) != e.nodeID {
		return nil, fmt.Errorf("invalid node address in contract")
	}

	hash, err := dl.ChallengeHash()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute deployment hash")
	}

	if contract.ContractType.NodeContract.DeploymentHash.String() != hex.EncodeToString(hash) {
		return nil, fmt.Errorf("contract hash does not match deployment hash")
	}

	return ctx, nil
}

// boot will make sure to re-deploy all stored reservation
// on boot.
func (e *NativeEngine) boot(root context.Context) error {
	storage := e.Storage()
	twins, err := storage.Twins()
	if err != nil {
		return errors.Wrap(err, "failed to list twins")
	}
	for _, twin := range twins {
		ids, err := storage.ByTwin(twin)
		if err != nil {
			log.Error().Err(err).Uint32("twin", twin).Msg("failed to list deployments for twin")
			continue
		}

		for _, id := range ids {
			dl, err := storage.Get(twin, id)
			if err != nil {
				log.Error().Err(err).Uint32("twin", twin).Uint64("id", id).Msg("failed to load deployment")
				continue
			}
			// unfortunately we have to inject this value here
			// since the boot runs outside the engine queue.

			if !dl.IsActive() {
				continue
			}

			job := engineJob{
				Target: dl,
				Op:     opProvisionNoValidation,
			}

			if err := e.queue.Enqueue(&job); err != nil {
				log.Error().
					Err(err).
					Uint32("twin", dl.TwinID).
					Uint64("dl", dl.ContractID).
					Msg("failed to queue deployment for processing")
			}
		}
	}

	return nil
}

func (e *NativeEngine) uninstallWorkload(ctx context.Context, wl *gridtypes.WorkloadWithID, reason string) error {
	twin, deployment, name, _ := wl.ID.Parts()
	log := log.With().
		Uint32("twin", twin).
		Uint64("deployment", deployment).
		Stringer("name", name).
		Str("type", wl.Type.String()).
		Logger()

	_, err := e.storage.Current(twin, deployment, name)
	if errors.Is(err, ErrWorkloadNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	log.Debug().Str("workload", string(wl.Name)).Msg("de-provisioning")

	result := gridtypes.Result{
		State: gridtypes.StateDeleted,
		Error: reason,
	}
	if err := e.provisioner.Deprovision(ctx, wl); err != nil {
		log.Error().Err(err).Stringer("id", wl.ID).Msg("failed to uninstall workload")
		result.State = gridtypes.StateError
		result.Error = err.Error()
	}

	result.Created = gridtypes.Timestamp(time.Now().Unix())

	if err := e.storage.Transaction(twin, deployment, wl.Workload.WithResults(result)); err != nil {
		return err
	}

	if result.State == gridtypes.StateDeleted {
		return e.storage.Remove(twin, deployment, name)
	}

	return nil
}

func (e *NativeEngine) installWorkload(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	// this workload is already deleted or in error state
	// we don't try again
	twin, deployment, name, _ := wl.ID.Parts()

	current, err := e.storage.Current(twin, deployment, name)
	if errors.Is(err, ErrWorkloadNotExist) {
		// this can happen if installWorkload was called upon a deployment update operation
		// so this is a totally new workload that was not part of the original deployment
		// hence a call to Add is needed
		if err := e.storage.Add(twin, deployment, *wl.Workload); err != nil {
			return errors.Wrap(err, "failed to add workload to storage")
		}
	} else if err != nil {
		// another error
		return errors.Wrapf(err, "failed to get last transaction for '%s'", wl.ID.String())
	} else {
		// workload exists, but we trying to re-install it so this might be
		// after a reboot. hence we need to check last state.
		// if it has been deleted,  error state, we do nothing.
		// otherwise, we-reinstall it
		if current.Result.State.IsAny(gridtypes.StateDeleted, gridtypes.StateError) {
			//nothing to do!
			return nil
		}
	}

	log := log.With().
		Uint32("twin", twin).
		Uint64("deployment", deployment).
		Stringer("name", wl.Name).
		Str("type", wl.Type.String()).
		Logger()

	log.Debug().Msg("provisioning")
	result, err := e.provisioner.Provision(ctx, wl)
	if errors.Is(err, ErrNoActionNeeded) {
		// workload already exist, so no need to create a new transaction
		return nil
	} else if err != nil {
		result.Created = gridtypes.Now()
		result.State = gridtypes.StateError
		result.Error = err.Error()
	}

	if result.State == gridtypes.StateError {
		log.Error().Str("error", result.Error).Msg("failed to deploy workload")
	}

	return e.storage.Transaction(
		twin,
		deployment,
		wl.Workload.WithResults(result))
}

func (e *NativeEngine) updateWorkload(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	twin, deployment, name, _ := wl.ID.Parts()
	log := log.With().
		Uint32("twin", twin).
		Uint64("deployment", deployment).
		Stringer("name", name).
		Str("type", wl.Type.String()).
		Logger()

	log.Debug().Msg("provisioning")

	var result gridtypes.Result
	var err error
	if e.provisioner.CanUpdate(ctx, wl.Type) {
		result, err = e.provisioner.Update(ctx, wl)
	} else {
		// deprecated. We should never update resources by decomission and then provsion
		// the check in Update method should prevent this
		// #unreachable
		err = fmt.Errorf("can not update this workload type")
	}

	if errors.Is(err, ErrNoActionNeeded) {
		return nil
	} else if err != nil {
		return err
	}

	return e.storage.Transaction(twin, deployment, wl.Workload.WithResults(result))
}

func (e *NativeEngine) lockWorkload(ctx context.Context, wl *gridtypes.WorkloadWithID, lock bool) error {
	// this workload is already deleted or in error state
	// we don't try again
	twin, deployment, name, _ := wl.ID.Parts()

	current, err := e.storage.Current(twin, deployment, name)
	if err != nil {
		// another error
		return errors.Wrapf(err, "failed to get last transaction for '%s'", wl.ID.String())
	} else {
		if !current.Result.State.IsOkay() {
			//nothing to do! it's either in error state or something else.
			return nil
		}
	}

	log := log.With().
		Uint32("twin", twin).
		Uint64("deployment", deployment).
		Stringer("name", wl.Name).
		Str("type", wl.Type.String()).
		Bool("lock", lock).
		Logger()

	log.Debug().Msg("setting locking on workload")
	action := e.provisioner.Resume
	if lock {
		action = e.provisioner.Pause
	}
	result, err := action(ctx, wl)
	if errors.Is(err, ErrNoActionNeeded) {
		// workload already exist, so no need to create a new transaction
		return nil
	} else if err != nil {
		return err
	}

	if result.State == gridtypes.StateError {
		log.Error().Str("error", result.Error).Msg("failed to set locking on workload")
	}

	return e.storage.Transaction(
		twin,
		deployment,
		wl.Workload.WithResults(result))
}

func (e *NativeEngine) uninstallDeployment(ctx context.Context, dl *gridtypes.Deployment, reason string) {
	var errors bool
	for i := len(e.order) - 1; i >= 0; i-- {
		typ := e.order[i]

		workloads := dl.ByType(typ)
		for _, wl := range workloads {
			if err := e.uninstallWorkload(ctx, wl, reason); err != nil {
				errors = true
				log.Error().Err(err).Stringer("id", wl.ID).Msg("failed to un-install workload")
			}
		}
	}

	if errors {
		return
	}

	if err := e.storage.Delete(dl.TwinID, dl.ContractID); err != nil {
		log.Error().Err(err).
			Uint32("twin", dl.TwinID).
			Uint64("contract", dl.ContractID).
			Msg("failed to delete deployment")
	}

}

func (e *NativeEngine) installDeployment(ctx context.Context, getter gridtypes.WorkloadGetter) {
	for _, typ := range e.order {
		workloads := getter.ByType(typ)

		for _, wl := range workloads {
			if err := e.installWorkload(ctx, wl); err != nil {
				log.Error().Err(err).Stringer("id", wl.ID).Msg("failed to install workload")
			}
		}
	}
}

func (e *NativeEngine) lockDeployment(ctx context.Context, getter gridtypes.WorkloadGetter) {
	for i := len(e.order) - 1; i >= 0; i-- {
		typ := e.order[i]

		workloads := getter.ByType(typ)

		for _, wl := range workloads {
			if err := e.lockWorkload(ctx, wl, true); err != nil {
				log.Error().Err(err).Stringer("id", wl.ID).Msg("failed to lock workload")
			}
		}
	}
}

func (e *NativeEngine) unlockDeployment(ctx context.Context, getter gridtypes.WorkloadGetter) {
	for _, typ := range e.order {
		workloads := getter.ByType(typ)

		for _, wl := range workloads {
			if err := e.lockWorkload(ctx, wl, false); err != nil {
				log.Error().Err(err).Stringer("id", wl.ID).Msg("failed to unlock workload")
			}
		}
	}
}

// sortOperations sortes the operations, removes first in reverse type order, then upgrades/creates in type order
func (e *NativeEngine) sortOperations(ops []gridtypes.UpgradeOp) {
	// maps an operation to an integer, less comes first in sorting
	opMap := func(op gridtypes.UpgradeOp) int {
		if op.Op == gridtypes.OpRemove {
			// removes are negative (typeIndex starts from 1) so they are always before creations/updates
			// negated to apply in reverse order
			return -e.typeIndex[op.WlID.Type]
		} else {
			// updates/creates are considered the same
			return e.typeIndex[op.WlID.Type]
		}
	}
	sort.SliceStable(ops, func(i, j int) bool {
		return opMap(ops[i]) < opMap(ops[j])
	})
}

func (e *NativeEngine) updateDeployment(ctx context.Context, ops []gridtypes.UpgradeOp) (changed bool) {
	e.sortOperations(ops)
	for _, op := range ops {
		var err error
		switch op.Op {
		case gridtypes.OpRemove:
			err = e.uninstallWorkload(ctx, op.WlID, "deleted by an update")
		case gridtypes.OpAdd:
			err = e.installWorkload(ctx, op.WlID)
		case gridtypes.OpUpdate:
			err = e.updateWorkload(ctx, op.WlID)
		}

		if err != nil {
			log.Error().Err(err).Stringer("id", op.WlID.ID).Stringer("operation", op.Op).Msg("error while updating deployment")
		}
	}
	return
}

// DecommissionCached implements the zbus interface
func (e *NativeEngine) DecommissionCached(id string, reason string) error {
	globalID := gridtypes.WorkloadID(id)
	twin, dlID, name, err := globalID.Parts()
	if err != nil {
		return err
	}
	wl, err := e.storage.Current(twin, dlID, name)
	if err != nil {
		return err
	}

	if wl.Result.State == gridtypes.StateDeleted ||
		wl.Result.State == gridtypes.StateError {
		//nothing to do!
		return nil
	}

	//to bad we have to repeat this here
	ctx := context.WithValue(context.Background(), engineKey{}, e)
	ctx = withDeployment(ctx, twin, dlID)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	err = e.uninstallWorkload(ctx, &gridtypes.WorkloadWithID{Workload: &wl, ID: globalID},
		fmt.Sprintf("workload decommissioned by system, reason: %s", reason),
	)

	return err
}
