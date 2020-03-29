package phonebook

import (
	"encoding/json"
	"net"

	schema "github.com/threefoldtech/zos/pkg/schema"
)

type User struct {
	ID          schema.ID `bson:"_id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Email       string    `bson:"email" json:"email"`
	Pubkey      string    `bson:"pubkey" json:"pubkey"`
	Ipaddr      net.IP    `bson:"ipaddr" json:"ipaddr"`
	Description string    `bson:"description" json:"description"`
	Signature   string    `bson:"signature" json:"signature"`
}

func NewUser() (User, error) {
	const value = "{}"
	var object User
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
