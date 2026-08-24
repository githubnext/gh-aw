//go:build !integration

package cli

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeAllocationCapacity(t *testing.T) {
	t.Parallel()

	t.Run("sums sizes when the result fits in int", func(t *testing.T) {
		assert.Equal(t, 5, safeAllocationCapacity(2, 3))
		assert.Equal(t, math.MaxInt, safeAllocationCapacity(math.MaxInt-1, 1))
	})

	t.Run("returns zero when the sum would overflow int", func(t *testing.T) {
		assert.Zero(t, safeAllocationCapacity(math.MaxInt, 1))
		assert.Zero(t, safeAllocationCapacity(math.MaxInt-1, 2))
	})

	t.Run("returns zero for negative parts", func(t *testing.T) {
		assert.Zero(t, safeAllocationCapacity(-1))
		assert.Zero(t, safeAllocationCapacity(2, -1))
	})
}
