package config

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
)

// Settings struct
type Settings struct {
	Network string
}

var (
	// Config is global explorer config
	Config Settings

	possibleNetworks = []string{stellar.NetworkProduction, stellar.NetworkTest}
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
	if Config.Network != "" && !in(Config.Network, possibleNetworks) {
		return fmt.Errorf("invalid network '%s'", Config.Network)
	}

	return nil
}
