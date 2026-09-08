//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestExtractEnvExpressionsFromValuePrefersFallbackForDuplicateVariable(t *testing.T) {
	result := ExtractEnvExpressionsFromValue("${{ env.HOST || 'localhost' }}/${{ env.HOST }}")

	assert.Equal(t, map[string]string{
		"HOST": "${{ env.HOST || 'localhost' }}",
	}, result)
}

func TestExtractEnvExpressionsFromMapPrefersFallbackForDuplicateVariable(t *testing.T) {
	result := ExtractEnvExpressionsFromMap(map[string]string{
		"HOST_WITH_FALLBACK": "${{ env.HOST || 'localhost' }}",
		"HOST":               "${{ env.HOST }}",
	})

	assert.Equal(t, map[string]string{
		"HOST": "${{ env.HOST || 'localhost' }}",
	}, result)
}

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

func TestExtractGitHubContextExpressionsFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:     "simple github.workflow",
			value:    `"branch":"assets/${{ github.workflow }}"`,
			expected: map[string]string{"GITHUB_WORKFLOW": "${{ github.workflow }}"},
		},
		{
			name:     "github.ref_name",
			value:    `"base-branch":"${{ github.ref_name }}"`,
			expected: map[string]string{"GITHUB_REF_NAME": "${{ github.ref_name }}"},
		},
		{
			name:     "github.run_id",
			value:    `"key":"cache-${{ github.run_id }}"`,
			expected: map[string]string{"GITHUB_RUN_ID": "${{ github.run_id }}"},
		},
		{
			name:     "multiple expressions",
			value:    `"branch":"${{ github.workflow }}/run-${{ github.run_id }}"`,
			expected: map[string]string{"GITHUB_WORKFLOW": "${{ github.workflow }}", "GITHUB_RUN_ID": "${{ github.run_id }}"},
		},
		{
			name:     "no expressions",
			value:    `"branch":"assets/my-workflow"`,
			expected: map[string]string{},
		},
		{
			name:     "secrets are not extracted",
			value:    `"token":"${{ secrets.MY_TOKEN }}"`,
			expected: map[string]string{},
		},
		{
			name:     "complex event payload not extracted",
			value:    `"title":"${{ github.event.issue.title }}"`,
			expected: map[string]string{},
		},
		{
			name:     "expression with spaces",
			value:    `"branch":"assets/${{  github.workflow  }}"`,
			expected: map[string]string{"GITHUB_WORKFLOW": "${{  github.workflow  }}"},
		},
		{
			name:     "github.actor",
			value:    `"actor":"${{ github.actor }}"`,
			expected: map[string]string{"GITHUB_ACTOR": "${{ github.actor }}"},
		},
		{
			name:     "github.repository",
			value:    `"repo":"${{ github.repository }}"`,
			expected: map[string]string{"GITHUB_REPOSITORY": "${{ github.repository }}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractGitHubContextExpressionsFromValue(tt.value)

			assert.Len(t, result, len(tt.expected), "Should extract expected number of GitHub context expressions")

			for varName, expr := range tt.expected {
				assert.Equal(t, expr, result[varName], "Env var %q should map to the correct expression", varName)
			}
		})
	}
}

func TestExtractWorkflowInputExpressionsFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:  "single input expression",
			value: `"repo":"${{ inputs.target_repo }}"`,
			expected: map[string]string{
				"GH_AW_INPUT_TARGET_REPO": "${{ inputs.target_repo }}",
			},
		},
		{
			name:  "multiple input expressions with bracket dash and underscore",
			value: `"repo":"${{ inputs['target-repo'] }}","base":"${{ inputs.base_branch }}"`,
			expected: map[string]string{
				"GH_AW_INPUT_TARGET_REPO": "${{ inputs['target-repo'] }}",
				"GH_AW_INPUT_BASE_BRANCH": "${{ inputs.base_branch }}",
			},
		},
		{
			name:  "dot notation with dash remains supported",
			value: `"repo":"${{ inputs.target-repo }}"`,
			expected: map[string]string{
				"GH_AW_INPUT_TARGET_REPO": "${{ inputs.target-repo }}",
			},
		},
		{
			name:  "double-quote bracket notation",
			value: `"base":"${{ inputs["base-branch"] }}"`,
			expected: map[string]string{
				"GH_AW_INPUT_BASE_BRANCH": `${{ inputs["base-branch"] }}`,
			},
		},
		{
			name:     "no input expressions",
			value:    `"repo":"${{ github.repository }}"`,
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractWorkflowInputExpressionsFromValue(tt.value)
			assert.Equal(t, tt.expected, result, "Should extract expected workflow input expressions")
		})
	}
}

func TestFormatInputNameAsEnvVar(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		expected  string
	}{
		{name: "underscore", inputName: "target_repo", expected: "GH_AW_INPUT_TARGET_REPO"},
		{name: "dash", inputName: "base-branch", expected: "GH_AW_INPUT_BASE_BRANCH"},
		{name: "consecutive separators", inputName: "my--input__name", expected: "GH_AW_INPUT_MY__INPUT__NAME"},
		{name: "mixed case and numeric", inputName: "Repo2Name", expected: "GH_AW_INPUT_REPO2NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatInputNameAsEnvVar(tt.inputName)
			assert.Equal(t, tt.expected, actual, "Input name should be converted to the expected env var")
		})
	}
}

func TestContainsJobOutputExpr(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "pure job output expression",
			value: "${{ needs.fetch_token.outputs.token }}",
			want:  true,
		},
		{
			name:  "job output with extra whitespace",
			value: "${{  needs.fetch_token.outputs.token  }}",
			want:  true,
		},
		{
			name:  "job output embedded in a larger string",
			value: "Bearer ${{ needs.auth_job.outputs.access_token }}",
			want:  true,
		},
		{
			name:  "secret expression is not a job output",
			value: "${{ secrets.GH_TOKEN }}",
			want:  false,
		},
		{
			name:  "static value is not a job output",
			value: "https://api.example.com",
			want:  false,
		},
		{
			name:  "needs.result (no .outputs.) is not a job output",
			value: "${{ needs.some_job.result }}",
			want:  false,
		},
		{
			name:  "github token expression is not a job output",
			value: "${{ github.token }}",
			want:  false,
		},
		{
			name:  "job output with hyphenated job name",
			value: "${{ needs.fetch-token.outputs.token }}",
			want:  true,
		},
		{
			name:  "needs not first token in expression (sub-expression with &&)",
			value: "${{ github.ref && needs.auth.outputs.token }}",
			want:  true,
		},
		{
			name:  "needs not first token in expression (sub-expression with ||)",
			value: "${{ env.FALLBACK || needs.mint.outputs.gh_token }}",
			want:  true,
		},
		{
			name:  "bracket notation with single quotes",
			value: "${{ needs['auth'].outputs['token'] }}",
			want:  true,
		},
		{
			name:  "bracket notation with double quotes",
			value: `${{ needs["auth"].outputs["token"] }}`,
			want:  true,
		},
		{
			name:  "full bracket notation (job and outputs key both in brackets)",
			value: "${{ needs['auth']['outputs']['token'] }}",
			want:  true,
		},
		{
			name:  "bracket notation embedded after other token",
			value: "${{ github.token || needs['mint'].outputs['gh_token'] }}",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsJobOutputExpr(tt.value)
			assert.Equal(t, tt.want, got, "ContainsJobOutputExpr(%q)", tt.value)
		})
	}
}
