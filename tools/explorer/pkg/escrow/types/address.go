package types

import (
	"context"

	"github.com/pkg/errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	// AddressCollection db collection name
	AddressCollection = "addresses"
)

var (
	// ErrAddressExists error for non existing doc
	ErrAddressExists = errors.New("address(s) for farmer customer already exists")
	// ErrAddressNotFound error for not found
	ErrAddressNotFound = errors.New("address information not found")
)

type (
	// FarmerCustomerAddress holds the farmer - customer key
	FarmerCustomerAddress struct {
		FarmerID    int64  `bson:"farmer_id" json:"farmer_id"`
		CustomerTID int64  `bson:"customer_tid" json:"customer_tid"`
		Address     string `bson:"address" json:"address"`
		Secret      string `bson:"secret" json:"secret"`
	}
)

// FarmerCustomerAddressCreate creates the reservation payment information
func FarmerCustomerAddressCreate(ctx context.Context, db *mongo.Database, fckAddress FarmerCustomerAddress) error {
	col := db.Collection(AddressCollection)
	_, err := col.InsertOne(ctx, fckAddress)
	if err != nil {
		if merr, ok := err.(mongo.WriteException); ok {
			errCode := merr.WriteErrors[0].Code
			if errCode == 11000 {
				return ErrAddressExists
			}
		}
		return err
	}
	return nil
}

// Get gets one address by farmerID and customerTID
func Get(ctx context.Context, db *mongo.Database, farmerID int64, customerTID int64) (FarmerCustomerAddress, error) {
	var farmerCustomerAddress FarmerCustomerAddress
	doc := db.Collection(AddressCollection).FindOne(ctx, bson.M{"farmer_id": farmerID, "customer_tid": customerTID})
	if errors.Is(doc.Err(), mongo.ErrNoDocuments) {
		return FarmerCustomerAddress{}, ErrAddressNotFound
	}
	err := doc.Decode(&farmerCustomerAddress)
	return farmerCustomerAddress, err
}

// GetByAddress gets one address using the address
func GetByAddress(ctx context.Context, db *mongo.Database, address string) (FarmerCustomerAddress, error) {
	var farmerCustomerAddress FarmerCustomerAddress
	doc := db.Collection(AddressCollection).FindOne(ctx, bson.M{"address": address})
	if errors.Is(doc.Err(), mongo.ErrNoDocuments) {
		return FarmerCustomerAddress{}, ErrAddressNotFound
	}
	err := doc.Decode(&farmerCustomerAddress)
	return farmerCustomerAddress, err
}
