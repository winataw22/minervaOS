package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFListWatcher(t *testing.T) {
	require := require.New(t)

	watcher := FListSemverWatcher{
		FList: "tf-zos/zos:development:latest.flist",
	}

	ctx := context.Background()

	ch, err := watcher.Watch(ctx)
	require.NoError(err)

	event := <-ch
	require.Equal(FList, event.EventType())

	info, ok := event.(*FListEvent)
	require.True(ok)

	require.Equal("zos:development:latest.flist", info.Name)
	require.Equal("symlink", info.Type)
}

func TestRepoWatcher(t *testing.T) {
	require := require.New(t)

	watcher := FListRepoWatcher{
		Repo: "tf-zos-bins",
	}

	ctx := context.Background()

	ch, err := watcher.Watch(ctx)
	require.NoError(err)

	event := <-ch
	require.Equal(Repo, event.EventType())

	info, ok := event.(*RepoEvent)
	require.True(ok)

	require.Equal("tf-zos-bins", info.Repo)
	require.Len(info.ToDel, 0) // we starting from empty Current
	require.True(len(info.ToAdd) > 0)
}
