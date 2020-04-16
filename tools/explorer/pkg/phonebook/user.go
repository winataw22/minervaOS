package phonebook

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	"github.com/threefoldtech/zos/tools/explorer/pkg/phonebook/types"
)

// UserAPI struct
type UserAPI struct{}

// create user entry point, makes sure name is free for reservation
func (u *UserAPI) create(r *http.Request) (interface{}, mw.Response) {
	var user types.User

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, mw.BadRequest(err)
	}

	// https://github.com/threefoldtech/zos/issues/706
	if err := user.Validate(); err != nil {
		return nil, mw.BadRequest(err)
	}

	db := mw.Database(r)
	user, err := types.UserCreate(r.Context(), db, user.Name, user.Email, user.Pubkey)
	if err != nil && errors.Is(err, types.ErrUserExists) {
		return nil, mw.Conflict(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return user, mw.Created()
}

/*
register
As implemented in threebot. It works as a USER update function. To update
any fields, you need to make sure your payload has an extra "sender_signature_hex"
field that is the signature of the payload using the user private key.

This signature is done on a message that is built as defined by the User.Encode() method
*/
func (u *UserAPI) register(r *http.Request) (interface{}, mw.Response) {
	id, err := u.parseID(mux.Vars(r)["user_id"])
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "invalid user id"))
	}

	var payload struct {
		types.User
		Signature string `json:"sender_signature_hex"` // because why not `signature`!
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, mw.BadRequest(err)
	}

	if len(payload.Signature) == 0 {
		return nil, mw.BadRequest(fmt.Errorf("signature is required"))
	}

	signature, err := hex.DecodeString(payload.Signature)
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "invalid signature hex"))
	}
	db := mw.Database(r)

	if err := types.UserUpdate(r.Context(), db, schema.ID(id), signature, payload.User); err != nil {
		if errors.Is(err, types.ErrBadUserUpdate) {
			return nil, mw.BadRequest(err)
		}
		return nil, mw.Error(err)
	}

	return nil, nil
}

func (u *UserAPI) list(r *http.Request) (interface{}, mw.Response) {
	var filter types.UserFilter
	filter = filter.WithName(r.FormValue("name"))
	filter = filter.WithEmail(r.FormValue("email"))

	db := mw.Database(r)
	pager := models.PageFromRequest(r)
	cur, err := filter.Find(r.Context(), db, pager)
	if err != nil {
		return nil, mw.Error(err)
	}

	users := []types.User{}
	if err := cur.All(r.Context(), &users); err != nil {
		return nil, mw.Error(err)
	}

	total, err := filter.Count(r.Context(), db)
	if err != nil {
		return nil, mw.Error(err, http.StatusInternalServerError)
	}

	nrPages := math.Ceil(float64(total) / float64(*pager.Limit))
	pages := fmt.Sprintf("%d", int64(nrPages))

	return users, mw.Ok().WithHeader("Pages", pages)
}

func (u *UserAPI) parseID(id string) (schema.ID, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid id format")
	}

	return schema.ID(v), nil
}

func (u *UserAPI) get(r *http.Request) (interface{}, mw.Response) {

	userID, err := u.parseID(mux.Vars(r)["user_id"])
	if err != nil {
		return nil, mw.BadRequest(err)
	}
	var filter types.UserFilter
	filter = filter.WithID(userID)

	db := mw.Database(r)
	user, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return user, nil
}

func (u *UserAPI) validate(r *http.Request) (interface{}, mw.Response) {
	var payload struct {
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}

	userID, err := u.parseID(mux.Vars(r)["user_id"])
	if err != nil {
		return nil, mw.BadRequest(err)
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, mw.BadRequest(err)
	}

	data, err := hex.DecodeString(payload.Payload)
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "payload must be hex encoded string of original data"))
	}

	signature, err := hex.DecodeString(payload.Signature)
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "signature must be hex encoded string of original data"))
	}

	var filter types.UserFilter
	filter = filter.WithID(userID)

	db := mw.Database(r)
	user, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	key, err := crypto.KeyFromHex(user.Pubkey)
	if err != nil {
		return nil, mw.Error(err)
	}

	if len(key) != ed25519.PublicKeySize {
		return nil, mw.Error(fmt.Errorf("public key has the wrong size"))
	}

	return struct {
		IsValid bool `json:"is_valid"`
	}{
		IsValid: ed25519.Verify(key, data, signature),
	}, nil
}
