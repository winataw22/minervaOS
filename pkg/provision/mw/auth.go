package mw

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/zaibon/httpsig"
)

type usePublicKeyID struct{}
type userKeyID struct{}

// UserPublicKey extracts user public key from request
func UserPublicKey(ctx context.Context) ed25519.PublicKey {
	value := ctx.Value(usePublicKeyID{})
	return value.(ed25519.PublicKey)
}

// UserID extracts user id from request
func UserID(ctx context.Context) gridtypes.ID {
	value := ctx.Value(userKeyID{})
	return value.(gridtypes.ID)
}

// UserMap implements httpsig.KeyGetter for the users collections
type UserMap map[gridtypes.ID]ed25519.PublicKey

// NewUserMap create a httpsig.KeyGetter that uses the users collection
// to find the key
func NewUserMap() UserMap {
	return UserMap{}
}

// AddKeyFromHex adds a user key to map from a hex string
func (u UserMap) AddKeyFromHex(id gridtypes.ID, key string) error {
	k, err := hex.DecodeString(key)
	if err != nil {
		return err
	}
	u[id] = ed25519.PublicKey(k)
	return nil
}

// GetKey implements httpsig.KeyGetter
func (u UserMap) GetKey(id gridtypes.ID) (ed25519.PublicKey, error) {
	key, ok := u[id]
	if !ok {
		return nil, fmt.Errorf("unknown user id '%s' in key map", id)
	}
	return key, nil
}

// requiredHeaders are the parameters to be used to generated the http signature
var requiredHeaders = []string{"(created)", "date"}

type keyGetter struct {
	users provision.Users
}

func (k *keyGetter) GetKey(id string) (interface{}, error) {
	return k.users.GetKey(gridtypes.ID(id))
}

func writeError(w http.ResponseWriter, err error) {
	object := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	if err := json.NewEncoder(w).Encode(object); err != nil {
		log.Error().Err(err).Msg("failed to encode return object")
	}
}

// NewAuthMiddleware creates a new AuthMiddleware using the v httpsig.Verifier
func NewAuthMiddleware(users provision.Users) mux.MiddlewareFunc {
	verifier := httpsig.NewVerifier(&keyGetter{users})
	verifier.SetRequiredHeaders(requiredHeaders)
	var challengeParams []string
	if headers := verifier.RequiredHeaders(); len(headers) > 0 {
		challengeParams = append(challengeParams,
			fmt.Sprintf("headers=%q", strings.Join(headers, " ")))
	}

	challenge := "Signature"
	if len(challengeParams) > 0 {
		challenge += fmt.Sprintf(" %s", strings.Join(challengeParams, ", "))
	}

	return func(handler http.Handler) http.Handler {
		//http.Error(w http.ResponseWriter, error string, code int)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			userID, err := verifier.Verify(req)
			if err != nil {
				w.Header()["WWW-Authenticate"] = []string{challenge}
				w.WriteHeader(http.StatusUnauthorized)

				writeError(w, errors.Wrap(err, "unauthorized access"))
				return
			}

			pk, err := users.GetKey(gridtypes.ID(userID))
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				writeError(w, err)
				return
			}

			ctx := req.Context()
			ctx = context.WithValue(ctx, userKeyID{}, gridtypes.ID(userID))
			ctx = context.WithValue(ctx, usePublicKeyID{}, pk)

			handler.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}
