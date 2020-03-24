package types

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// ReservationCollection db collection name
	ReservationCollection = "reservation"
	queueCollection       = "workqueue"
)

const (
	// Create action
	Create = generated.NextActionCreate
	// Sign action
	Sign = generated.NextActionSign
	// Pay action
	Pay = generated.NextActionPay
	// Deploy action
	Deploy = generated.NextActionDeploy
	// Delete action
	Delete = generated.NextActionDelete
	// Invalid action
	Invalid = generated.NextActionInvalid
	// Deleted action
	Deleted = generated.NextActionDeleted
)

// ReservationFilter type
type ReservationFilter bson.D

// WithID filter reservation with ID
func (f ReservationFilter) WithID(id schema.ID) ReservationFilter {
	return append(f, bson.E{Key: "_id", Value: id})
}

// WithIDGE return find reservations with
func (f ReservationFilter) WithIDGE(id schema.ID) ReservationFilter {
	return append(f, bson.E{
		Key: "_id", Value: bson.M{"$gte": id},
	})
}

// WithNextAction filter reservations with next action
func (f ReservationFilter) WithNextAction(action generated.NextActionEnum) ReservationFilter {
	return append(f, bson.E{
		Key: "next_action", Value: action,
	})
}

// WithNodeID searsch reservations with NodeID
func (f ReservationFilter) WithNodeID(id string) ReservationFilter {
	//data_reservation.{containers, volumes, zdbs, networks, kubernetes}.node_id
	// we need to search ALL types for any reservation that has the node ID
	or := []bson.M{}
	for _, typ := range []string{"containers", "volumes", "zdbs", "kubernetes"} {
		key := fmt.Sprintf("data_reservation.%s.node_id", typ)
		or = append(or, bson.M{key: id})
	}

	// network workload is special because node id is set on the network_resources.
	or = append(or, bson.M{"data_reservation.networks.network_resources.node_id": id})

	// we find any reservation that has this node ID set.
	return append(f, bson.E{Key: "$or", Value: or})
}

// Or returns filter that reads as (f or o)
func (f ReservationFilter) Or(o ReservationFilter) ReservationFilter {
	return ReservationFilter{
		bson.E{
			Key:   "$or",
			Value: bson.A{f, o},
		},
	}
}

// Get gets single reservation that matches the filter
func (f ReservationFilter) Get(ctx context.Context, db *mongo.Database) (reservation Reservation, err error) {
	if f == nil {
		f = ReservationFilter{}
	}

	result := db.Collection(ReservationCollection).FindOne(ctx, f)
	if err = result.Err(); err != nil {
		return
	}

	err = result.Decode(&reservation)
	return
}

// Find all users that matches filter
func (f ReservationFilter) Find(ctx context.Context, db *mongo.Database, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	if f == nil {
		f = ReservationFilter{}
	}
	return db.Collection(ReservationCollection).Find(ctx, f, opts...)
}

// Count number of documents matching
func (f ReservationFilter) Count(ctx context.Context, db *mongo.Database) (int64, error) {
	col := db.Collection(ReservationCollection)
	if f == nil {
		f = ReservationFilter{}
	}

	return col.CountDocuments(ctx, f)
}

// Reservation is a wrapper around generated type
type Reservation generated.Reservation

// Validate that the reservation is valid
func (r *Reservation) validate() error {
	if r.CustomerTid == 0 {
		return fmt.Errorf("customer_tid is required")
	}

	if len(r.CustomerSignature) == 0 {
		return fmt.Errorf("customer_signature is required")
	}

	var data generated.ReservationData

	if err := json.Unmarshal([]byte(r.Json), &data); err != nil {
		return errors.Wrap(err, "invalid json data on reservation")
	}

	if !reflect.DeepEqual(r.DataReservation, data) {
		return fmt.Errorf("json data does not match the reservation data")
	}

	ids := make(map[int64]struct{})

	// yes, it's ugly. live with it.
	for _, w := range r.DataReservation.Containers {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Networks {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Zdbs {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Volumes {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Kubernetes {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	return nil
}

// Verify signature against Reserveration.JSON
// pk is the public key used as verification key in hex encoded format
// the signature is the signature to verify (in raw binary format)
func (r *Reservation) Verify(pk string, sig []byte) error {
	key, err := crypto.KeyFromHex(pk)
	if err != nil {
		return errors.Wrap(err, "invalid verification key")
	}

	return crypto.Verify(key, []byte(r.Json), sig)
}

// SignatureVerify is similar to Verify but the verification is done
// against `str(Reservation.ID) + Reservation.JSON`
func (r *Reservation) SignatureVerify(pk string, sig []byte) error {
	key, err := crypto.KeyFromHex(pk)
	if err != nil {
		return errors.Wrap(err, "invalid verification key")
	}

	var buf bytes.Buffer
	if _, err := buf.WriteString(fmt.Sprint(int64(r.ID))); err != nil {
		return errors.Wrap(err, "failed to write id to buffer")
	}

	if _, err := buf.WriteString(r.Json); err != nil {
		return errors.Wrap(err, "failed to write json to buffer")
	}

	return crypto.Verify(key, buf.Bytes(), sig)
}

// Expired checks if this reservation has expired
func (r *Reservation) Expired() bool {
	return time.Until(r.DataReservation.ExpirationReservation.Time) <= 0
}

// IsAny checks if the reservation status is any of the given status
func (r *Reservation) IsAny(status ...generated.NextActionEnum) bool {
	for _, s := range status {
		if r.NextAction == s {
			return true
		}
	}

	return false
}

//ResultOf return result of a workload ID
func (r *Reservation) ResultOf(id string) *Result {
	for _, result := range r.Results {
		if result.WorkloadId == id {
			r := Result(result)
			return &r
		}
	}

	return nil
}

// AllDeleted checks of all workloads has been marked
func (r *Reservation) AllDeleted() bool {
	// check if all workloads have been deleted.
	for _, wl := range r.Workloads("") {
		result := r.ResultOf(wl.WorkloadId)
		if result == nil ||
			result.State != generated.ResultStateDeleted {
			return false
		}
	}

	return true
}

// Workloads returns all reservation workloads (filter by nodeID)
// if nodeID is empty, return all workloads
func (r *Reservation) Workloads(nodeID string) []Workload {
	data := &r.DataReservation
	var workloads []Workload
	for _, wl := range data.Containers {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			ReservationWorkload: generated.ReservationWorkload{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.WorkloadTypeContainer,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Volumes {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			ReservationWorkload: generated.ReservationWorkload{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.WorkloadTypeVolume,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Zdbs {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			ReservationWorkload: generated.ReservationWorkload{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.WorkloadTypeZDB,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Kubernetes {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			ReservationWorkload: generated.ReservationWorkload{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.WorkloadTypeKubernetes,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Networks {
		for _, nr := range wl.NetworkResources {

			if len(nodeID) > 0 && nr.NodeId != nodeID {
				continue
			}
			// QUESTION: the problem here is that we have multiple workloads that
			// has the same global workload-id, hence it's gonna be a problem
			// when the node report their results. because it means only last
			// result is what is gonna be visible. We need to (may be) change
			// the workload id to have the network resource index
			workload := Workload{
				ReservationWorkload: generated.ReservationWorkload{
					WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
					User:       fmt.Sprint(r.CustomerTid),
					Type:       generated.WorkloadTypeNetwork,
					Content:    wl,
					Created:    r.Epoch,
					Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
					ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
				},
				NodeID: nr.NodeId,
			}

			workloads = append(workloads, workload)
		}
	}

	return workloads
}

// ReservationCreate save new reservation to database.
// NOTE: use reservations only that are returned from calling Pipeline.Next()
// no validation is done here, this is just a CRUD operation
func ReservationCreate(ctx context.Context, db *mongo.Database, r Reservation) (schema.ID, error) {
	id := models.MustID(ctx, db, ReservationCollection)
	r.ID = id

	_, err := db.Collection(ReservationCollection).InsertOne(ctx, r)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// ReservationSetNextAction update the reservation next action in db
func ReservationSetNextAction(ctx context.Context, db *mongo.Database, id schema.ID, action generated.NextActionEnum) error {
	var filter ReservationFilter
	filter = filter.WithID(id)

	col := db.Collection(ReservationCollection)
	_, err := col.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"next_action": action,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

// SignatureMode type
type SignatureMode string

const (
	// SignatureProvision mode
	SignatureProvision SignatureMode = "signatures_provision"
	// SignatureDelete mode
	SignatureDelete SignatureMode = "signatures_delete"
)

//ReservationPushSignature push signature to reservation
func ReservationPushSignature(ctx context.Context, db *mongo.Database, id schema.ID, mode SignatureMode, signature generated.SigningSignature) error {

	var filter ReservationFilter
	filter = filter.WithID(id)
	col := db.Collection(ReservationCollection)
	// NOTE: this should be a transaction not a bulk write
	// but i had so many issues with transaction, and i couldn't
	// get it to work. so I used bulk write in place instead
	// until we figure this issue out.
	// Note, the reason we don't just use addToSet is the signature
	// object always have the current 'time' which means it's a different
	// value than the one in the document even if it has same user id.
	_, err := col.BulkWrite(ctx, []mongo.WriteModel{
		mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(
			bson.M{
				"$pull": bson.M{
					string(mode): bson.M{"tid": signature.Tid},
				},
			}),
		mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(
			bson.M{
				"$addToSet": bson.M{
					string(mode): signature,
				},
			}),
	}, options.BulkWrite().SetOrdered(true))

	return err
}

// Workload is a wrapper around generated TfgridWorkloadsReservationWorkload1 type
type Workload struct {
	generated.ReservationWorkload `bson:",inline"`
	NodeID                        string `json:"node_id" bson:"node_id"`
}

// QueueFilter for workloads in temporary queue
type QueueFilter bson.D

// WithNodeID search queue with node-id
func (f QueueFilter) WithNodeID(nodeID string) QueueFilter {
	return append(f, bson.E{Key: "node_id", Value: nodeID})
}

// Find runs the filter, and return a cursor
func (f QueueFilter) Find(ctx context.Context, db *mongo.Database, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	col := db.Collection(queueCollection)
	return col.Find(ctx, f, opts...)
}

// WorkloadPush pushes a workload to the queue
func WorkloadPush(ctx context.Context, db *mongo.Database, w ...Workload) error {
	col := db.Collection(queueCollection)
	docs := make([]interface{}, 0, len(w))
	for _, wl := range w {
		docs = append(docs, wl)
	}
	_, err := col.InsertMany(ctx, docs)

	return err
}

// WorkloadPop removes workload from queue
func WorkloadPop(ctx context.Context, db *mongo.Database, id string) error {
	col := db.Collection(queueCollection)
	_, err := col.DeleteOne(ctx, bson.M{"workload_id": id})

	return err
}

// Result is a wrapper around TfgridWorkloadsReservationResult1 type
type Result generated.Result

func (r *Result) encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := buf.WriteByte(byte(r.State)); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(r.Message); err != nil {
		return nil, err
	}
	if _, err := buf.Write(r.DataJson); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Verify that the signature matches the result data
func (r *Result) Verify(pk string) error {
	sig, err := hex.DecodeString(r.Signature)
	if err != nil {
		return errors.Wrap(err, "invalid signature expecting hex encoded")
	}

	key, err := crypto.KeyFromID(pkg.StrIdentifier(pk))
	if err != nil {
		return errors.Wrap(err, "invalid verification key")
	}

	bytes, err := r.encode()
	if err != nil {
		return err
	}

	return crypto.Verify(key, bytes, sig)
}

// ResultPush pushes result to a reservation result array.
// NOTE: this is just a crud operation, no validation is done here
func ResultPush(ctx context.Context, db *mongo.Database, id schema.ID, result Result) error {
	col := db.Collection(ReservationCollection)
	var filter ReservationFilter
	filter = filter.WithID(id)

	// we don't care if we couldn't delete old result.
	// in case it never existed, or the array is nil.
	col.UpdateOne(ctx, filter, bson.M{
		"$pull": bson.M{
			"results": bson.M{
				"workload_id": result.WorkloadId,
				"node_id":     result.NodeId,
			},
		},
	})

	_, err := col.UpdateOne(ctx, filter, bson.D{
		{
			Key: "$push",
			Value: bson.M{
				"results": result,
			},
		},
	})

	return err
}
