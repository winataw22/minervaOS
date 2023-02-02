package noded

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/kernel"
)

const (
	keyType = "ed25519"
)

func withDefaultPort(substrateUrl string) (string, error) {
	u, err := url.ParseRequestURI(substrateUrl)
	if err != nil {
		return "", err
	}

	if u.Port() != "" {
		// already have the port
		return substrateUrl, nil
	}

	if u.Scheme == "ws" {
		u.Host += ":80"
	} else {
		u.Host += ":443"
	}

	return u.String(), nil
}

func runMsgBus(ctx context.Context, sk ed25519.PrivateKey, substrateURLs []string, relayAddr string, redisAddr string) error {
	// select the first one as only one URL is set for now
	if len(substrateURLs) == 0 {
		return errors.New("at least one substrate URL must be provided")
	}

	seed := sk.Seed()
	seedHex := fmt.Sprintf("0x%s", hex.EncodeToString(seed))

	log.Info().Msg("starting rmb...")

	substrateURL, err := withDefaultPort(substrateURLs[0])
	if err != nil {
		return err
	}

	args := []string{
		"-s", substrateURL,
		"--relay", relayAddr,
		"-k", keyType,
		"--seed", seedHex,
		"--redis", redisAddr,
	}

	if kernel.GetParams().IsDebug() {
		args = append(args, "-d")
	}

	command := exec.CommandContext(
		ctx,
		"rmb",
		args...,
	)

	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug().Stringer("command", command).Msg("running rmb")
	return command.Run()
}
