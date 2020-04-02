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

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/pkg/phonebook/types"
	"github.com/zaibon/httpsig"

	"go.mongodb.org/mongo-driver/mongo"
)

// UserKeyGetter implements httpsig.KeyGetter for the users collections
type UserKeyGetter struct {
	db *mongo.Database
}

// NewUserKeyGetter create a httpsig.KeyGetter that uses the users collection
// to find the key
func NewUserKeyGetter(db *mongo.Database) UserKeyGetter {
	return UserKeyGetter{db: db}
}

// GetKey implements httpsig.KeyGetter
func (u UserKeyGetter) GetKey(id string) interface{} {
	ctx := context.TODO()

	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil
	}

	f := types.UserFilter{}
	f = f.WithID(schema.ID(uid))

	user, err := f.Get(ctx, u.db)
	if err != nil {
		return nil
	}

	pk, err := hex.DecodeString(user.Pubkey)
	if err != nil {
		return nil
	}
	return ed25519.PublicKey(pk)
}

// NodeKeyGetter implements httpsig.KeyGetter for the nodes collections
type NodeKeyGetter struct{}

// NewNodeKeyGetter create a httpsig.KeyGetter that uses the nodes collection
// to find the key
func NewNodeKeyGetter() NodeKeyGetter {
	return NodeKeyGetter{}
}

// GetKey implements httpsig.KeyGetter
func (m NodeKeyGetter) GetKey(id string) interface{} {
	// the node ID is its public key base58 encoded, so we just need
	// to decode it to get the []byte version of the key
	return ed25519.PublicKey(base58.Decode(id))
}

// requiredHeaders are the parameters to be used to generated the http signature
var requiredHeaders = []string{"(created)", "date", "threebot-id"}

// AuthMiddleware implements https://tools.ietf.org/html/draft-cavage-http-signatures-12
// authentication scheme as an HTTP middleware
type AuthMiddleware struct {
	verifier *httpsig.Verifier
}

// NewAuthMiddleware creates a new AuthMiddleware using the v httpsig.Verifier
func NewAuthMiddleware(v *httpsig.Verifier) *AuthMiddleware {
	v.SetRequiredHeaders(requiredHeaders)
	return &AuthMiddleware{
		verifier: v,
	}
}

// Middleware implements mux.Middlware interface
func (a *AuthMiddleware) Middleware(handler http.Handler) http.Handler {
	var challengeParams []string
	if headers := a.verifier.RequiredHeaders(); len(headers) > 0 {
		challengeParams = append(challengeParams,
			fmt.Sprintf("headers=%q", strings.Join(headers, " ")))
	}

	challenge := "Signature"
	if len(challengeParams) > 0 {
		challenge += fmt.Sprintf(" %s", strings.Join(challengeParams, ", "))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		keyID, err := a.verifier.Verify(req)
		if err != nil {
			w.Header()["WWW-Authenticate"] = []string{challenge}
			w.WriteHeader(http.StatusUnauthorized)

			log.Error().Err(err).Msgf("unauthorized access to %s", req.URL.Path)

			object := struct {
				Error string `json:"error"`
			}{
				Error: errors.Wrap(err, "unauthorized access").Error(),
			}
			if err := json.NewEncoder(w).Encode(object); err != nil {
				log.Error().Err(err).Msg("failed to encode return object")
			}
			return
		}
		handler.ServeHTTP(w, req.WithContext(httpsig.WithKeyID(req.Context(), keyID)))
	})
}
