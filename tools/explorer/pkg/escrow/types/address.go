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
	// CustomerAddress holds the customer escrow address and key
	CustomerAddress struct {
		CustomerTID int64  `bson:"customer_tid" json:"customer_tid"`
		Address     string `bson:"address" json:"address"`
		Secret      string `bson:"secret" json:"secret"`
	}
)

// CustomerAddressCreate creates the reservation payment information
func CustomerAddressCreate(ctx context.Context, db *mongo.Database, cAddress CustomerAddress) error {
	col := db.Collection(AddressCollection)
	_, err := col.InsertOne(ctx, cAddress)
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

// CustomerAddressGet one address bycustomerTID
func CustomerAddressGet(ctx context.Context, db *mongo.Database, customerTID int64) (CustomerAddress, error) {
	var customerAddress CustomerAddress
	doc := db.Collection(AddressCollection).FindOne(ctx, bson.M{"customer_tid": customerTID})
	if errors.Is(doc.Err(), mongo.ErrNoDocuments) {
		return CustomerAddress{}, ErrAddressNotFound
	}
	err := doc.Decode(&customerAddress)
	return customerAddress, err
}

// CustomerAddressByAddress gets one address using the address
func CustomerAddressByAddress(ctx context.Context, db *mongo.Database, address string) (CustomerAddress, error) {
	var customerAddress CustomerAddress
	doc := db.Collection(AddressCollection).FindOne(ctx, bson.M{"address": address})
	if errors.Is(doc.Err(), mongo.ErrNoDocuments) {
		return CustomerAddress{}, ErrAddressNotFound
	}
	err := doc.Decode(&customerAddress)
	return customerAddress, err
}
