package flist

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheCleaner(t *testing.T) {

	cache, err := ioutil.TempDir("", "clean-cache-*")

	defer os.RemoveAll(cache)

	require.NoError(t, err)
	flister := flistModule{
		cache: cache,
	}

	// fixing the now
	now := time.Now()

	create := func(name string, atime time.Time) error {
		file, err := os.Create(name)
		if err != nil {
			return err
		}

		file.Close()

		cmd := exec.Command("touch",
			"-a",
			// format from touch man page is [[CC]YY]MMDDhhmm[.ss]
			"-t", atime.Format("200601021504"),
			name)

		return cmd.Run()

	}

	count := 100
	// each file is one day older than previous one.
	for i := 0; i < count; i++ {
		name := filepath.Join(cache, fmt.Sprintf("file-%02d", i))
		atime := now.Add(time.Duration(-i*24) * time.Hour)
		err := create(name, atime)
		require.NoError(t, err)
	}

	err = flister.cleanCache(now, 50*24*time.Hour) // this should delete 50 files!
	require.NoError(t, err)

	files, err := ioutil.ReadDir(cache)
	require.NoError(t, err)

	assert.Len(t, files, 50)

	assert.Equal(t, "file-00", files[0].Name())
	assert.Equal(t, "file-49", files[49].Name())
}
