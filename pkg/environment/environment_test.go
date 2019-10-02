package environment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/kernel"
)

// There are no test against GetEnvironment since the
// result cannot be deterministic if you have kernel
// argument set or not
func TestManager(t *testing.T) {
	// Development mode
	params := kernel.Params{"runmode": {"dev"}}
	value := getEnvironmentFromParams(params)

	assert.Equal(t, RunningDev, value.RunningMode)

	// Testing mode
	params = kernel.Params{"runmode": {"test"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, RunningTest, value.RunningMode)

	// Main mode
	params = kernel.Params{"runmode": {"prod"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, RunningMain, value.RunningMode)

	// Fallback
	params = kernel.Params{"nope": {"lulz"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, RunningMain, value.RunningMode)

	// Fallback on undefined
	params = kernel.Params{"runmode": {"dunno"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.RunningMode, RunningMain)
}

func TestEnvironmentOverride(t *testing.T) {
	os.Setenv("ZOS_BCDB_URL", "localhost:1234")

	params := kernel.Params{"runmode": {"dev"}}
	value := getEnvironmentFromParams(params)

	assert.Equal(t, value.BcdbURL, "localhost:1234")
}
