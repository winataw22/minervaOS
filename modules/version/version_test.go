package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionParse(t *testing.T) {
	v, r, err := Parse("Version: master @Revision: some-revision-goes-here")
	require.NoError(t, err)

	require.Equal(t, "master", v)
	require.Equal(t, "some-revision-goes-here", r)

	v, r, err = Parse("Version: @Revision: some-revision-goes-here")
	require.NoError(t, err)

	require.Equal(t, "", v)
	require.Equal(t, "some-revision-goes-here", r)

	v, r, err = Parse("Version: master @Revision: some-revision-goes-here (dirty-repo)")
	require.NoError(t, err)

	require.Equal(t, "master", v)
	require.Equal(t, "some-revision-goes-here", r)

	_, _, err = Parse("invalid")
	require.Error(t, err)
}
