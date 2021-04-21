package environment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/kernel"
)

// There are no test against GetEnvironment since the
// result cannot be deterministic if you have kernel
// argument set or not
func TestManager(t *testing.T) {
	// Development mode
	params := kernel.Params{"runmode": {"dev"}}
	value, err := getEnvironmentFromParams(params)
	require.NoError(t, err)

	assert.Equal(t, RunningDev, value.RunningMode)

	// Testing mode
	params = kernel.Params{"runmode": {"test"}}
	value, err = getEnvironmentFromParams(params)
	require.NoError(t, err)

	assert.Equal(t, RunningTest, value.RunningMode)

	// Main mode
	params = kernel.Params{"runmode": {"prod"}}
	value, err = getEnvironmentFromParams(params)
	require.NoError(t, err)

	assert.Equal(t, RunningMain, value.RunningMode)

	// Fallback
	params = kernel.Params{"nope": {"lulz"}}
	value, err = getEnvironmentFromParams(params)
	require.NoError(t, err)

	assert.Equal(t, RunningMain, value.RunningMode)

	// Fallback on undefined
	params = kernel.Params{"runmode": {"dunno"}}
	value, err = getEnvironmentFromParams(params)
	require.NoError(t, err)

	assert.Equal(t, value.RunningMode, RunningMain)
}

func TestEnvironmentOverride(t *testing.T) {
	os.Setenv("ZOS_SUBSTRATE_URL", "localhost:1234")

	params := kernel.Params{"runmode": {"dev3"}}
	value, err := getEnvironmentFromParams(params)
	require.NoError(t, err)

	assert.Equal(t, value.SubstrateURL, "localhost:1234")
}
