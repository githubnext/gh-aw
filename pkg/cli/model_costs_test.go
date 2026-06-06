//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindModelPricing(t *testing.T) {
	pricing, ok := findModelPricing("anthropic", "claude-sonnet-4.6")
	require.True(t, ok)
	assert.InDelta(t, 0.000003, pricing["input"], 1e-12)
}

func TestComputeModelInferenceAIC(t *testing.T) {
	aic := computeModelInferenceAIC("anthropic", "claude-sonnet-4.6", 1000, 200, 400, 50, 25)
	assert.InDelta(t, 0.54825, aic, 1e-9)
}
