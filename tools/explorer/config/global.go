package config

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
)

// Settings struct
type Settings struct {
	Network string
	Asset   string
	Seed    string
}

var (
	// Config is global explorer config
	Config Settings

	possibleNetworks = []string{stellar.NetworkProduction, stellar.NetworkTest}
	possibleAssets   = []string{stellar.TFTCode, stellar.FreeTFTCode}
)

// Valid checks if Config is filled with valid data
func Valid() error {
	in := func(s string, l []string) bool {
		for _, a := range l {
			if strings.EqualFold(s, a) {
				return true
			}
		}
		return false
	}
	if !in(Config.Network, possibleNetworks) {
		return fmt.Errorf("invalid network '%s'", Config.Network)
	}

	if !in(Config.Asset, possibleAssets) {
		return fmt.Errorf("invalid asset '%s'", Config.Asset)
	}

	return nil
}
