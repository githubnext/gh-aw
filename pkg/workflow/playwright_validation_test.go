//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePlaywrightMode tests the validatePlaywrightMode deprecation warning.
func TestValidatePlaywrightMode(t *testing.T) {
	tests := []struct {
		name        string
		tools       map[string]any
		strictMode  bool
		expectWarn  bool
		expectError bool
		errorSubstr string
	}{
		{
			name:       "playwright not configured",
			tools:      map[string]any{},
			expectWarn: false,
		},
		{
			name:       "playwright set to false",
			tools:      map[string]any{"playwright": false},
			expectWarn: false,
		},
		{
			name:       "playwright nil (enabled with default settings)",
			tools:      map[string]any{"playwright": nil},
			expectWarn: true,
		},
		{
			name:       "playwright enabled with empty map (MCP mode default)",
			tools:      map[string]any{"playwright": map[string]any{}},
			expectWarn: true,
		},
		{
			name:       "playwright explicit mcp mode",
			tools:      map[string]any{"playwright": map[string]any{"mode": "mcp"}},
			expectWarn: true,
		},
		{
			name:       "playwright cli mode — no warning",
			tools:      map[string]any{"playwright": map[string]any{"mode": "cli"}},
			expectWarn: false,
		},
		{
			name:       "playwright cli mode uppercase — no warning",
			tools:      map[string]any{"playwright": map[string]any{"mode": "CLI"}},
			expectWarn: false,
		},
		{
			name:        "playwright mcp mode in strict mode returns error",
			tools:       map[string]any{"playwright": map[string]any{"mode": "mcp"}},
			strictMode:  true,
			expectError: true,
			errorSubstr: "strict mode",
		},
		{
			name:        "playwright nil (default MCP) in strict mode returns error",
			tools:       map[string]any{"playwright": nil},
			strictMode:  true,
			expectError: true,
			errorSubstr: "strict mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			workflowData := &WorkflowData{
				Tools: tt.tools,
			}

			err := compiler.validatePlaywrightMode(workflowData)

			if tt.expectError {
				require.Error(t, err, "expected an error but got none")
				assert.Contains(t, err.Error(), tt.errorSubstr,
					"error %q should contain %q", err.Error(), tt.errorSubstr)
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}

// TestValidatePlaywrightModeNilWorkflow ensures no panic on nil/empty input.
func TestValidatePlaywrightModeNilWorkflow(t *testing.T) {
	compiler := NewCompiler()

	err := compiler.validatePlaywrightMode(nil)
	require.NoError(t, err, "nil workflowData should not return error")

	err = compiler.validatePlaywrightMode(&WorkflowData{Tools: nil})
	require.NoError(t, err, "nil tools should not return error")
}
