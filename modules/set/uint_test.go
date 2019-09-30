package set

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	s := NewUint()
	assert.NotNil(t, s.m)
}

func TestAdd(t *testing.T) {
	s := NewUint()

	for i := 0; i < 10; i++ {
		err := s.Add(uint(i))
		assert.NoError(t, err)
	}

	for i := 0; i < 10; i++ {
		_, ok := s.m[uint(i)]
		assert.True(t, ok, "%d should be present in the map", i)
	}

	err := s.Add(0)
	assert.Error(t, err, "adding already present number should return an error")
	assert.True(t, errors.Is(err, ErrConflict{}))
}

func TestRemove(t *testing.T) {
	s := NewUint()

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
	s := NewUint()

	err := s.Add(1)
	assert.NoError(t, err)
	err = s.Add(2)
	assert.NoError(t, err)
	err = s.Add(3)
	assert.NoError(t, err)

	l := s.List()
	expected := []uint{1, 2, 3}
	assert.Equal(t, len(expected), len(l))
	assert.Subset(t, expected, l)
}

func TestConcurent(t *testing.T) {
	// TODO
}
