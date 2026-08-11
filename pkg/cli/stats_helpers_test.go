//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafePercent(t *testing.T) {
	assert.InDelta(t, 0.0, safePercent(1, 0), 1e-12, "zero total")
	assert.InDelta(t, 0.0, safePercent(0, 4), 1e-12, "zero part")
	assert.InDelta(t, 25.0, safePercent(1, 4), 1e-12, "percentage")
}
