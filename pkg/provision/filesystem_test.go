package provision

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"gotest.tools/assert"
)

func TestFilesystemNameFromReservation(t *testing.T) {
	r := Reservation{
		ID: "123-1",
	}

	name := FilesystemName(r)
	assert.Equal(t, r.ID, name)

	id, err := WorkloadIDFromFilesystem(pkg.Filesystem{
		Name: name,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(123), id)
}
