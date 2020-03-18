package set

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDir(t *testing.T) string {
	td, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	path := filepath.Join(td, "wireguard_ports")
	err = os.MkdirAll(path, 0770)
	require.NoError(t, err)

	return path
}

func TestNew(t *testing.T) {
	td := testDir(t)

	s := NewUint(td)
	assert.Equal(t, td, s.root)
}

func TestAdd(t *testing.T) {
	td := testDir(t)

	s := NewUint(td)

	for i := 0; i < 10; i++ {
		err := s.Add(uint(i))
		assert.NoError(t, err)
	}

	infos, err := ioutil.ReadDir(s.root)
	require.NoError(t, err)

	assert.EqualValues(t, 10, len(infos))

	err = s.Add(0)
	assert.Error(t, err, "adding already present number should return an error")
	assert.True(t, errors.Is(err, ErrConflict{}))
}

func TestRemove(t *testing.T) {
	td := testDir(t)

	s := NewUint(td)

	for i := 0; i < 10; i++ {
		err := s.Add(uint(i))
		assert.NoError(t, err)
	}

	for i := 0; i < 10; i++ {
		s.Remove(uint(i))
	}

	infos, err := ioutil.ReadDir(s.root)
	require.NoError(t, err)

	assert.EqualValues(t, 0, len(infos))

	s.Remove(99999) //ensure remove never panics
}

func TestList(t *testing.T) {
	td := testDir(t)

	s := NewUint(td)

	err := s.Add(1)
	assert.NoError(t, err)
	err = s.Add(2)
	assert.NoError(t, err)
	err = s.Add(3)
	assert.NoError(t, err)

	l, err := s.List()
	require.NoError(t, err)

	expected := []uint{1, 2, 3}
	assert.Equal(t, len(expected), len(l))
	assert.Subset(t, expected, l)
}
