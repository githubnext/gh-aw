//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDefaultAiCreditsPricing(t *testing.T) {
	t.Run("nil pricing is allowed", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{})
		require.NoError(t, err)
	})

	t.Run("positive input and output are valid", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:  3.0,
				Output: 15.0,
			},
		})
		require.NoError(t, err)
	})

	t.Run("small positive values are valid for free models", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:  0.000001,
				Output: 0.000001,
			},
		})
		require.NoError(t, err)
	})

	t.Run("zero input is rejected", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:  0,
				Output: 1.0,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input")
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("zero output is rejected", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:  1.0,
				Output: 0,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "output")
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("negative input is rejected", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:  -1.0,
				Output: 1.0,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input")
	})

	t.Run("zero cache_read is rejected when set", func(t *testing.T) {
		v := 0.0
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:       1.0,
				Output:      1.0,
				CachedInput: &v,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache_read")
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("zero cache_write is rejected when set", func(t *testing.T) {
		v := 0.0
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:      1.0,
				Output:     1.0,
				CacheWrite: &v,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache_write")
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("nil cache_read and nil cache_write are allowed", func(t *testing.T) {
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:  1.0,
				Output: 1.0,
				// CachedInput and CacheWrite are nil
			},
		})
		require.NoError(t, err)
	})

	t.Run("positive cache fields are valid", func(t *testing.T) {
		cachedInput := 0.3
		cacheWrite := 3.0
		err := validateDefaultAiCreditsPricing(&WorkflowData{
			DefaultAiCreditsPricing: &AiCreditsPricingConfig{
				Input:       3.0,
				Output:      15.0,
				CachedInput: &cachedInput,
				CacheWrite:  &cacheWrite,
			},
		})
		require.NoError(t, err)
	})
}
