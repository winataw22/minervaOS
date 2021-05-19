package primitives

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestCounterDecrement(t *testing.T) {
	t.Run("regular_decrement", func(t *testing.T) {
		counter := AtomicUnit(0)
		counter.Increment(5)
		dec := counter.Decrement(3)
		assert.Equal(t, dec, gridtypes.Unit(2))
		assert.Equal(t, counter.Current(), gridtypes.Unit(2))
	})
	t.Run("decrement_to_zero", func(t *testing.T) {
		counter := AtomicUnit(0)
		counter.Increment(5)
		dec := counter.Decrement(5)
		assert.Equal(t, dec, gridtypes.Unit(0))
		assert.Equal(t, counter.Current(), gridtypes.Unit(0))
	})
	t.Run("decrement_negative", func(t *testing.T) {
		counter := AtomicUnit(0)
		counter.Increment(5)
		dec := counter.Decrement(7)
		assert.Equal(t, dec, gridtypes.Unit(0))
		assert.Equal(t, counter.Current(), gridtypes.Unit(0))
	})
}
