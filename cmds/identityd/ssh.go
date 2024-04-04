package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/kernel"
)

var (
	mainNetFarms = []pkg.FarmID{
		1, 79, 77, 76,
	}
)

func manageSSHKeys() error {
	extraUser, addUser := kernel.GetParams().GetOne("ssh-user")

	authorizedKeysPath := filepath.Join("/", "root", ".ssh", "authorized_keys")
	err := os.Remove(authorizedKeysPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to deleted authorized_keys file: %+w", err)
	}

	env := environment.MustGet()
	config, err := environment.GetConfig()
	if err != nil {
		return err
	}

	authorizedUsers := config.Users.Authorized

	if env.RunningMode == environment.RunningMain {
		// we don't support adding the user passed as ssh-user on mainnet
		addUser = false
	}

	// if we are in mainnet but one of the managed farms we will use the user list from testnet
	// instead
	if env.RunningMode == environment.RunningMain && slices.Contains(mainNetFarms, env.FarmID) {
		// that's only if main config has no configured users
		if len(authorizedUsers) == 0 {
			config, err = environment.GetConfigForMode(environment.RunningTest)
			if err != nil {
				return err
			}

			authorizedUsers = config.Users.Authorized
		}
	}

	// check if we will add the extra user
	if addUser {
		authorizedUsers = append(authorizedUsers, extraUser)
	}

	file, err := os.OpenFile(authorizedKeysPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open authorized_keys files: %w", err)
	}

	defer file.Close()

	for _, user := range authorizedUsers {
		fetchKey := func() error {
			res, err := http.Get(fmt.Sprintf("https://github.com/%s.keys", user))

			if err != nil {
				return fmt.Errorf("failed to fetch user keys: %+w", err)
			}

			if res.StatusCode == http.StatusNotFound {
				return backoff.Permanent(fmt.Errorf("failed to get user keys for user (%s): keys not found", user))
			}

			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to get user keys for user (%s) with status code %d", user, res.StatusCode)
			}

			_, err = io.Copy(file, res.Body)
			return err
		}

		log.Info().Str("user", user).Msg("fetching user ssh keys")
		err = backoff.Retry(fetchKey, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 3))
		if err != nil {
			// skip user if failed to load the keys multiple times
			// this means the username is not correct and need to be skipped
			log.Error().Str("user", user).Err(err).Msg("failed to retrieve user keys")
		}
	}

	return nil
}
