package primitives

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterDecrement(t *testing.T) {
	t.Run("regular_decrement", func(t *testing.T) {
		counter := CounterUint64(0)
		counter.Increment(5)
		dec := counter.Decrement(3)
		assert.Equal(t, dec, uint64(2))
		assert.Equal(t, counter.Current(), uint64(2))
	})
	t.Run("decrement_to_zero", func(t *testing.T) {
		counter := CounterUint64(0)
		counter.Increment(5)
		dec := counter.Decrement(5)
		assert.Equal(t, dec, uint64(0))
		assert.Equal(t, counter.Current(), uint64(0))
	})
	t.Run("decrement_negative", func(t *testing.T) {
		counter := CounterUint64(0)
		counter.Increment(5)
		dec := counter.Decrement(7)
		assert.Equal(t, dec, uint64(0))
		assert.Equal(t, counter.Current(), uint64(0))
	})
}
