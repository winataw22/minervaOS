package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"

	"github.com/urfave/cli"
)

type multisigWallet struct {
	wallet stellar.Wallet
	kp     *keypair.Full
}

func create(c *cli.Context) error {
	seed := c.String("seed")
	network := c.String("network")
	asset := c.String("asset")
	from := c.String("from")
	destination := c.String("destination")
	amount := c.String("amount")

	kp, err := keypair.ParseFull(seed)
	if err != nil {
		return err
	}

	wallet, err := stellar.New(seed, network, nil)
	if err != nil {
		return err
	}
	msWallet := multisigWallet{
		wallet: *wallet,
		kp:     kp,
	}

	tx, err := msWallet.createMultisigTransaction(from, destination, amount, asset)
	if err != nil {
		return err
	}
	fmt.Printf("Transaction to be signed: %s\n", tx)
	return nil
}

func signAndSubmit(c *cli.Context) error {
	seed := c.String("seed")
	network := c.String("network")
	transaction := c.String("transaction")

	wallet, err := stellar.New(seed, network, nil)
	if err != nil {
		return err
	}
	msWallet := multisigWallet{wallet: *wallet}

	tx, err := msWallet.signAndSubmitMultisigTransaction(transaction)
	if err != nil {
		fmt.Printf("Transaction needs another signature: %s\n", tx)
		fmt.Println()
		return err
	}
	return nil
}

// createMultisigTransaction will create a multisig transaction from an address to a destination
// This is will be used in the multisig client
func (w *multisigWallet) createMultisigTransaction(from, destination, amount, assetCode string) (string, error) {
	sourceAccount, err := w.wallet.GetAccountDetails(from)
	if err != nil {
		return "", errors.Wrap(err, "failed to get source account")
	}

	asset, err := w.wallet.AssetFromCode(assetCode)
	if err != nil {
		return "", errors.Wrap(err, "could not load asset")
	}
	paymentOP := txnbuild.Payment{
		Destination: destination,
		Amount:      amount,
		Asset: txnbuild.CreditAsset{
			Code:   asset.Code(),
			Issuer: asset.Issuer(),
		},
	}

	tx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&paymentOP},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.wallet.GetNetworkPassPhrase(),
	}

	return tx.BuildSignEncode(w.kp)
}

// signAndSubmitMultisigTransaction signs off on a multisig transaction and tries to submit it to the network
func (w *multisigWallet) signAndSubmitMultisigTransaction(transaction string) (string, error) {
	tx, err := txnbuild.TransactionFromXDR(transaction)
	if err != nil {
		return "", errors.Wrap(err, "failed parse xdr to a transaction")
	}

	tx.Network = w.wallet.GetNetworkPassPhrase()

	err = tx.Sign(w.kp)
	if err != nil {
		return "", errors.Wrap(err, "failed to sign transaction")
	}

	client, err := w.wallet.GetHorizonClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to get horizon client")
	}

	txXDR, err := tx.Base64()
	if err != nil {
		return "", errors.Wrap(err, "failed to parse transaction to xdr")
	}
	// Submit the transaction
	_, err = client.SubmitTransaction(tx)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Debug().Msgf("%+v", hError.Problem.Extras)
		return txXDR, errors.Wrap(hError.Problem, "error submitting transaction, perhaps another signature is required")
	}
	return txXDR, nil
}
