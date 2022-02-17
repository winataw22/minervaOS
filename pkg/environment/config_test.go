package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigMerge(t *testing.T) {
	var cfg Config
	cfg.Monitor = []uint32{0, 1}
	cfg.Yggdrasil.Peers = []string{"a", "b"}

	var ex Config
	ex.Monitor = []uint32{1, 2}
	ex.Yggdrasil.Peers = []string{"c", "b"}

	cfg.Merge(ex)

	require.Equal(t, []string{"a", "b", "c"}, cfg.Yggdrasil.Peers)
	require.Len(t, cfg.Monitor, 3)
}

func TestConfigMergeEmpty(t *testing.T) {
	var cfg Config
	cfg.Monitor = []uint32{0, 1}
	cfg.Yggdrasil.Peers = []string{"a", "b"}

	var ex Config

	cfg.Merge(ex)

	require.Equal(t, []string{"a", "b"}, cfg.Yggdrasil.Peers)
	require.Len(t, cfg.Monitor, 2)
}
