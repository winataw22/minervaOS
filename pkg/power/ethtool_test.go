package power

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	const input = `Settings for enp6s0f0:
	Supported ports: [ TP ]
	Supported link modes:   10baseT/Half 10baseT/Full
	                        100baseT/Half 100baseT/Full
	                        1000baseT/Full
	Supported pause frame use: Symmetric
	Supports auto-negotiation: Yes
	Supported FEC modes: Not reported
	Advertised link modes:  10baseT/Half 10baseT/Full
	                        100baseT/Half 100baseT/Full
	                        1000baseT/Full
	Advertised pause frame use: Symmetric
	Advertised auto-negotiation: Yes
	Advertised FEC modes: Not reported
	Speed: 1000Mb/s
	Duplex: Full
	Port: Twisted Pair
	PHYAD: 1
	Transceiver: internal
	Auto-negotiation: on
	MDI-X: off (auto)
	Supports Wake-on: pumbg
	Wake-on: g
	Current message level: 0x00000007 (7)
			       drv probe link
	Link detected: yes
	`

	value, err := valueOfFlag([]byte(input), SupportsWakeOn)
	require.NoError(t, err)

	require.Equal(t, "pumbg", value)
}

func TestParseFlagNotSet(t *testing.T) {
	const input = `Settings for enp6s0f0:
	Supported ports: [ TP ]
	Supported link modes:   10baseT/Half 10baseT/Full
	                        100baseT/Half 100baseT/Full
	                        1000baseT/Full
	Supported pause frame use: Symmetric
	Supports auto-negotiation: Yes
	Supported FEC modes: Not reported
	Advertised link modes:  10baseT/Half 10baseT/Full
	                        100baseT/Half 100baseT/Full
	                        1000baseT/Full
	Advertised pause frame use: Symmetric
	Advertised auto-negotiation: Yes
	Advertised FEC modes: Not reported
	Speed: 1000Mb/s
	Duplex: Full
	Port: Twisted Pair
	PHYAD: 1
	Transceiver: internal
	Auto-negotiation: on
	MDI-X: off (auto)
	Wake-on: g
	Current message level: 0x00000007 (7)
			       drv probe link
	Link detected: yes
	`

	_, err := valueOfFlag([]byte(input), SupportsWakeOn)
	require.ErrorIs(t, err, ErrFlagNotFound)

}
