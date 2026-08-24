//go:build !integration

package typeutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeAllocationCapacity(t *testing.T) {
	t.Parallel()

	t.Run("sums sizes when the result fits in int", func(t *testing.T) {
		assert.Equal(t, 5, SafeAllocationCapacity(2, 3))
		assert.Equal(t, math.MaxInt, SafeAllocationCapacity(math.MaxInt-1, 1))
	})

	t.Run("returns zero when the sum would overflow int", func(t *testing.T) {
		assert.Zero(t, SafeAllocationCapacity(math.MaxInt, 1))
		assert.Zero(t, SafeAllocationCapacity(math.MaxInt-1, 2))
	})

	t.Run("returns zero for negative parts", func(t *testing.T) {
		assert.Zero(t, SafeAllocationCapacity(-1))
		assert.Zero(t, SafeAllocationCapacity(2, -1))
	})
}
