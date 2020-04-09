package types

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/xdr"

	"github.com/threefoldtech/zos/pkg/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	// EscrowCollection db collection name
	EscrowCollection = "escrow"
)

var (
	// ErrEscrowExists is returned when trying to save escrow information for a
	// reservation that already has escrow information
	ErrEscrowExists = errors.New("escrow(s) for reservation already exists")
	// ErrEscrowNotFound is returned if escrow information is not found
	ErrEscrowNotFound = errors.New("escrow information not found")
)

type (
	// ReservationPaymentInformation stores the reservation payment information
	ReservationPaymentInformation struct {
		ReservationID schema.ID      `bson:"_id"`
		Address       string         `bson:"address"`
		Expiration    schema.Date    `bson:"expiration"`
		Infos         []EscrowDetail `bson:"infos"`
		// Paid indicates the reservation escrows have been fully funded, and
		// the reservation has been moved from the "PAY" state to the "DEPLOY"
		// state
		Paid bool `bson:"paid"`
		// Released indicates the reservation has been fully deployed, and
		// that an attempt was made to pay the farmers. If this flag is set, it is
		// still possible that there are funds on an escrow related to this transaction
		// either because someone funded it after the reservation was already
		// deployed , because there was an error paying the farmers
		// or because there was an error refunding any overpaid amount.
		Released bool `bson:"released"`
		// Canceled indicates that the reservation was canceled, either by the
		// user, or because a workload deployment failed, which resulted in the
		// entire reservation being canceled. As a result, an attempt was made
		// to refund the client. It is possible for this to have failed.
		Canceled bool `bson:"canceled"`
	}

	// EscrowDetail hold the details of an escrow address
	EscrowDetail struct {
		FarmerID    schema.ID `bson:"farmer_id" json:"farmer_id"`
		TotalAmount xdr.Int64 `bson:"total_amount" json:"total_amount"`
	}

	// CustomerEscrowInformation is the escrow information which will get exposed
	// to the customer once he creates a reservation
	CustomerEscrowInformation struct {
		Address string         `json:"address"`
		Details []EscrowDetail `json:"details"`
	}
)

// ReservationPaymentInfoCreate creates the reservation payment information
func ReservationPaymentInfoCreate(ctx context.Context, db *mongo.Database, reservationPaymentInfo ReservationPaymentInformation) error {
	col := db.Collection(EscrowCollection)
	_, err := col.InsertOne(ctx, reservationPaymentInfo)
	if err != nil {
		if merr, ok := err.(mongo.WriteException); ok {
			errCode := merr.WriteErrors[0].Code
			if errCode == 11000 {
				return ErrEscrowExists
			}
		}
		return err
	}
	return nil
}

// ReservationPaymentInfoUpdate update reservation payment info
func ReservationPaymentInfoUpdate(ctx context.Context, db *mongo.Database, update ReservationPaymentInformation) error {
	filter := bson.M{"_id": update.ReservationID}
	// actually update the user with final data
	if _, err := db.Collection(EscrowCollection).UpdateOne(ctx, filter, bson.M{"$set": update}); err != nil {
		return err
	}

	return nil
}

// ReservationPaymentInfoGet a single reservation escrow info using its id
func ReservationPaymentInfoGet(ctx context.Context, db *mongo.Database, id schema.ID) (ReservationPaymentInformation, error) {
	col := db.Collection(EscrowCollection)
	var rpi ReservationPaymentInformation
	res := col.FindOne(ctx, bson.M{"_id": id})
	if err := res.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return rpi, ErrEscrowNotFound
		}
		return rpi, err
	}
	err := res.Decode(&rpi)
	return rpi, err

}

// GetAllActiveReservationPaymentInfos get all active reservation payment information
func GetAllActiveReservationPaymentInfos(ctx context.Context, db *mongo.Database) ([]ReservationPaymentInformation, error) {
	filter := bson.M{"paid": false, "expiration": bson.M{"$gt": schema.Date{Time: time.Now()}}}
	cursor, err := db.Collection(EscrowCollection).Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cursor over active payment infos")
	}
	paymentInfos := make([]ReservationPaymentInformation, 0)
	err = cursor.All(ctx, &paymentInfos)
	if err != nil {
		err = errors.Wrap(err, "failed to decode active payment information")
	}
	return paymentInfos, err
}

// GetAllExpiredReservationPaymentInfos get all active reservation payment information
func GetAllExpiredReservationPaymentInfos(ctx context.Context, db *mongo.Database) ([]ReservationPaymentInformation, error) {
	filter := bson.M{"released": false, "canceled": false, "expiration": bson.M{"$lte": schema.Date{Time: time.Now()}}}
	cursor, err := db.Collection(EscrowCollection).Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cursor over expired payment infos")
	}
	paymentInfos := make([]ReservationPaymentInformation, 0)
	err = cursor.All(ctx, &paymentInfos)
	if err != nil {
		err = errors.Wrap(err, "failed to decode expired payment information")
	}
	return paymentInfos, err
}
