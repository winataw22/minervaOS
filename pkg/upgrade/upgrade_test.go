package upgrade

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/0-fs/meta"
)

func TestUpgraderDownload(t *testing.T) {
	require := require.New(t)

	up := &Upgrader{
		cache: "/tmp/zfs-test-cache",
	}

	err := Storage(defaultHubStorage)(up)
	require.NoError(err)

	const flist = "azmy.3bot/test-flist.flist"

	store, err := up.getFlist(flist)
	require.NoError(err)
	tmp, err := ioutil.TempDir("", "download-*")

	require.NoError(err)
	defer os.RemoveAll(tmp)

	err = up.copyRecursive(store, tmp)
	require.NoError(err)

	// validation of download
	err = store.Walk("", func(path string, info meta.Meta) error {
		downloaded := filepath.Join(tmp, path)
		stat, err := os.Lstat(downloaded)
		require.NoError(err)
		require.Equal(info.IsDir(), stat.IsDir())
		if info.IsDir() {
			return nil
		}

		switch info.Info().Type {
		case meta.RegularType:
			require.Equal(info.Info().Size, uint64(stat.Size()))
		case meta.LinkType:
			require.Equal(os.ModeSymlink, stat.Mode()&os.ModeType)
		}

		return nil
	})

	require.NoError(err)
}
