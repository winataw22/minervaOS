package hub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHub(t *testing.T) {
	const tag = "3b51aa5"
	const repo = "tf-autobuilder"

	hub := NewHubClient(defaultHubCallTimeout)

	results, err := hub.ListTag(repo, tag)
	require.NoError(t, err)
	require.Len(t, results, 20)

	var zos Symlink
	for _, f := range results {
		if f.Name == "zos.flist" {
			zos = f
			break
		}
	}

	// symlink starts from repo, and can point us to
	// another repo destRepo
	destRepo, name, err := zos.Destination(repo)
	require.NoError(t, err)

	info, err := hub.Info(destRepo, name)
	require.NoError(t, err)

	regular := NewRegular(info)

	files, err := regular.Files(repo)
	require.NoError(t, err)
	require.NotEmpty(t, files)
}
