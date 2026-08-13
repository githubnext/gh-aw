package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMemoryValidationTimeoutMinutesRejectsFractionalValues(t *testing.T) {
	_, err := parseMemoryValidationTimeoutMinutes(1.9, "tools.cache-memory.validation.timeout-minutes")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be an integer number of minutes")
}

func TestParseMemoryValidationTimeoutMinutesRejectsOutOfRangeFloat(t *testing.T) {
	_, err := parseMemoryValidationTimeoutMinutes(6.0, "tools.cache-memory.validation.timeout-minutes")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 1 and 5 minutes")
}

func TestParseMemoryValidationConfigUsesTimeoutMinutes(t *testing.T) {
	config, err := parseMemoryValidationConfig(map[string]any{
		"validation": map[string]any{
			"script":          "console.log('validate')",
			"timeout-minutes": 2,
		},
	}, "tools.cache-memory.validation")

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, 2, config.TimeoutMinutes)
	assert.Equal(t, 120, memoryValidationTimeoutSeconds(config))
}

func TestParseMemoryValidationConfigRejectsTimeout(t *testing.T) {
	_, err := parseMemoryValidationConfig(map[string]any{
		"validation": map[string]any{
			"script":  "console.log('validate')",
			"timeout": 2,
		},
	}, "tools.cache-memory.validation")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "has been renamed to tools.cache-memory.validation.timeout-minutes")
}
