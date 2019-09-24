package main

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

const (
	// IYOPublicKey is itsyouonline public key
	IYOPublicKey = `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAES5X8XrfKdx9gYayFITc89wad4usrk0n2
7MjiGYvqalizeSWTHEpnd7oea9IQ8T5oJjMVH5cc0H5tFSKilFFeh//wngxIyny6
6+Vq5t5B0V0Ehy01+2ceEon2Y0XDkIKv
-----END PUBLIC KEY-----`
)

// JWTUser validates token and extract user name
func JWTUser(token string) (string, error) {

	pub, err := jwt.ParseECPublicKeyFromPEM([]byte(IYOPublicKey))
	if err != nil {
		return "", err
	}

	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		m, ok := token.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		if token.Header["alg"] != m.Alg() {
			return nil, fmt.Errorf("Unexpected signing algorithm: %v", token.Header["alg"])
		}
		return pub, nil
	})

	if err != nil {
		return "", errors.Wrap(err, "failed to validate token")
	}

	if claims, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
		return claims["username"].(string), nil
	}

	return "", fmt.Errorf("could not extract user")
}
