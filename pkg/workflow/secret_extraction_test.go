//go:build !integration

package workflow

import (
	"testing"
)

// TestSharedExtractSecretName tests the shared ExtractSecretName utility function
func TestSharedExtractSecretName(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "simple secret",
			value:    "${{ secrets.DD_API_KEY }}",
			expected: "DD_API_KEY",
		},
		{
			name:     "secret with default value",
			value:    "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			expected: "DD_SITE",
		},
		{
			name:     "secret with spaces",
			value:    "${{  secrets.API_TOKEN  }}",
			expected: "API_TOKEN",
		},
		{
			name:     "bearer token",
			value:    "Bearer ${{ secrets.TAVILY_API_KEY }}",
			expected: "TAVILY_API_KEY",
		},
		{
			name:     "no secret",
			value:    "plain value",
			expected: "",
		},
		{
			name:     "empty value",
			value:    "",
			expected: "",
		},
		{
			name:     "secret with underscore",
			value:    "${{ secrets.MY_SECRET_KEY }}",
			expected: "MY_SECRET_KEY",
		},
		{
			name:     "secret with numbers",
			value:    "${{ secrets.API_KEY_123 }}",
			expected: "API_KEY_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSecretName(tt.value)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSharedExtractSecretsFromValue tests the shared ExtractSecretsFromValue utility function
func TestSharedExtractSecretsFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:  "simple secret",
			value: "${{ secrets.DD_API_KEY }}",
			expected: map[string]string{
				"DD_API_KEY": "${{ secrets.DD_API_KEY }}",
			},
		},
		{
			name:  "secret with default value",
			value: "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			expected: map[string]string{
				"DD_SITE": "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			},
		},
		{
			name:  "bearer token",
			value: "Bearer ${{ secrets.TAVILY_API_KEY }}",
			expected: map[string]string{
				"TAVILY_API_KEY": "${{ secrets.TAVILY_API_KEY }}",
			},
		},
		{
			name:  "multiple secrets in one value",
			value: "${{ secrets.KEY1 }} and ${{ secrets.KEY2 }}",
			expected: map[string]string{
				"KEY1": "${{ secrets.KEY1 }}",
				"KEY2": "${{ secrets.KEY2 }}",
			},
		},
		{
			name:     "no secrets",
			value:    "plain value",
			expected: map[string]string{},
		},
		{
			name:     "empty value",
			value:    "",
			expected: map[string]string{},
		},
		{
			name:  "secret with complex default",
			value: "${{ secrets.CONFIG || 'default-config-value' }}",
			expected: map[string]string{
				"CONFIG": "${{ secrets.CONFIG || 'default-config-value' }}",
			},
		},
		{
			name:  "sub-expression: github.workflow && secrets.TOKEN",
			value: "${{ github.workflow && secrets.TOKEN }}",
			expected: map[string]string{
				"TOKEN": "${{ github.workflow && secrets.TOKEN }}",
			},
		},
		{
			name:  "sub-expression: secrets in OR expression with env",
			value: "${{ secrets.DB_PASS || env.FALLBACK }}",
			expected: map[string]string{
				"DB_PASS": "${{ secrets.DB_PASS || env.FALLBACK }}",
			},
		},
		{
			name:  "sub-expression: secrets in parentheses",
			value: "${{ (github.actor || secrets.HIDDEN) }}",
			expected: map[string]string{
				"HIDDEN": "${{ (github.actor || secrets.HIDDEN) }}",
			},
		},
		{
			name:  "sub-expression: complex boolean with secrets",
			value: "${{ (github.workflow || secrets.TOKEN) && github.repository }}",
			expected: map[string]string{
				"TOKEN": "${{ (github.workflow || secrets.TOKEN) && github.repository }}",
			},
		},
		{
			name:  "sub-expression: NOT operator with secrets",
			value: "${{ !secrets.PRIVATE_KEY && github.workflow }}",
			expected: map[string]string{
				"PRIVATE_KEY": "${{ !secrets.PRIVATE_KEY && github.workflow }}",
			},
		},
		{
			name:  "sub-expression: multiple secrets in same expression",
			value: "${{ secrets.KEY1 && secrets.KEY2 }}",
			expected: map[string]string{
				"KEY1": "${{ secrets.KEY1 && secrets.KEY2 }}",
				"KEY2": "${{ secrets.KEY1 && secrets.KEY2 }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSecretsFromValue(tt.value)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d secrets, got %d", len(tt.expected), len(result))
			}

			for varName, expr := range tt.expected {
				if result[varName] != expr {
					t.Errorf("Expected secret %q to have expression %q, got %q", varName, expr, result[varName])
				}
			}
		})
	}
}

// TestSharedExtractSecretsFromMap tests the shared ExtractSecretsFromMap utility function
func TestSharedExtractSecretsFromMap(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]string
		expected map[string]string
	}{
		{
			name: "HTTP headers with secrets",
			values: map[string]string{
				"DD_API_KEY":         "${{ secrets.DD_API_KEY }}",
				"DD_APPLICATION_KEY": "${{ secrets.DD_APPLICATION_KEY }}",
				"DD_SITE":            "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			},
			expected: map[string]string{
				"DD_API_KEY":         "${{ secrets.DD_API_KEY }}",
				"DD_APPLICATION_KEY": "${{ secrets.DD_APPLICATION_KEY }}",
				"DD_SITE":            "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			},
		},
		{
			name: "env vars with secrets",
			values: map[string]string{
				"API_KEY": "${{ secrets.API_KEY }}",
				"TOKEN":   "${{ secrets.TOKEN }}",
			},
			expected: map[string]string{
				"API_KEY": "${{ secrets.API_KEY }}",
				"TOKEN":   "${{ secrets.TOKEN }}",
			},
		},
		{
			name: "mixed secrets and plain values",
			values: map[string]string{
				"Authorization": "Bearer ${{ secrets.AUTH_TOKEN }}",
				"Content-Type":  "application/json",
				"API_KEY":       "${{ secrets.API_KEY }}",
			},
			expected: map[string]string{
				"AUTH_TOKEN": "${{ secrets.AUTH_TOKEN }}",
				"API_KEY":    "${{ secrets.API_KEY }}",
			},
		},
		{
			name: "no secrets",
			values: map[string]string{
				"SIMPLE_VAR": "plain value",
			},
			expected: map[string]string{},
		},
		{
			name: "duplicate secrets (same secret in multiple values)",
			values: map[string]string{
				"Header1": "${{ secrets.API_KEY }}",
				"Header2": "${{ secrets.API_KEY }}",
			},
			expected: map[string]string{
				"API_KEY": "${{ secrets.API_KEY }}",
			},
		},
		{
			name:     "empty map",
			values:   map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSecretsFromMap(tt.values)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d secrets, got %d", len(tt.expected), len(result))
			}

			for varName, expr := range tt.expected {
				if result[varName] != expr {
					t.Errorf("Expected secret %q to have expression %q, got %q", varName, expr, result[varName])
				}
			}
		})
	}
}

// TestSharedReplaceSecretsWithEnvVars tests the shared ReplaceSecretsWithEnvVars utility function
func TestSharedReplaceSecretsWithEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		secrets  map[string]string
		expected string
	}{
		{
			name:  "simple replacement",
			value: "${{ secrets.DD_API_KEY }}",
			secrets: map[string]string{
				"DD_API_KEY": "${{ secrets.DD_API_KEY }}",
			},
			expected: "\\${DD_API_KEY}",
		},
		{
			name:  "replacement with default value",
			value: "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			secrets: map[string]string{
				"DD_SITE": "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			},
			expected: "\\${DD_SITE}",
		},
		{
			name:  "bearer token replacement",
			value: "Bearer ${{ secrets.TAVILY_API_KEY }}",
			secrets: map[string]string{
				"TAVILY_API_KEY": "${{ secrets.TAVILY_API_KEY }}",
			},
			expected: "Bearer \\${TAVILY_API_KEY}",
		},
		{
			name:  "multiple replacements",
			value: "${{ secrets.KEY1 }} and ${{ secrets.KEY2 }}",
			secrets: map[string]string{
				"KEY1": "${{ secrets.KEY1 }}",
				"KEY2": "${{ secrets.KEY2 }}",
			},
			expected: "\\${KEY1} and \\${KEY2}",
		},
		{
			name:     "no replacements",
			value:    "plain value",
			secrets:  map[string]string{},
			expected: "plain value",
		},
		{
			name:  "partial replacement",
			value: "${{ secrets.API_KEY }} and plain text",
			secrets: map[string]string{
				"API_KEY": "${{ secrets.API_KEY }}",
			},
			expected: "\\${API_KEY} and plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceSecretsWithEnvVars(tt.value, tt.secrets)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSharedExtractSecretsFromValueEdgeCases tests edge cases for the shared ExtractSecretsFromValue utility function
func TestSharedExtractSecretsFromValueEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:     "malformed expression - missing closing braces",
			value:    "${{ secrets.KEY",
			expected: map[string]string{},
		},
		{
			name:     "malformed expression - missing opening braces",
			value:    "secrets.KEY }}",
			expected: map[string]string{},
		},
		{
			name:     "incomplete expression",
			value:    "${{ secrets.",
			expected: map[string]string{},
		},
		{
			name:  "secret name with trailing space before pipe",
			value: "${{ secrets.KEY  || 'default' }}",
			expected: map[string]string{
				"KEY": "${{ secrets.KEY  || 'default' }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSecretsFromValue(tt.value)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d secrets, got %d", len(tt.expected), len(result))
			}

			for varName, expr := range tt.expected {
				if result[varName] != expr {
					t.Errorf("Expected secret %q to have expression %q, got %q", varName, expr, result[varName])
				}
			}
		})
	}
}

// TestExpressionToEnvVarName tests the expressionToEnvVarName utility function
func TestExpressionToEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "secret expression",
			expr:     "${{ secrets.DD_API_KEY }}",
			expected: "DD_API_KEY",
		},
		{
			name:     "secret with default",
			expr:     "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			expected: "DD_SITE",
		},
		{
			name:     "github context",
			expr:     "${{ github.workflow }}",
			expected: "GH_AW_GITHUB_WORKFLOW",
		},
		{
			name:     "github nested context",
			expr:     "${{ github.event.repository.default_branch }}",
			expected: "GH_AW_GITHUB_EVENT_REPOSITORY_DEFAULT_BRANCH",
		},
		{
			name:     "github ref_name",
			expr:     "${{ github.ref_name }}",
			expected: "GH_AW_GITHUB_REF_NAME",
		},
		{
			name:     "steps output",
			expr:     "${{ steps.parse-guard-vars.outputs.blocked_users }}",
			expected: "GH_AW_STEP_PARSE_GUARD_VARS_BLOCKED_USERS",
		},
		{
			name:     "steps simple output",
			expr:     "${{ steps.build.outputs.result }}",
			expected: "GH_AW_STEP_BUILD_RESULT",
		},
		{
			name:     "vars context",
			expr:     "${{ vars.MY_VAR }}",
			expected: "GH_AW_VARS_MY_VAR",
		},
		{
			name:     "inputs context",
			expr:     "${{ inputs.my_input }}",
			expected: "GH_AW_INPUTS_MY_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expressionToEnvVarName(tt.expr)
			if result != tt.expected {
				t.Errorf("expressionToEnvVarName(%q) = %q, want %q", tt.expr, result, tt.expected)
			}
		})
	}
}

// TestExtractAllExpressionsFromValue tests the ExtractAllExpressionsFromValue utility function
func TestExtractAllExpressionsFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:     "empty value",
			value:    "",
			expected: map[string]string{},
		},
		{
			name:     "no expressions",
			value:    `{"branch":"assets/my-workflow","max-size":10240}`,
			expected: map[string]string{},
		},
		{
			name:  "single github expression",
			value: `{"branch":"assets/${{ github.workflow }}","max-size":10240}`,
			expected: map[string]string{
				"GH_AW_GITHUB_WORKFLOW": "${{ github.workflow }}",
			},
		},
		{
			name:  "single secret expression",
			value: `{"token":"${{ secrets.MY_TOKEN }}"}`,
			expected: map[string]string{
				"MY_TOKEN": "${{ secrets.MY_TOKEN }}",
			},
		},
		{
			name:  "multiple mixed expressions",
			value: `{"branch":"${{ github.ref_name }}","token":"${{ secrets.GH_TOKEN }}"}`,
			expected: map[string]string{
				"GH_AW_GITHUB_REF_NAME": "${{ github.ref_name }}",
				"GH_TOKEN":              "${{ secrets.GH_TOKEN }}",
			},
		},
		{
			name:  "step output expression",
			value: `{"data":"${{ steps.fetch.outputs.result }}"}`,
			expected: map[string]string{
				"GH_AW_STEP_FETCH_RESULT": "${{ steps.fetch.outputs.result }}",
			},
		},
		{
			name:  "duplicate expressions return single entry",
			value: `{"a":"${{ github.workflow }}","b":"${{ github.workflow }}"}`,
			expected: map[string]string{
				"GH_AW_GITHUB_WORKFLOW": "${{ github.workflow }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAllExpressionsFromValue(tt.value)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d entries, got %d: %v", len(tt.expected), len(result), result)
			}

			for varName, expr := range tt.expected {
				if result[varName] != expr {
					t.Errorf("Expected env var %q -> %q, got %q", varName, expr, result[varName])
				}
			}
		})
	}
}

// TestSanitizeEnvVarName tests the sanitizeEnvVarName utility function
func TestSanitizeEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "lowercase", input: "foo", expected: "FOO"},
		{name: "dots replaced", input: "a.b.c", expected: "A_B_C"},
		{name: "hyphens replaced", input: "my-var", expected: "MY_VAR"},
		{name: "underscores kept", input: "MY_VAR", expected: "MY_VAR"},
		{name: "mixed", input: "parse-guard-vars.outputs.blocked_users", expected: "PARSE_GUARD_VARS_OUTPUTS_BLOCKED_USERS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeEnvVarName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeEnvVarName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEscapeNonPlaceholderDollars tests the escapeNonPlaceholderDollars helper
func TestEscapeNonPlaceholderDollars(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		allowedVars []string
		expected    string
	}{
		{
			name:        "no dollar signs",
			input:       `{"key":"value"}`,
			allowedVars: []string{"FOO"},
			expected:    `{"key":"value"}`,
		},
		{
			name:        "allowed placeholder preserved",
			input:       `{"branch":"assets/${GH_AW_GITHUB_WORKFLOW}"}`,
			allowedVars: []string{"GH_AW_GITHUB_WORKFLOW"},
			expected:    `{"branch":"assets/${GH_AW_GITHUB_WORKFLOW}"}`,
		},
		{
			name:        "stray dollar escaped",
			input:       `{"title":"Price: $100","branch":"assets/${GH_AW_GITHUB_WORKFLOW}"}`,
			allowedVars: []string{"GH_AW_GITHUB_WORKFLOW"},
			expected:    `{"title":"Price: \$100","branch":"assets/${GH_AW_GITHUB_WORKFLOW}"}`,
		},
		{
			name:        "multiple stray dollars escaped",
			input:       `$HOME and $PATH but ${ALLOWED}`,
			allowedVars: []string{"ALLOWED"},
			expected:    `\$HOME and \$PATH but ${ALLOWED}`,
		},
		{
			name:        "no allowed vars escapes all dollars",
			input:       `$FOO ${BAR}`,
			allowedVars: []string{},
			expected:    `\$FOO \${BAR}`,
		},
		{
			name:        "multiple allowed vars",
			input:       `${A} $stray ${B}`,
			allowedVars: []string{"A", "B"},
			expected:    `${A} \$stray ${B}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeNonPlaceholderDollars(tt.input, tt.allowedVars)
			if result != tt.expected {
				t.Errorf("escapeNonPlaceholderDollars(%q, %v) = %q, want %q", tt.input, tt.allowedVars, result, tt.expected)
			}
		})
	}
}
