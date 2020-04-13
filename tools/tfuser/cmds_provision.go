package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/client"

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
		seedPath = mainSeed
		d        = c.String("duration")
		assets   = c.StringSlice("asset")
		userID   = int64(mainui.ThreebotID)
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

	signer, err := client.NewSigner(mainui.Key().PrivateKey.Seed())
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
	//reservation.User = keypair.Identity()

	custom, ok := provCustomModifiers[reservation.Type]
	fmt.Println("customization: ", ok)
	if ok {
		fmt.Println("running customization function", reservation.NodeID)
		if err := custom(&reservation); err != nil {
			return err
		}
	}

	jsx, err := reservation.ToSchemaType()
	if err != nil {
		return errors.Wrap(err, "failed to convert reservation to schema type")
	}
	jsx.CustomerTid = userID
	// we always allow user to delete his own reservations
	jsx.DataReservation.SigningRequestDelete.QuorumMin = 1
	jsx.DataReservation.SigningRequestDelete.Signers = []int64{userID}

	// set allowed the currencies as provided by the user
	jsx.DataReservation.Currencies = assets

	bytes, err := json.Marshal(jsx.DataReservation)
	if err != nil {
		return err
	}

	jsx.Json = string(bytes)
	_, signature, err := signer.SignHex(jsx.Json)
	if err != nil {
		return errors.Wrap(err, "failed to sign the reservation")
	}

	jsx.CustomerSignature = signature

	if c.Bool("dry-run") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(jsx)
	}

	response, err := bcdb.Workloads.Create(jsx)
	if err != nil {
		return errors.Wrap(err, "failed to send reservation")
	}

	totalAmount := xdr.Int64(0)
	for _, detail := range response.EscrowInformation.Details {
		totalAmount += detail.TotalAmount
	}

	fmt.Printf("Reservation for %v send to node bcdb\n", duration)
	fmt.Printf("Resource: /reservations/%v\n", response.ID)
	fmt.Println()

	fmt.Printf("Reservation id: %d \n", response.ID)
	fmt.Printf("Asset to pay: %s\n", response.EscrowInformation.Asset)
	fmt.Printf("Reservation escrow address: %s \n", response.EscrowInformation.Address)
	fmt.Printf("Reservation amount: %s %s\n", formatCurrency(totalAmount), response.EscrowInformation.Asset.Code())

	for _, detail := range response.EscrowInformation.Details {
		fmt.Println()
		fmt.Printf("FarmerID: %v\n", detail.FarmerID)
		fmt.Printf("Amount: %s\n", formatCurrency(detail.TotalAmount))
	}

	return nil
}

func formatCurrency(amount xdr.Int64) string {
	currency := big.NewRat(int64(amount), 1e7)
	return currency.FloatString(7)
}

func embed(schema interface{}, t provision.ReservationType, node string) (*provision.Reservation, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	r := &provision.Reservation{
		NodeID: node,
		Type:   t,
		Data:   raw,
	}

	return r, nil
}

func cmdsDeleteReservation(c *cli.Context) error {
	var (
		resID  = c.Int64("reservation")
		userID = mainui.ThreebotID
		//seedPath = c.GlobalString("seed")
	)

	reservation, err := bcdb.Workloads.Get(schema.ID(resID))
	if err != nil {
		return errors.Wrap(err, "failed to get reservation info")
	}

	signer, err := client.NewSigner(mainui.Key().PrivateKey.Seed())
	if err != nil {
		return errors.Wrapf(err, "failed to load signer")
	}

	_, signature, err := signer.SignHex(resID, reservation.Json)
	if err != nil {
		return errors.Wrap(err, "failed to sign the reservation")
	}

	if err := bcdb.Workloads.SignDelete(schema.ID(resID), schema.ID(userID), signature); err != nil {
		return errors.Wrapf(err, "failed to sign deletion of reservation: %d", resID)
	}

	fmt.Printf("Reservation %v marked as to be deleted\n", resID)
	return nil
}
