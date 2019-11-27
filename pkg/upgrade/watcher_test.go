package upgrade

import "testing"

import "context"

import "github.com/stretchr/testify/require"

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
