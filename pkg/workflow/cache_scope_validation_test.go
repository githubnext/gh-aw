//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheMemoryScopeValidationObject tests scope validation with object notation
func TestCacheMemoryScopeValidationObject(t *testing.T) {
	tests := []struct {
		name      string
		scope     string
		wantError bool
		errorText string
	}{
		{
			name:      "valid workflow scope",
			scope:     "workflow",
			wantError: false,
		},
		{
			name:      "valid repo scope",
			scope:     "repo",
			wantError: false,
		},
		{
			name:      "invalid organization scope",
			scope:     "organization",
			wantError: true,
			errorText: `invalid cache-memory scope "organization": must be one of [workflow repo]`,
		},
		{
			name:      "invalid global scope",
			scope:     "global",
			wantError: true,
			errorText: `invalid cache-memory scope "global": must be one of [workflow repo]`,
		},
		{
			name:      "invalid whitespace scope",
			scope:     "   ",
			wantError: true,
			errorText: `invalid cache-memory scope "   ": must be one of [workflow repo]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"cache-memory": map[string]any{
					"scope": tt.scope,
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Should parse tools config")

			compiler := NewCompiler()
			_, err = compiler.extractCacheMemoryConfig(toolsConfig)

			if tt.wantError {
				require.Error(t, err, "Should return error for invalid scope")
				assert.ErrorContains(t, err, tt.errorText, "Error should contain expected message")
			} else {
				assert.NoError(t, err, "Should not return error for valid scope")
			}
		})
	}
}

// TestCacheMemoryScopeValidationArray tests scope validation with array notation
func TestCacheMemoryScopeValidationArray(t *testing.T) {
	tests := []struct {
		name      string
		scope     string
		wantError bool
		errorText string
	}{
		{
			name:      "valid workflow scope in array",
			scope:     "workflow",
			wantError: false,
		},
		{
			name:      "valid repo scope in array",
			scope:     "repo",
			wantError: false,
		},
		{
			name:      "invalid scope in array",
			scope:     "global",
			wantError: true,
			errorText: `invalid cache-memory scope "global": must be one of [workflow repo]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"cache-memory": []any{
					map[string]any{
						"id":    "test-cache",
						"scope": tt.scope,
					},
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Should parse tools config")

			compiler := NewCompiler()
			_, err = compiler.extractCacheMemoryConfig(toolsConfig)

			if tt.wantError {
				require.Error(t, err, "Should return error for invalid scope")
				assert.ErrorContains(t, err, tt.errorText, "Error should contain expected message")
			} else {
				assert.NoError(t, err, "Should not return error for valid scope")
			}
		})
	}
}

// TestCacheMemoryScopeDefault tests that omitting scope defaults to "workflow"
func TestCacheMemoryScopeDefault(t *testing.T) {
	toolsMap := map[string]any{
		"cache-memory": map[string]any{
			"key": "my-cache-key",
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Should parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractCacheMemoryConfig(toolsConfig)

	require.NoError(t, err, "Should not error when scope is omitted")
	require.NotNil(t, config, "Config should not be nil")
	require.Len(t, config.Caches, 1, "Should have exactly one cache entry")

	assert.Equal(t, "workflow", config.Caches[0].Scope, "Default scope should be 'workflow'")
}
