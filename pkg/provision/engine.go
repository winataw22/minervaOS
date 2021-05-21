package provision

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joncrlsn/dque"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/rs/zerolog/log"
)

// EngineOption interface
type EngineOption interface {
	apply(e *NativeEngine)
}

// WithJanitor sets a janitor for the engine.
// a janitor is executed periodically to clean up
// the deployed resources.
func WithJanitor(j Janitor) EngineOption {
	return &withJanitorOpt{j}
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

// WithRerunAll if set forces the engine to re-run all reservations
// on engine start.
func WithRerunAll(t bool) EngineOption {
	return &withRerunAll{t}
}

type jobOperation int

const (
	opProvision jobOperation = iota
	opDeprovision
	opUpdate
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
	twins    Twins
	admins   Twins
	order    []gridtypes.WorkloadType
	rerunAll bool
}

var _ Engine = (*NativeEngine)(nil)
var _ pkg.Provision = (*NativeEngine)(nil)

type withJanitorOpt struct {
	j Janitor
}

func (o *withJanitorOpt) apply(e *NativeEngine) {
	panic("not imple=nted")
	// e.janitor = o.j
}

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
	}
	// now move everything else
	for typ := range all {
		ordered = append(ordered, typ)
	}

	e.order = ordered
}

type withRerunAll struct {
	t bool
}

func (w *withRerunAll) apply(e *NativeEngine) {
	e.rerunAll = w.t
}

type nullKeyGetter struct{}

func (n *nullKeyGetter) GetKey(id uint32) (ed25519.PublicKey, error) {
	return nil, fmt.Errorf("null user key getter")
}

type engineKey struct{}
type deploymentKey struct{}

// GetEngine gets engine from context
func GetEngine(ctx context.Context) Engine {
	return ctx.Value(engineKey{}).(Engine)
}

// GetDeployment gets a copy of the current deployment
func GetDeployment(ctx context.Context) gridtypes.Deployment {
	return ctx.Value(deploymentKey{}).(gridtypes.Deployment)
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(storage Storage, provisioner Provisioner, root string, opts ...EngineOption) (*NativeEngine, error) {
	queue, err := dque.NewOrOpen("jobs", root, 512, func() interface{} { return &engineJob{} })
	if err != nil {
		// if this happens it means data types has been changed in that case we need
		// to clean up the queue and start over. unfortunately any un applied changes
		os.RemoveAll(filepath.Join(root, "jobs"))
		return nil, errors.Wrap(err, "failed to create job queue")
	}
	e := &NativeEngine{
		storage:     storage,
		provisioner: provisioner,
		queue:       queue,
		twins:       &nullKeyGetter{},
		admins:      &nullKeyGetter{},
		order:       gridtypes.Types(),
	}

	for _, opt := range opts {
		opt.apply(e)
	}

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
	if err := e.storage.Add(deployment); err != nil {
		return err
	}

	job := engineJob{
		Target: deployment,
		Op:     opProvision,
	}

	return e.queue.Enqueue(&job)
}

// Deprovision workload
func (e *NativeEngine) Deprovision(ctx context.Context, twin, id uint32, reason string) error {
	deployment, err := e.storage.Get(twin, id)
	if err != nil {
		return err
	}

	log.Debug().
		Uint32("twin", deployment.TwinID).
		Uint32("deployment", deployment.DeploymentID).
		Msg("schedule for deprovision")

	job := engineJob{
		Target: deployment,
		Op:     opDeprovision,
	}

	return e.queue.Enqueue(&job)
}

// Update workloads
func (e *NativeEngine) Update(ctx context.Context, update gridtypes.Deployment) error {
	deployment, err := e.storage.Get(update.TwinID, update.DeploymentID)
	if err != nil {
		return err
	}

	// this will just calculate the update
	// steps we run it here as a sort of validation
	// that this update is acceptable.
	_, err = deployment.Upgrade(&update)
	if err != nil {
		return errors.Wrap(ErrDeploymentUpgradeValidationError, err.Error())
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
		ctx := context.WithValue(root, deploymentKey{}, job.Target)
		switch job.Op {
		case opProvision:
			e.installDeployment(ctx, &job.Target)
			// update the state of the deployment in one go.
			if err := e.storage.Set(job.Target); err != nil {
				log.Error().Err(err).Msg("failed to set workload result")
			}
		case opDeprovision:
			e.uninstallDeployment(ctx, &job.Target, job.Message)
			if err := e.storage.Set(job.Target); err != nil {
				log.Error().Err(err).Msg("failed to set workload result")
			}
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
				log.Error().Err(err).Uint32("twin", job.Target.TwinID).Uint32("id", job.Target.DeploymentID).Msg("failed to get update procedure")
				break
			}

			e.uninstallDeployment(ctx, workloads(update.ToRemove), "deleted by an update")
			e.updateDeployment(ctx, workloads(update.ToUpdate))
			e.installDeployment(ctx, workloads(update.ToAdd))

			if err := e.storage.Set(job.Target); err != nil {
				log.Error().Err(err).Msg("failed to set workload result")
			}
		}

		e.queue.Dequeue()
	}
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
				log.Error().Err(err).Uint32("twin", twin).Uint32("id", id).Msg("failed to load deployment")
				continue
			}
			// unfortunately we have to inject this value here
			// since the boot runs outside the engine queue.

			ctx := context.WithValue(root, deploymentKey{}, dl)
			if e.installDeployment(ctx, &dl) {
				if err := e.storage.Set(dl); err != nil {
					log.Error().Err(err).Msg("failed to set workload result")
				}
			}
		}
	}

	return nil
}

func (e *NativeEngine) uninstallWorkload(ctx context.Context, wl *gridtypes.WorkloadWithID, reason string) error {
	err := e.provisioner.Decommission(ctx, wl)
	result := &gridtypes.Result{
		Error: reason,
		State: gridtypes.StateDeleted,
	}

	if err != nil {
		log.Error().Err(err).Stringer("global-id", wl.ID).Msg("failed to uninstall workload")
		result.State = gridtypes.StateError
		result.Error = errors.Wrapf(err, "error while decommission reservation because of: '%s'", result.Error).Error()
	}

	result.Created = gridtypes.Timestamp(time.Now().Unix())
	wl.Result = *result
	return err
}

func (e *NativeEngine) uninstallDeployment(ctx context.Context, getter gridtypes.WorkloadByTypeGetter, reason string) {
	for i := len(e.order) - 1; i >= 0; i-- {
		typ := e.order[i]

		workloads := getter.ByType(typ)
		for _, wl := range workloads {
			twin, deployment, name, _ := wl.ID.Parts()
			log := log.With().
				Uint32("twin", twin).
				Uint32("deployment", deployment).
				Str("name", name).
				Str("type", wl.Type.String()).
				Logger()

			log.Debug().Str("workload", wl.Name).Msg("de-provisioning")
			if wl.Result.State == gridtypes.StateDeleted {
				//nothing to do!
				continue
			}

			_ = e.uninstallWorkload(ctx, wl, reason)
		}
	}
}

func (e *NativeEngine) updateDeployment(ctx context.Context, getter gridtypes.WorkloadByTypeGetter) (changed bool) {
	for _, typ := range e.order {
		workloads := getter.ByType(typ)

		for _, wl := range workloads {
			// support redeployment by version update
			// if wl.Result.State == gridtypes.StateDeleted ||
			// 	wl.Result.State == gridtypes.StateError {
			// 	//nothing to do!
			// 	continue
			// }

			twin, deployment, name, _ := wl.ID.Parts()
			log := log.With().
				Uint32("twin", twin).
				Uint32("deployment", deployment).
				Str("name", name).
				Str("type", wl.Type.String()).
				Logger()

			log.Debug().Msg("provisioning")

			var result *gridtypes.Result
			var err error
			if e.provisioner.CanUpdate(ctx, wl.Type) {
				result, err = e.provisioner.Update(ctx, wl)
			} else {
				if err := e.provisioner.Decommission(ctx, wl); err != nil {
					log.Error().Err(err).Msg("failed to decomission workload")
				}

				result, err = e.provisioner.Provision(ctx, wl)
			}

			if err != nil {
				log.Error().Err(err).Msg("failed to deploy workload")
				result = &gridtypes.Result{
					Error: err.Error(),
					State: gridtypes.StateError,
				}
			}

			if result.State == gridtypes.StateError {
				log.Error().Str("error", result.Error).Msg("failed to deploy workload")
			}

			result.Created = gridtypes.Timestamp(time.Now().Unix())
			wl.Result = *result
			changed = true
		}
	}

	return
}

func (e *NativeEngine) installDeployment(ctx context.Context, getter gridtypes.WorkloadByTypeGetter) (changed bool) {
	for _, typ := range e.order {
		workloads := getter.ByType(typ)

		for _, wl := range workloads {
			// this workload is already deleted or in error state
			// we don't try again
			if wl.Result.State == gridtypes.StateDeleted ||
				wl.Result.State == gridtypes.StateError {
				//nothing to do!
				continue
			}

			twin, deployment, name, _ := wl.ID.Parts()
			log := log.With().
				Uint32("twin", twin).
				Uint32("deployment", deployment).
				Str("name", name).
				Str("type", wl.Type.String()).
				Logger()

			log.Debug().Msg("provisioning")
			result, err := e.provisioner.Provision(ctx, wl)
			if errors.Is(err, ErrDidNotChange) {
				continue
			}

			if err != nil {
				log.Error().Err(err).Msg("failed to deploy workload")
				result = &gridtypes.Result{
					Error: err.Error(),
					State: gridtypes.StateError,
				}
			}

			if result.State == gridtypes.StateError {
				log.Error().Str("error", result.Error).Msg("failed to deploy workload")
			}

			result.Created = gridtypes.Timestamp(time.Now().Unix())
			wl.Result = *result
			changed = true
		}
	}

	return
}

// Counters implements the zbus interface
func (e *NativeEngine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	//TODO: implement counters
	// this is probably need to be moved to
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			ch <- pkg.ProvisionCounters{}
		}
	}()

	return ch
}

// DecommissionCached implements the zbus interface
func (e *NativeEngine) DecommissionCached(id string, reason string) error {
	globalID := gridtypes.WorkloadID(id)
	twin, dlID, name, err := globalID.Parts()
	if err != nil {
		return err
	}
	dl, err := e.storage.Get(twin, dlID)
	if err != nil {
		return err
	}

	wl, err := dl.Get(name)
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
	ctx = context.WithValue(ctx, deploymentKey{}, dl)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	err = e.uninstallWorkload(ctx, wl, reason)

	if err := e.storage.Set(dl); err != nil {
		log.Error().Err(err).Msg("failed to set workload result")
	}

	return err
}

type workloads []*gridtypes.WorkloadWithID

func (l workloads) ByType(typ gridtypes.WorkloadType) []*gridtypes.WorkloadWithID {
	var results []*gridtypes.WorkloadWithID
	for _, wl := range l {
		if wl.Type == typ {
			results = append(results, wl)
		}
	}

	return results
}
