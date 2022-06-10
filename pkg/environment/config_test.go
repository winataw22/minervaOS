package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigMerge(t *testing.T) {
	var cfg Config
	cfg.Yggdrasil.Peers = []string{"a", "b"}

	var ex Config
	ex.Yggdrasil.Peers = []string{"c", "b"}

	cfg.Merge(ex)

	require.Equal(t, []string{"a", "b", "c"}, cfg.Yggdrasil.Peers)
}

func TestConfigMergeEmpty(t *testing.T) {
	var cfg Config
	cfg.Yggdrasil.Peers = []string{"a", "b"}

	var ex Config

	cfg.Merge(ex)

	require.Equal(t, []string{"a", "b"}, cfg.Yggdrasil.Peers)
}
