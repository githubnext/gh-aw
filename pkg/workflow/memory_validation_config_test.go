package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMemoryValidationTimeoutRejectsFractionalValues(t *testing.T) {
	_, err := parseMemoryValidationTimeout(1.9, "tools.cache-memory.validation.timeout")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be an integer number of seconds")
}

func TestParseMemoryValidationTimeoutRejectsOutOfRangeFloat(t *testing.T) {
	_, err := parseMemoryValidationTimeout(301.0, "tools.cache-memory.validation.timeout")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 1 and 300 seconds")
}
