//go:build !integration

package workflow

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeAllocationCapacity(t *testing.T) {
	t.Run("handles zero inputs", func(t *testing.T) {
		assert.Zero(t, safeAllocationCapacity())
		assert.Zero(t, safeAllocationCapacity(0, 0))
		assert.Equal(t, 5, safeAllocationCapacity(0, 5))
		assert.Equal(t, 5, safeAllocationCapacity(5, 0))
	})

	t.Run("sums sizes when the result fits in int", func(t *testing.T) {
		assert.Equal(t, 5, safeAllocationCapacity(2, 3))
		assert.Equal(t, 6000, safeAllocationCapacity(1000, 5000))
		assert.Equal(t, math.MaxInt, safeAllocationCapacity(math.MaxInt-1, 1))
		assert.Equal(t, math.MaxInt, safeAllocationCapacity(math.MaxInt-2, 1, 1))
	})

	t.Run("returns zero when the sum would overflow int", func(t *testing.T) {
		assert.Zero(t, safeAllocationCapacity(math.MaxInt, 1))
		assert.Zero(t, safeAllocationCapacity(math.MaxInt-1, 2))
		assert.Zero(t, safeAllocationCapacity(math.MaxInt-2, 2, 1))
	})

	t.Run("returns zero for negative parts", func(t *testing.T) {
		assert.Zero(t, safeAllocationCapacity(-1))
		assert.Zero(t, safeAllocationCapacity(2, -1))
	})
}
