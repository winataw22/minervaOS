package stellar

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// AddressValidator validates stellar address
type AddressValidator struct {
	network string
	asset   Asset
}

// NewAddressValidator creates an address validator instance
func NewAddressValidator(network, assetCode string) (*AddressValidator, error) {
	w, err := New("", network, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create wallet")
	}
	asset, err := w.AssetFromCode(assetCode)
	if err != nil {
		return nil, errors.Wrap(err, "could not load asset code")
	}
	return &AddressValidator{network: network, asset: asset}, nil
}

// Valid validates a stellar address, and only return nil if address is valid
func (a *AddressValidator) Valid(address string) error {
	if a.network == NetworkDebug {
		return nil
	}

	account, err := a.getAccountDetails(address)
	if err != nil {
		return errors.Wrap(err, "invalid account address")
	}

	issuer := a.asset.Issuer()

	for _, balance := range account.Balances {
		if balance.Code != a.asset.Code() || balance.Issuer != issuer {
			continue
		}
		limit, err := strconv.ParseFloat(balance.Limit, 64)
		if err != nil {
			//probably an empty string.
			continue
		}
		if limit > 0 {
			//valid address
			return nil
		}
	}

	return fmt.Errorf("addess has no trustline")
}

func (a *AddressValidator) getAccountDetails(address string) (account hProtocol.Account, err error) {
	client, err := a.getHorizonClient()
	if err != nil {
		return hProtocol.Account{}, err
	}
	ar := horizonclient.AccountRequest{AccountID: address}
	account, err = client.AccountDetail(ar)
	return
}

func (a *AddressValidator) getHorizonClient() (*horizonclient.Client, error) {
	switch a.network {
	case NetworkTest:
		return horizonclient.DefaultTestNetClient, nil
	case NetworkProduction:
		return horizonclient.DefaultPublicNetClient, nil
	default:
		return nil, errors.New("network is not supported")
	}
}
