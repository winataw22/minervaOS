package provisiond

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"github.com/joncrlsn/dque"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/farmer"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision/storage"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	every = 5 * 60 // 5 minutes
)

type many []error

func (m many) Error() string {
	return m.WithPrefix("")
}

func (m many) WithPrefix(p string) string {
	var buf bytes.Buffer
	for _, err := range m {
		if buf.Len() > 0 {
			buf.WriteRune('\n')
		}
		if err, ok := err.(many); ok {
			buf.WriteString(err.WithPrefix(p + " "))
			continue
		}

		buf.WriteString(err.Error())
	}

	return buf.String()
}

func (m many) append(err error) many {
	return append(m, err)
}

func (m many) AsError() error {
	if len(m) == 0 {
		return nil
	}

	return m
}

// Reporter structure
type Reporter struct {
	identity *stubs.IdentityManagerStub
	storage  *storage.Fs
	queue    *dque.DQue
	farmer   *farmer.Client
	nodeID   string
}

func reportBuilder() interface{} {
	return &farmer.Report{}
}

// NewReported creates a new capacity reporter
func NewReported(store *storage.Fs, identity *stubs.IdentityManagerStub, root string) (*Reporter, error) {
	env, err := environment.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runtime environment")
	}

	fm, err := env.FarmerClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create farmer client")
	}

	queue, err := dque.NewOrOpen("consumption", root, 1024, reportBuilder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup report persisted queue")
	}

	return &Reporter{
		storage:  store,
		identity: identity,
		queue:    queue,
		farmer:   fm,
		nodeID:   identity.NodeID(context.TODO()).Identity(),
	}, nil
}

func (r *Reporter) pushOne() error {
	item, err := r.queue.PeekBlock()
	if err != nil {
		return errors.Wrap(err, "failed to peek into capacity queue. #properlyfatal")
	}

	report := item.(*farmer.Report)

	// DEBUG
	log.Debug().Int64("timestamp", report.Timestamp).Msg("sending capacity report")
	if err := r.farmer.NodeReport(r.nodeID, *report); err != nil {
		return errors.Wrap(err, "failed to publish consumption report")
	}

	// only removed if report is reported to
	// farmer
	// remove item from queue
	_, err = r.queue.Dequeue()

	return err
}

func (r *Reporter) pusher() {
	for {
		if err := r.pushOne(); err != nil {
			log.Error().Err(err).Msg("error while processing capacity report")
			<-time.After(3 * time.Second)
		}

		log.Debug().Msg("capacity report pushed to farmer")
	}
}

// Run runs the reporter
func (r *Reporter) Run(ctx context.Context) error {
	// go over all user reservations
	// take into account the following:
	// every is in seconds.

	go r.pusher()

	ticker := time.NewTicker(every * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			// align time.
			u := t.Unix()
			u = (u / every) * every
			// so any reservation that is deleted but this
			// happened 'after' this time stamp is still
			// considered as consumption because it lies in the current
			// 5 minute slot.
			// but if it was stopped before that timestamp, then it will
			// not be counted.
			report, err := r.collect(time.Unix(u, 0))
			if err != nil {
				log.Error().Err(err).Msg("failed to collect users consumptions")
				continue
			}

			if len(report.Consumption) == 0 {
				// nothing to report
				continue
			}

			if err := r.push(report); err != nil {
				log.Error().Err(err).Msg("failed to push capacity report")
			}
		}
	}
}

func (r *Reporter) collect(since time.Time) (rep farmer.Report, err error) {
	users, err := r.storage.Twins()
	if err != nil {
		return rep, err
	}

	rep.Timestamp = since.Unix()
	rep.Consumption = make(map[uint32]farmer.Consumption)

	for _, user := range users {
		cap, err := r.user(since, user)
		if err != nil {
			log.Error().Err(err).Msg("failed to collect all user capacity")
			// NOTE: we intentionally not doing a 'continue' or 'return'
			// here because even if an error is returned we can have partially
			// collected some of the user consumption, we still can report that
		}

		if cap.Capacity.Zero() {
			continue
		}

		rep.Consumption[user] = cap
	}

	return
}

func (r *Reporter) push(report farmer.Report) error {
	// create signature
	var buf bytes.Buffer
	if err := report.Challenge(&buf); err != nil {
		return errors.Wrap(err, "failed to create report challenge")
	}

	signature, err := r.identity.Sign(context.TODO(), buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "failed to sign report")
	}

	report.Signature = hex.EncodeToString(signature)
	return r.queue.Enqueue(&report)
}

func (r *Reporter) user(since time.Time, user uint32) (farmer.Consumption, error) {
	var m many
	types := gridtypes.Types()
	consumption := farmer.Consumption{
		Workloads: make(map[gridtypes.WorkloadType][]gridtypes.WorkloadID),
	}

	for _, typ := range types {
		consumption.Workloads[typ] = make([]gridtypes.WorkloadID, 0)
		ids, err := r.storage.ByTwin(user)
		if err != nil {
			m = m.append(errors.Wrapf(err, "failed to get reservation for user '%s' type '%s", user, typ))
			continue
		}

		for _, id := range ids {
			dl, err := r.storage.Get(user, id)
			if err != nil {
				m = m.append(errors.Wrapf(err, "failed to get reservation '%s'", id))
				continue
			}

			for i := range dl.Workloads {
				wl := &dl.Workloads[i]

				if wl.Result.IsNil() {
					// no results yet
					continue
				}

				if r.shouldCount(since, &wl.Result) {
					cap, err := wl.Capacity()
					if err != nil {
						m = m.append(errors.Wrapf(err, "failed to get reservation '%s' capacity", id))
						continue
					}

					wlID, _ := gridtypes.NewWorkloadID(user, id, wl.Name)
					consumption.Workloads[typ] = append(consumption.Workloads[typ], wlID)
					consumption.Capacity.Add(&cap)
				}
			}

		}
	}

	return consumption, m.AsError()
}

func (r *Reporter) shouldCount(since time.Time, result *gridtypes.Result) bool {
	if result.State == gridtypes.StateOk {
		return true
	}

	if result.State == gridtypes.StateDeleted {
		// if it was stopped before the 'since' .
		if result.Created.Time().Before(since) {
			return false
		}
		// otherwise it's true
		return true
	}

	return false
}
