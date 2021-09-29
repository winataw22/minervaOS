// Package zinit exposes function to interat with zinit service life cyle management
package zinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestParseService(t *testing.T) {
	b := []byte(`
exec: /bin/true
test: test -e /bin/true
oneshot: false
log: ring
after:
 - one
 - two
`)
	var s InitService
	err := yaml.Unmarshal(b, &s)
	require.NoError(t, err)

	assert.Equal(t, InitService{
		Exec:    "/bin/true",
		Test:    "test -e /bin/true",
		Oneshot: false,
		Log:     RingLogType,
		After:   []string{"one", "two"},
	}, s)
}
