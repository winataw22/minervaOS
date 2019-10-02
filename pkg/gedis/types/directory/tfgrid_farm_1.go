package directory

import (
	schema "github.com/threefoldtech/zos/pkg/schema"
)

//TfgridFarm1 jsx schema
type TfgridFarm1 struct {
	ID              int64                      `json:"id,omitempty"`
	ThreebotID      string                     `json:"threebot_id"`
	IyoOrganization string                     `json:"iyo_organization"`
	Name            string                     `json:"name"`
	WalletAddresses []string                   `json:"wallet_addresses"`
	Location        TfgridLocation1            `json:"location"`
	Email           string                     `json:"email"`
	ResourcePrices  []TfgridNodeResourcePrice1 `json:"resource_prices"`
	PrefixZero      schema.IPRange             `json:"prefix_zero"`
}

//TfgridNodeResourcePrice1 jsx schema
type TfgridNodeResourcePrice1 struct {
	Currency TfgridNodeResourcePrice1CurrencyEnum `json:"currency"`
	Cru      float64                              `json:"cru"`
	Mru      float64                              `json:"mru"`
	Hru      float64                              `json:"hru"`
	Sru      float64                              `json:"sru"`
	Nru      float64                              `json:"nru"`
}

//TfgridNodeResourcePrice1CurrencyEnum jsx schema
type TfgridNodeResourcePrice1CurrencyEnum uint8

// TfgridNodeResourcePrice1CurrencyEnum jsx schema
const (
	TfgridNodeResourcePrice1CurrencyEUR TfgridNodeResourcePrice1CurrencyEnum = iota
	TfgridNodeResourcePrice1CurrencyUSD
	TfgridNodeResourcePrice1CurrencyTFT
	TfgridNodeResourcePrice1CurrencyAED
	TfgridNodeResourcePrice1CurrencyGBP
)

// String implement stringer interface
func (e TfgridNodeResourcePrice1CurrencyEnum) String() string {
	switch e {
	case TfgridNodeResourcePrice1CurrencyEUR:
		return "EUR"
	case TfgridNodeResourcePrice1CurrencyUSD:
		return "USD"
	case TfgridNodeResourcePrice1CurrencyTFT:
		return "TFT"
	case TfgridNodeResourcePrice1CurrencyAED:
		return "AED"
	case TfgridNodeResourcePrice1CurrencyGBP:
		return "GBP"
	}
	return "UNKNOWN"
}
