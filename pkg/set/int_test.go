package set

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := NewInt()
	assert.NotNil(t, s.m)
}

func TestAdd(t *testing.T) {
	s := NewInt()

	for i := 0; i < 10; i++ {
		err := s.Add(uint(i))
		assert.NoError(t, err)
	}

	assert.EqualValues(t, 10, len(s.m))

	err := s.Add(0)
	assert.Error(t, err, "adding already present number should return an error")
	assert.True(t, errors.Is(err, ErrConflict{}))
}

func TestRemove(t *testing.T) {
	s := NewInt()

	for i := 0; i < 10; i++ {
		err := s.Add(uint(i))
		assert.NoError(t, err)
	}

	for i := 0; i < 10; i++ {
		s.Remove(uint(i))
	}

	assert.EqualValues(t, 0, len(s.m))

	s.Remove(99999) //ensure remove never panics
}

func TestList(t *testing.T) {
	s := NewInt()

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
