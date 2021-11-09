package mw

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/provision"
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

// UserMap implements provision.Twins for the users collections
type UserMap map[uint32]ed25519.PublicKey

// NewUserMap create a new UserMap that uses the users collection
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

// GetKey implements interface
func (u UserMap) GetKey(id uint32) ([]byte, error) {
	key, ok := u[id]
	if !ok {
		return nil, fmt.Errorf("unknown user id '%d' in key map", id)
	}
	return key, nil
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)

	object := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	if err := json.NewEncoder(w).Encode(object); err != nil {
		log.Error().Err(err).Msg("failed to encode return object")
	}
}

// NewAuthMiddleware creates a new AuthMiddleware using jwt signed by the caller
func NewAuthMiddleware(users provision.Twins) mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := jwt.ParseHeader(r.Header, "authorization",
				jwt.WithValidate(true),
				jwt.WithAudience("zos"),
				jwt.WithAcceptableSkew(10*time.Second),
			)
			if err != nil {
				writeError(w, errors.Wrap(err, "failed to parse jwt token"))
				return
			}

			if time.Until(token.Expiration()) > 2*time.Minute {
				writeError(w, fmt.Errorf("the expiration date should not be more than 2 minutes"))
				return
			}
			twinID, err := strconv.ParseUint(token.Issuer(), 10, 32)
			if err != nil {
				writeError(w, errors.Wrap(err, "failed to parse issued id, expecting a 32 bit uint"))
				return
			}
			pk, err := users.GetKey(uint32(twinID))
			if err != nil {
				writeError(w, errors.Wrap(err, "failed to get twin public key"))
				return
			}
			// reparse the token but with signature validation
			_, err = jwt.ParseHeader(r.Header, "authorization", jwt.WithValidate(true),
				jwt.WithAudience("zos"),
				jwt.WithAcceptableSkew(10*time.Second),
				jwt.WithVerify(jwa.EdDSA, pk),
			)

			if err != nil {
				writeError(w, errors.Wrap(err, "failed to get twin public key"))
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, twinKeyID{}, uint32(twinID))
			ctx = context.WithValue(ctx, twinPublicKeyID{}, pk)

			handler.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
