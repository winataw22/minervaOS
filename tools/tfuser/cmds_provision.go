package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/provision"

	"github.com/urfave/cli"
)

var (
	day             = time.Hour * 24
	defaultDuration = day * 30
)

func encryptSecret(plain, nodeID string) (string, error) {
	if len(plain) == 0 {
		return "", nil
	}

	pubkey, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return "", err
	}

	encrypted, err := crypto.Encrypt([]byte(plain), pubkey)
	return hex.EncodeToString(encrypted), err
}

func provisionCustomZDB(r *provision.Reservation) error {
	var config provision.ZDB
	if err := json.Unmarshal(r.Data, &config); err != nil {
		return errors.Wrap(err, "failed to load zdb reservation schema")
	}

	encrypted, err := encryptSecret(config.Password, r.NodeID)
	if err != nil {
		return err
	}

	config.Password = encrypted
	r.Data, err = json.Marshal(config)

	return err
}

func provisionCustomContainer(r *provision.Reservation) error {
	var config provision.Container
	var err error
	if err := json.Unmarshal(r.Data, &config); err != nil {
		return errors.Wrap(err, "failed to load zdb reservation schema")
	}

	if config.SecretEnv == nil {
		config.SecretEnv = make(map[string]string)
	}

	for k, v := range config.Env {
		v, err := encryptSecret(v, r.NodeID)
		if err != nil {
			return errors.Wrapf(err, "failed to encrypt env with key '%s'", k)
		}
		config.SecretEnv[k] = v
	}
	config.Env = make(map[string]string)
	r.Data, err = json.Marshal(config)

	return err
}

var (
	provCustomModifiers = map[provision.ReservationType]func(r *provision.Reservation) error{
		provision.ZDBReservation:       provisionCustomZDB,
		provision.ContainerReservation: provisionCustomContainer,
	}
)

func cmdsProvision(c *cli.Context) error {
	var (
		schema   []byte
		path     = c.String("schema")
		nodeIDs  = c.StringSlice("node")
		seedPath = c.String("seed")
		d        = c.String("duration")
		duration time.Duration
		err      error
	)

	if d == "" {
		duration = defaultDuration
	} else {
		duration, err = time.ParseDuration(d)
		if err != nil {
			nrDays, err := strconv.Atoi(d)
			if err != nil {
				return errors.Wrap(err, "unsupported duration format")
			}
			duration = time.Duration(nrDays) * day
		}
	}

	keypair, err := identity.LoadKeyPair(seedPath)
	if err != nil {
		return errors.Wrapf(err, "could not find seed file at %s", seedPath)
	}

	if path == "-" {
		schema, err = ioutil.ReadAll(os.Stdin)
	} else {
		schema, err = ioutil.ReadFile(path)
	}
	if err != nil {
		return errors.Wrap(err, "could not find provision schema")
	}

	var reservation provision.Reservation
	if err := json.Unmarshal(schema, &reservation); err != nil {
		return errors.Wrap(err, "failed to read the provision schema")
	}

	reservation.Duration = duration
	reservation.Created = time.Now()
	// set the user ID into the reservation schema
	reservation.User = keypair.Identity()

	for _, nodeID := range nodeIDs {
		r := reservation //make a copy
		r.NodeID = nodeID

		custom, ok := provCustomModifiers[r.Type]
		if ok {
			if err := custom(&r); err != nil {
				return err
			}
		}

		if err := r.Sign(keypair.PrivateKey); err != nil {
			return errors.Wrap(err, "failed to sign the reservation")
		}

		id, err := client.Reserve(&r)
		if err != nil {
			return errors.Wrap(err, "failed to send reservation")
		}

		fmt.Printf("Reservation for %v send to node %s\n", duration, r.NodeID)
		fmt.Printf("Resource: %v\n", id)
	}

	return nil
}

func embed(schema interface{}, t provision.ReservationType) (*provision.Reservation, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	r := &provision.Reservation{
		Type: t,
		Data: raw,
	}

	return r, nil
}

func cmdsDeleteReservation(c *cli.Context) error {
	id := c.String("id")

	if err := client.Delete(id); err != nil {
		return errors.Wrapf(err, "failed to mark reservation %s to be deleted", id)
	}
	fmt.Printf("Reservation %v marked as to be deleted\n", id)
	return nil
}
