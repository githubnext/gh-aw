//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEnvSecrets(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		strictMode  bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "no env section is allowed",
			frontmatter: map[string]any{
				"on": "push",
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "env with no secrets is allowed",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"NODE_ENV":    "production",
					"API_URL":     "https://api.example.com",
					"ENABLE_LOGS": "true",
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "env with plain values in non-strict mode is allowed",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"NODE_ENV": "production",
					"API_URL":  "https://api.example.com",
				},
			},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "env with single secret in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container. Found: ${{ secrets.API_KEY }}",
		},
		{
			name: "env with multiple secrets in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"API_KEY":    "${{ secrets.API_KEY }}",
					"DB_PASS":    "${{ secrets.DATABASE_PASSWORD }}",
					"AUTH_TOKEN": "${{ secrets.AUTH_TOKEN }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
		{
			name: "env with secret with fallback in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY || 'default' }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
		{
			name: "env with secret embedded in string in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"AUTH_HEADER": "Bearer ${{ secrets.TOKEN }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container. Found: ${{ secrets.TOKEN }}",
		},
		{
			name: "env with mixed secrets and plain values in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"NODE_ENV": "production",
					"API_KEY":  "${{ secrets.API_KEY }}",
					"API_URL":  "https://api.example.com",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container. Found: ${{ secrets.API_KEY }}",
		},
		{
			name: "env with env variables (not secrets) is allowed",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"NODE_ENV":     "production",
					"WORKSPACE":    "${{ github.workspace }}",
					"RUNNER_TEMP":  "${{ runner.temp }}",
					"GITHUB_TOKEN": "${{ github.token }}",
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "empty env section is allowed",
			frontmatter: map[string]any{
				"on":  "push",
				"env": map[string]any{},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "env section with non-string values is allowed (edge case)",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"PORT":         8080,
					"ENABLE_DEBUG": true,
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "env with sub-expression: github.workflow && secrets.TOKEN fails in strict mode",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"TOKEN": "${{ github.workflow && secrets.TOKEN }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
		{
			name: "env with sub-expression: secrets in OR with env fails in strict mode",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"DB_PASS": "${{ secrets.DB_PASSWORD || env.DEFAULT_PASS }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
		{
			name: "env with sub-expression: secrets in parentheses fails in strict mode",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"AUTH": "${{ (github.actor || secrets.HIDDEN_KEY) }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
		{
			name: "env with sub-expression: NOT operator with secrets fails in strict mode",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"CHECK": "${{ !secrets.PRIVATE_KEY && github.workflow }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
		{
			name: "env with multiple sub-expressions fails in strict mode",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"SIMPLE":    "${{ secrets.SIMPLE_KEY }}",
					"SUB_EXPR1": "${{ github.workflow && secrets.TOKEN }}",
					"SUB_EXPR2": "${{ (github.actor || secrets.HIDDEN) }}",
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			err := compiler.validateEnvSecrets(tt.frontmatter)

			if tt.expectError {
				require.Error(t, err, "Expected an error but got none")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateEnvSecretsNonStrictMode(t *testing.T) {
	// Note: This test cannot capture the warning output directly as it goes to stderr
	// The warning behavior is tested by ensuring no error is returned in non-strict mode
	tests := []struct {
		name        string
		frontmatter map[string]any
	}{
		{
			name: "env with single secret in non-strict mode emits warning",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY }}",
				},
			},
		},
		{
			name: "env with multiple secrets in non-strict mode emits warning",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY }}",
					"DB_PASS": "${{ secrets.DATABASE_PASSWORD }}",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = false

			// In non-strict mode, this should not return an error (only emit a warning)
			err := compiler.validateEnvSecrets(tt.frontmatter)
			assert.NoError(t, err, "Non-strict mode should not return an error, only emit a warning")
		})
	}
}

func TestValidateEnvSecretsIntegration(t *testing.T) {
	// Test that validateEnvSecrets is properly integrated with the compiler
	tests := []struct {
		name        string
		frontmatter map[string]any
		strictMode  bool
		expectError bool
	}{
		{
			name: "compilation with env secrets in strict mode should fail",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "copilot",
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY }}",
				},
			},
			strictMode:  true,
			expectError: true,
		},
		{
			name: "compilation with env secrets in non-strict mode should succeed with warning",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "copilot",
				"strict": false,
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY }}",
				},
			},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "compilation without env secrets in strict mode should succeed",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "copilot",
				"env": map[string]any{
					"NODE_ENV": "production",
				},
			},
			strictMode:  true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			err := compiler.validateEnvSecrets(tt.frontmatter)

			if tt.expectError {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateEnvSecretsErrorMessage(t *testing.T) {
	compiler := NewCompiler()
	compiler.strictMode = true

	frontmatter := map[string]any{
		"on": "push",
		"env": map[string]any{
			"API_KEY": "${{ secrets.API_KEY }}",
		},
	}

	err := compiler.validateEnvSecrets(frontmatter)
	require.Error(t, err, "Expected an error")

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "strict mode", "Error should mention strict mode")
	assert.Contains(t, errorMsg, "secrets detected in 'env' section", "Error should explain the issue")
	assert.Contains(t, errorMsg, "leaked to the agent container", "Error should explain the security risk")
	assert.Contains(t, errorMsg, "${{ secrets.API_KEY }}", "Error should list the specific secret found")
	assert.Contains(t, errorMsg, "Use engine-specific secret configuration", "Error should provide guidance")
}

func TestExtractSecretsFromEnv(t *testing.T) {
	// Test the underlying secret extraction logic
	tests := []struct {
		name           string
		envMap         map[string]string
		expectedCount  int
		expectedSecret string
	}{
		{
			name: "single secret",
			envMap: map[string]string{
				"API_KEY": "${{ secrets.API_KEY }}",
			},
			expectedCount:  1,
			expectedSecret: "${{ secrets.API_KEY }}",
		},
		{
			name: "multiple secrets",
			envMap: map[string]string{
				"API_KEY": "${{ secrets.API_KEY }}",
				"DB_PASS": "${{ secrets.DB_PASSWORD }}",
			},
			expectedCount: 2,
		},
		{
			name: "secret with fallback",
			envMap: map[string]string{
				"API_KEY": "${{ secrets.API_KEY || 'default' }}",
			},
			expectedCount: 1,
		},
		{
			name: "embedded secret",
			envMap: map[string]string{
				"AUTH_HEADER": "Bearer ${{ secrets.TOKEN }}",
			},
			expectedCount:  1,
			expectedSecret: "${{ secrets.TOKEN }}",
		},
		{
			name: "no secrets",
			envMap: map[string]string{
				"NODE_ENV": "production",
				"API_URL":  "https://api.example.com",
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets := ExtractSecretsFromMap(tt.envMap)
			assert.Len(t, secrets, tt.expectedCount, "Secret count mismatch")

			if tt.expectedSecret != "" {
				found := false
				for _, secret := range secrets {
					if secret == tt.expectedSecret {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected secret %s not found in extracted secrets", tt.expectedSecret)
			}
		})
	}
}

func TestValidateEnvSecretsMultipleSecretsErrorMessage(t *testing.T) {
	compiler := NewCompiler()
	compiler.strictMode = true

	frontmatter := map[string]any{
		"on": "push",
		"env": map[string]any{
			"API_KEY":    "${{ secrets.API_KEY }}",
			"DB_PASS":    "${{ secrets.DB_PASSWORD }}",
			"AUTH_TOKEN": "${{ secrets.AUTH_TOKEN }}",
		},
	}

	err := compiler.validateEnvSecrets(frontmatter)
	require.Error(t, err, "Expected an error")

	errorMsg := err.Error()
	// Should contain all three secrets or at least indicate multiple secrets
	secretCount := strings.Count(errorMsg, "${{ secrets.")
	assert.GreaterOrEqual(t, secretCount, 1, "Error should mention at least one secret")
}

func TestValidateEngineEnvSecrets(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		strictMode  bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "engine string format has no env section to validate",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "copilot",
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "engine object without env is allowed",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "engine.env with no secrets is allowed",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "copilot",
					"env": map[string]any{
						"NODE_ENV": "production",
						"API_URL":  "https://api.example.com",
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "engine.env with single secret in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "copilot",
					"env": map[string]any{
						"API_KEY": "${{ secrets.API_KEY }}",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'engine.env' section will be leaked to the agent container. Found: ${{ secrets.API_KEY }}",
		},
		{
			name: "engine.env with multiple secrets in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "copilot",
					"env": map[string]any{
						"API_KEY": "${{ secrets.API_KEY }}",
						"DB_PASS": "${{ secrets.DATABASE_PASSWORD }}",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'engine.env' section will be leaked to the agent container",
		},
		{
			name: "engine.env with secret embedded in string in strict mode fails",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "copilot",
					"env": map[string]any{
						"AUTH_HEADER": "Bearer ${{ secrets.TOKEN }}",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'engine.env' section will be leaked to the agent container. Found: ${{ secrets.TOKEN }}",
		},
		{
			name: "engine.env with secret in non-strict mode emits warning (no error)",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "copilot",
					"env": map[string]any{
						"API_KEY": "${{ secrets.API_KEY }}",
					},
				},
			},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "both env and engine.env with secrets in strict mode fails on env first",
			frontmatter: map[string]any{
				"on": "push",
				"env": map[string]any{
					"TOP_KEY": "${{ secrets.TOP_SECRET }}",
				},
				"engine": map[string]any{
					"id": "copilot",
					"env": map[string]any{
						"API_KEY": "${{ secrets.API_KEY }}",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets detected in 'env' section will be leaked to the agent container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			err := compiler.validateEnvSecrets(tt.frontmatter)

			if tt.expectError {
				require.Error(t, err, "Expected an error but got none")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}
