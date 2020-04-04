package stellar

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	hProtocol "github.com/stellar/go/protocols/horizon"
	horizoneffects "github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg/schema"
)

const (
	// TFTCode is the asset code for TFT on stellar
	TFTCode          = "TFT"
	tftIssuerTestnet = "GA47YZA3PKFUZMPLQ3B5F2E3CJIB57TGGU7SPCQT2WAEYKN766PWIMB3"
	tftIssuerProd    = "GBOVQKJYHXRR3DX6NOX2RRYFRCUMSADGDESTDNBDS6CDVLGVESRTAC47"

	// FreeTFTCode is the asset code for TFT on stellar
	FreeTFTCode          = "FreeTFT"
	freeTftIssuerTestnet = "GBLDUINEFYTF7XEE7YNWA3JQS4K2VD37YU7I2YAE7R5AHZDKQXSS2J6R"
	freeTftIssuerProd    = "GCBGS5TFE2BPPUVY55ZPEMWWGR6CLQ7T6P46SOFGHXEBJ34MSP6HVEUT"

	stellarPrecision       = 1e7
	stellarPrecisionDigits = 7

	// NetworkProduction uses stellar production network
	NetworkProduction = "production"
	// NetworkTest uses stellar test network
	NetworkTest = "testnet"
	// NetworkDebug doesn't do validation, and always address validation is skipped
	// Only supported by the AddressValidator
	NetworkDebug = "debug"
)

type assetCodeEnum string

const (
	// TFT assetcode
	TFT assetCodeEnum = TFTCode
	// FreeTFT assetcode
	FreeTFT assetCodeEnum = FreeTFTCode
)

// ErrInsuficientBalance is an error that is used when there is insufficient balance
var ErrInsuficientBalance = errors.New("insuficient balance")

// Wallet is the foundation wallet
// Payments will be funded and fees will be taken with this wallet
type Wallet struct {
	keypair *keypair.Full
	network string
	asset   assetCodeEnum
}

// New from seed
func New(seed string, network string, asset string) (*Wallet, error) {
	kp, err := keypair.ParseFull(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		keypair: kp,
		network: network,
		asset:   assetCodeEnum(asset),
	}, nil
}

// CreateAccount and activates
func (w *Wallet) CreateAccount() (keypair.Full, error) {
	client, err := w.getHorizonClient()
	if err != nil {
		return keypair.Full{}, err
	}
	newKp, err := keypair.Random()
	if err != nil {
		return keypair.Full{}, err
	}

	sourceAccount, err := w.getAccountDetails(w.keypair.Address())
	if err != nil {
		return keypair.Full{}, errors.Wrap(err, "failed to get source account")
	}
	createAccountOp := txnbuild.CreateAccount{
		Destination: newKp.Address(),
		Amount:      "10",
	}
	tx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&createAccountOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.getNetworkPassPhrase(),
	}

	txeBase64, err := tx.BuildSignEncode(w.keypair)
	if err != nil {
		return keypair.Full{}, errors.Wrap(err, "failed to get build transaction")
	}

	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return keypair.Full{}, errors.Wrap(hError, "error submitting transaction")
	}

	// Set the trustline
	sourceAccount, err = w.getAccountDetails(newKp.Address())
	if err != nil {
		return keypair.Full{}, errors.Wrap(err, "failed to get account details")
	}
	changeTrustOp := txnbuild.ChangeTrust{
		SourceAccount: &sourceAccount,
		Line: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.getIssuer(),
		},
	}
	trustTx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&changeTrustOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.getNetworkPassPhrase(),
	}

	txeBase64, err = trustTx.BuildSignEncode(newKp)
	if err != nil {
		return keypair.Full{}, errors.Wrap(err, "failed to get build transaction")
	}

	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return keypair.Full{}, errors.Wrap(hError.Problem, "error submitting transaction")
	}

	return *newKp, nil
}

// KeyPairFromSeed parses a seed and creates a keypair for it, which can be
// used to sign transactions
func (w *Wallet) KeyPairFromSeed(seed string) (*keypair.Full, error) {
	return keypair.ParseFull(seed)
}

// GetBalance gets balance for an address and a given reservation id. It also returns
// a list of addresses which funded the given address.
func (w *Wallet) GetBalance(address string, id schema.ID) (xdr.Int64, []string, error) {
	var total xdr.Int64
	horizonClient, err := w.getHorizonClient()
	if err != nil {
		return 0, nil, err
	}

	cursor := ""

	txReq := horizonclient.TransactionRequest{
		ForAccount: address,
		Cursor:     cursor,
	}

	txes, err := horizonClient.Transactions(txReq)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not get transactions")
	}

	donors := make(map[string]struct{})
	for len(txes.Embedded.Records) != 0 {
		for _, tx := range txes.Embedded.Records {
			if tx.Memo == strconv.FormatInt(int64(id), 10) {
				effectsReq := horizonclient.EffectRequest{
					ForTransaction: tx.Hash,
				}
				effects, err := horizonClient.Effects(effectsReq)
				if err != nil {
					log.Error().Err(err).Msgf("failed to get transaction effects")
					continue
				}
				// first check if we have been paid
				var isFunding bool
				for _, effect := range effects.Embedded.Records {
					if effect.GetAccount() != address {
						continue
					}
					if effect.GetType() == "account_credited" {
						isFunding = true
						creditedEffect := effect.(horizoneffects.AccountCredited)
						parsedAmount, err := amount.Parse(creditedEffect.Amount)
						if err != nil {
							continue
						}
						total += parsedAmount
					} else if effect.GetType() == "account_debited" {
						isFunding = false
						debitedEffect := effect.(horizoneffects.AccountDebited)
						parsedAmount, err := amount.Parse(debitedEffect.Amount)
						if err != nil {
							continue
						}
						total -= parsedAmount
					}
				}
				if isFunding {
					for _, effect := range effects.Embedded.Records {
						if effect.GetType() == "account_debited" && effect.GetAccount() != address {
							donors[effect.GetAccount()] = struct{}{}
						}
					}
				}
			}
			cursor = tx.PagingToken()
		}
		txReq.Cursor = cursor
		txes, err = horizonClient.Transactions(txReq)
		if err != nil {
			return 0, nil, errors.Wrap(err, "could not get transactions")
		}
	}

	donorList := []string{}
	for donor := range donors {
		donorList = append(donorList, donor)
	}
	log.Info().
		Int64("balance", int64(total)).
		Str("address", address).
		Int64("id", int64(id)).Msgf("status of balance for reservation")
	return total, donorList, nil
}

// Refund using a keypair
// keypair is account associated with farmer - user
// refund destination is the first address in the "funder" list as returned by
// GetBalance
// id is the reservation ID to refund for
func (w *Wallet) Refund(keypair keypair.Full, id schema.ID) error {
	sourceAccount, err := w.getAccountDetails(keypair.Address())
	if err != nil {
		return errors.Wrap(err, "failed to get source account")
	}
	amount, funders, err := w.GetBalance(keypair.Address(), id)
	if err != nil {
		return errors.Wrap(err, "failed to get balance")
	}
	// if no balance for this reservation, do nothing
	if amount == 0 {
		return nil
	}
	destination := funders[0]

	paymentOP := txnbuild.Payment{
		Destination: destination,
		Amount:      big.NewRat(int64(amount), stellarPrecision).FloatString(stellarPrecisionDigits),
		Asset: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.getIssuer(),
		},
		SourceAccount: &sourceAccount,
	}

	formattedMemo := fmt.Sprintf("%d", id)
	memo := txnbuild.MemoText(formattedMemo)
	tx := txnbuild.Transaction{
		Operations: []txnbuild.Operation{&paymentOP},
		Timebounds: txnbuild.NewTimeout(300),
		Network:    w.getNetworkPassPhrase(),
		Memo:       memo,
	}

	fundedTx, err := w.fundTransaction(&tx)
	if err != nil {
		return errors.Wrap(err, "failed to fund transaction")
	}

	err = w.signAndSubmitTx(&keypair, fundedTx)
	if err != nil {
		return errors.Wrap(err, "failed to sign and submit transaction")
	}
	return nil
}

// PayoutFarmer using a keypair
// keypair is account assiociated with farmer - user
// destination is the farmer destination address
// id is the reservation ID to pay for
func (w *Wallet) PayoutFarmer(keypair keypair.Full, destination string, amount xdr.Int64, id schema.ID) error {
	sourceAccount, err := w.getAccountDetails(keypair.Address())
	if err != nil {
		return errors.Wrap(err, "failed to get source account")
	}
	balance, _, err := w.GetBalance(keypair.Address(), id)
	if err != nil {
		return errors.Wrap(err, "failed to get balance")
	}
	if balance < amount {
		return ErrInsuficientBalance
	}

	// 10% cut for the foundation
	/*
		Based on the way we calculate the cost of reservation we know it has at most
		6 digit precision whereas stellar has 7 digits precision.
		This means that any valid reservation must necessarily have a "0" as least
		significant digit (when expressed as `stropes` as is the case here).
		With this knowledge it is safe to perform the 90% cut as regular integer operations
		instead of using floating points which might lead to floating point errors
	*/
	if amount%10 != 0 {
		return errors.New("invalid reservation cost")
	}
	foundationCut := amount / 10 * 1
	amountDue := amount / 10 * 9

	farmerPaymentOP := txnbuild.Payment{
		Destination: destination,
		Amount:      big.NewRat(int64(amountDue), stellarPrecision).FloatString(stellarPrecisionDigits),
		Asset: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.getIssuer(),
		},
		SourceAccount: &sourceAccount,
	}
	foundationPaymentOP := txnbuild.Payment{
		Destination: w.keypair.Address(),
		Amount:      big.NewRat(int64(foundationCut), stellarPrecision).FloatString(stellarPrecisionDigits),
		Asset: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.getIssuer(),
		},
		SourceAccount: &sourceAccount,
	}

	formattedMemo := fmt.Sprintf("%d", id)
	memo := txnbuild.MemoText(formattedMemo)
	tx := txnbuild.Transaction{
		Operations: []txnbuild.Operation{&farmerPaymentOP, &foundationPaymentOP},
		Timebounds: txnbuild.NewTimeout(300),
		Network:    w.getNetworkPassPhrase(),
		Memo:       memo,
	}

	fundedTx, err := w.fundTransaction(&tx)
	if err != nil {
		return errors.Wrap(err, "failed to fund transaction")
	}

	err = w.signAndSubmitTx(&keypair, fundedTx)
	if err != nil {
		return errors.Wrap(err, "failed to sign and submit transaction")
	}
	return nil
}

// fundTransaction funds a transaction with the foundation wallet
// For every operation in the transaction, the fee will be paid by the foundation wallet
func (w *Wallet) fundTransaction(tx *txnbuild.Transaction) (*txnbuild.Transaction, error) {
	sourceAccount, err := w.getAccountDetails(w.keypair.Address())
	if err != nil {
		return &txnbuild.Transaction{}, errors.Wrap(err, "failed to get source account")
	}

	// set the source account of the tx to the foundation account
	tx.SourceAccount = &sourceAccount

	if len(tx.Operations) == 0 {
		return &txnbuild.Transaction{}, errors.New("no operations were set on the transaction")
	}

	// calculate total fee based on the operations in the transaction
	tx.BaseFee = tx.BaseFee * uint32(len(tx.Operations))
	err = tx.Build()
	if err != nil {
		return &txnbuild.Transaction{}, errors.Wrap(err, "failed to build transaction")
	}

	err = tx.Sign(w.keypair)
	if err != nil {
		return &txnbuild.Transaction{}, errors.Wrap(err, "failed to sign transaction")
	}

	return tx, nil
}

// signAndSubmitTx sings of on a transaction with a given keypair
// and submits it to the network
func (w *Wallet) signAndSubmitTx(keypair *keypair.Full, tx *txnbuild.Transaction) error {
	client, err := w.getHorizonClient()
	if err != nil {
		return errors.Wrap(err, "failed to get horizon client")
	}

	err = tx.Sign(keypair)
	if err != nil {
		return errors.Wrap(err, "failed to sign transaction with keypair")
	}

	// Submit the transaction
	_, err = client.SubmitTransaction(*tx)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Debug().
			Err(fmt.Errorf("%+v", hError.Problem.Extras)).
			Msg("error submitting transaction")
		return errors.Wrap(hError.Problem, "error submitting transaction")
	}
	return nil
}

func (w *Wallet) getAccountDetails(address string) (account hProtocol.Account, err error) {
	client, err := w.getHorizonClient()
	if err != nil {
		return hProtocol.Account{}, err
	}
	ar := horizonclient.AccountRequest{AccountID: address}
	account, err = client.AccountDetail(ar)
	return
}

func (w *Wallet) getHorizonClient() (*horizonclient.Client, error) {
	switch w.network {
	case "testnet":
		return horizonclient.DefaultTestNetClient, nil
	case "production":
		return horizonclient.DefaultPublicNetClient, nil
	default:
		return nil, errors.New("network is not supported")
	}
}

func (w *Wallet) getIssuer() string {
	switch w.asset {
	case TFT:
		switch w.network {
		case "testnet":
			return tftIssuerTestnet
		case "production":
			return tftIssuerProd
		default:
			return tftIssuerTestnet
		}
	case FreeTFT:
		switch w.network {
		case "testnet":
			return freeTftIssuerTestnet
		case "production":
			return freeTftIssuerProd
		default:
			return freeTftIssuerTestnet
		}
	default:
		return tftIssuerTestnet
	}
}

func (w *Wallet) getNetworkPassPhrase() string {
	switch w.network {
	case "testnet":
		return network.TestNetworkPassphrase
	case "production":
		return network.PublicNetworkPassphrase
	default:
		return network.TestNetworkPassphrase
	}
}

func (e assetCodeEnum) String() string {
	switch e {
	case TFT:
		return TFTCode
	case FreeTFT:
		return FreeTFTCode
	}
	return "UNKNOWN"
}
