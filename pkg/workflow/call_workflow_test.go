//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseCallWorkflowConfig_ArrayFormat tests parsing call-workflow with array format
func TestParseCallWorkflowConfig_ArrayFormat(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"call-workflow": []any{"worker-a", "worker-b", "worker-c"},
	}

	config := compiler.parseCallWorkflowConfig(outputMap)
	require.NotNil(t, config, "Should return a config")
	assert.Equal(t, []string{"worker-a", "worker-b", "worker-c"}, config.Workflows, "Should parse all workflows")
	assert.Equal(t, "1", *config.Max, "Default max should be 1")
}

// TestParseCallWorkflowConfig_MapFormat tests parsing call-workflow with map format
func TestParseCallWorkflowConfig_MapFormat(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"call-workflow": map[string]any{
			"workflows": []any{"spring-boot-bugfix", "frontend-dep-upgrade"},
			"max":       3,
		},
	}

	config := compiler.parseCallWorkflowConfig(outputMap)
	require.NotNil(t, config, "Should return a config")
	assert.Equal(t, []string{"spring-boot-bugfix", "frontend-dep-upgrade"}, config.Workflows, "Should parse workflows")
	assert.Equal(t, "3", *config.Max, "Should parse max value")
}

// TestParseCallWorkflowConfig_DefaultMax tests that max defaults to 1
func TestParseCallWorkflowConfig_DefaultMax(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"call-workflow": map[string]any{
			"workflows": []any{"worker-a"},
		},
	}

	config := compiler.parseCallWorkflowConfig(outputMap)
	require.NotNil(t, config, "Should return a config")
	assert.Equal(t, "1", *config.Max, "Default max should be 1")
}

// TestParseCallWorkflowConfig_MaxCap tests that max is capped at 50
func TestParseCallWorkflowConfig_MaxCap(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"call-workflow": map[string]any{
			"workflows": []any{"worker-a"},
			"max":       100,
		},
	}

	config := compiler.parseCallWorkflowConfig(outputMap)
	require.NotNil(t, config, "Should return a config")
	assert.Equal(t, "50", *config.Max, "Max should be capped at 50")
}

// TestParseCallWorkflowConfig_NotPresent tests that nil is returned when call-workflow is not configured
func TestParseCallWorkflowConfig_NotPresent(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{}

	config := compiler.parseCallWorkflowConfig(outputMap)
	assert.Nil(t, config, "Should return nil when call-workflow is not configured")
}

// TestContainsWorkflowCall_StringForm tests containsWorkflowCall with string form
func TestContainsWorkflowCall_StringForm(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "workflow_call string",
			input:    "workflow_call",
			expected: true,
		},
		{
			name:     "push string",
			input:    "push",
			expected: false,
		},
		{
			name:     "array with workflow_call",
			input:    []any{"push", "workflow_call"},
			expected: true,
		},
		{
			name:     "array without workflow_call",
			input:    []any{"push", "pull_request"},
			expected: false,
		},
		{
			name:     "map with workflow_call",
			input:    map[string]any{"workflow_call": map[string]any{"inputs": nil}},
			expected: true,
		},
		{
			name:     "map without workflow_call",
			input:    map[string]any{"push": nil},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsWorkflowCall(tt.input)
			assert.Equal(t, tt.expected, result, "containsWorkflowCall result should match expected")
		})
	}
}
