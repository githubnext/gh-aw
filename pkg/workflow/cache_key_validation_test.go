//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheKeyRunIDValidationObject tests that github.run_id in a cache key raises
// a compilation error (object notation).
func TestCacheKeyRunIDValidationObject(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
		errorText string
	}{
		{
			name:      "key without run_id is accepted",
			key:       "trending-data-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}",
			wantError: false,
		},
		{
			name:      "key with run_id is rejected",
			key:       "trending-data-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
			wantError: true,
			errorText: "cache key must not reference github.run_id",
		},
		{
			name:      "key with only run_id is rejected",
			key:       "${{ github.run_id }}",
			wantError: true,
			errorText: "cache key must not reference github.run_id",
		},
		{
			name:      "plain key without expressions is accepted",
			key:       "my-stable-cache-key",
			wantError: false,
		},
		{
			name:      "key with github.run_identifier is not a false positive",
			key:       "my-cache-${{ github.run_identifier }}",
			wantError: false,
		},
		{
			name:      "key with github.run_id_backup suffix is not a false positive",
			key:       "my-cache-github.run_id_backup",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"cache-memory": map[string]any{
					"key": tt.key,
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Should parse tools config")

			compiler := NewCompiler()
			_, err = compiler.extractCacheMemoryConfig(toolsConfig)

			if tt.wantError {
				require.Error(t, err, "Should return error for key containing run_id")
				assert.ErrorContains(t, err, tt.errorText, "Error should contain expected message")
			} else {
				assert.NoError(t, err, "Should not return error for valid key")
			}
		})
	}
}

// TestCacheKeyRunIDValidationArray tests that github.run_id in a cache key raises
// a compilation error (array notation).
func TestCacheKeyRunIDValidationArray(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
		errorText string
	}{
		{
			name:      "array entry key without run_id is accepted",
			key:       "my-data-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}",
			wantError: false,
		},
		{
			name:      "array entry key with run_id is rejected",
			key:       "my-data-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
			wantError: true,
			errorText: "cache key must not reference github.run_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"cache-memory": []any{
					map[string]any{
						"id":  "test-cache",
						"key": tt.key,
					},
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Should parse tools config")

			compiler := NewCompiler()
			_, err = compiler.extractCacheMemoryConfig(toolsConfig)

			if tt.wantError {
				require.Error(t, err, "Should return error for key containing run_id")
				assert.ErrorContains(t, err, tt.errorText, "Error should contain expected message")
			} else {
				assert.NoError(t, err, "Should not return error for valid key")
			}
		})
	}
}
