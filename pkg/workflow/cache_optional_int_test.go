//go:build !integration

package workflow

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOptionalInt(t *testing.T) {
	t.Run("integer float is accepted", func(t *testing.T) {
		value := parseOptionalInt(7.0)
		require.NotNil(t, value)
		assert.Equal(t, 7, *value)
	})

	t.Run("fractional float is rejected", func(t *testing.T) {
		assert.Nil(t, parseOptionalInt(7.5))
	})

	t.Run("nan and infinity are rejected", func(t *testing.T) {
		assert.Nil(t, parseOptionalInt(math.NaN()))
		assert.Nil(t, parseOptionalInt(math.Inf(1)))
	})

	t.Run("uint64 above max int is rejected", func(t *testing.T) {
		assert.Nil(t, parseOptionalInt(uint64(math.MaxInt)+1))
	})

	t.Run("large exact float is architecture aware", func(t *testing.T) {
		value := parseOptionalInt(1e12)
		if strconv.IntSize == 32 {
			assert.Nil(t, value)
			return
		}
		require.NotNil(t, value)
		assert.Equal(t, 1000000000000, *value)
	})
}
