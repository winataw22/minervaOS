package mw

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/zaibon/httpsig"
)

type twinPublicKeyID struct{}
type twinKeyID struct{}

// TwinPublicKey extracts twin public key from request
func TwinPublicKey(ctx context.Context) ed25519.PublicKey {
	value := ctx.Value(twinPublicKeyID{})
	return value.(ed25519.PublicKey)
}

// TwinID extracts twin id from request
func TwinID(ctx context.Context) uint32 {
	value := ctx.Value(twinKeyID{})
	return value.(uint32)
}

// UserMap implements httpsig.KeyGetter for the users collections
type UserMap map[uint32]ed25519.PublicKey

// NewUserMap create a httpsig.KeyGetter that uses the users collection
// to find the key
func NewUserMap() UserMap {
	return UserMap{}
}

// AddKeyFromHex adds a user key to map from a hex string
func (u UserMap) AddKeyFromHex(id uint32, key string) error {
	k, err := hex.DecodeString(key)
	if err != nil {
		return err
	}
	u[id] = ed25519.PublicKey(k)
	return nil
}

// GetKey implements httpsig.KeyGetter
func (u UserMap) GetKey(id uint32) (ed25519.PublicKey, error) {
	key, ok := u[id]
	if !ok {
		return nil, fmt.Errorf("unknown user id '%d' in key map", id)
	}
	return key, nil
}

// requiredHeaders are the parameters to be used to generated the http signature
var requiredHeaders = []string{"(created)", "date"}

type keyGetter struct {
	twins provision.Twins
}

func (k *keyGetter) GetKey(id string) (interface{}, error) {
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "expected uint user id")
	}

	return k.twins.GetKey(uint32(idUint))
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
func NewAuthMiddleware(users provision.Twins) mux.MiddlewareFunc {
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
			id, err := verifier.Verify(req)
			if err != nil {
				w.Header()["WWW-Authenticate"] = []string{challenge}
				w.WriteHeader(http.StatusUnauthorized)

				writeError(w, errors.Wrap(err, "unauthorized access"))
				return
			}

			twinID, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				// this should never happen because we already passed
				// the verifier but just in case
				w.WriteHeader(http.StatusInternalServerError)
				writeError(w, err)
				return
			}

			pk, err := users.GetKey(uint32(twinID))
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				writeError(w, err)
				return
			}

			ctx := req.Context()
			ctx = context.WithValue(ctx, twinKeyID{}, uint32(twinID))
			ctx = context.WithValue(ctx, twinPublicKeyID{}, pk)

			handler.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}
