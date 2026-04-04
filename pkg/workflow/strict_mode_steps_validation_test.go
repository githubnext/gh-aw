//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStepsSecrets(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		strictMode  bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "no steps section is allowed",
			frontmatter: map[string]any{
				"on": "push",
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "steps without secrets is allowed",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Setup",
						"run":  "echo hello",
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "steps with GITHUB_TOKEN are allowed (built-in token is exempt)",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Use GH CLI",
						"env": map[string]any{
							"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
						},
						"run": "gh issue list",
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "post-steps without secrets is allowed",
			frontmatter: map[string]any{
				"post-steps": []any{
					map[string]any{
						"name": "Cleanup",
						"run":  "echo done",
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "steps with secret in run field in strict mode fails",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Use secret",
						"run":  "curl -H 'Authorization: ${{ secrets.API_TOKEN }}' https://example.com",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
		{
			name: "steps with secret in env field in strict mode fails",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Use secret",
						"run":  "echo hi",
						"env": map[string]any{
							"API_KEY": "${{ secrets.API_KEY }}",
						},
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
		{
			name: "steps with secret in with field in strict mode fails",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"uses": "some/action@v1",
						"with": map[string]any{
							"token": "${{ secrets.MY_API_TOKEN }}",
						},
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
		{
			name: "post-steps with secret in strict mode fails",
			frontmatter: map[string]any{
				"post-steps": []any{
					map[string]any{
						"name": "Notify",
						"run":  "echo ${{ secrets.SLACK_TOKEN }}",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'post-steps' section",
		},
		{
			name: "steps with secret in non-strict mode emits warning but no error",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Use secret",
						"run":  "echo ${{ secrets.API_KEY }}",
					},
				},
			},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "post-steps with secret in non-strict mode emits warning but no error",
			frontmatter: map[string]any{
				"post-steps": []any{
					map[string]any{
						"name": "Notify",
						"run":  "echo ${{ secrets.SLACK_TOKEN }}",
					},
				},
			},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "steps section that is not a list is skipped",
			frontmatter: map[string]any{
				"steps": "not-a-list",
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "multiple secrets in steps are all reported",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Step 1",
						"env": map[string]any{
							"KEY1": "${{ secrets.KEY1 }}",
							"KEY2": "${{ secrets.KEY2 }}",
						},
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			err := compiler.validateStepsSecrets(tt.frontmatter)

			if tt.expectError {
				require.Error(t, err, "expected an error but got none")
				assert.Contains(t, err.Error(), tt.errorMsg,
					"error %q should contain %q", err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}

func TestExtractSecretsFromStepValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "nil value returns empty",
			input:    nil,
			expected: nil,
		},
		{
			name:     "plain string without secrets returns empty",
			input:    "echo hello",
			expected: nil,
		},
		{
			name:     "string with secret expression returns it",
			input:    "${{ secrets.TOKEN }}",
			expected: []string{"${{ secrets.TOKEN }}"},
		},
		{
			name:     "string with secret in larger expression returns it",
			input:    "curl -H 'Authorization: ${{ secrets.TOKEN }}'",
			expected: []string{"${{ secrets.TOKEN }}"},
		},
		{
			name: "map with nested secret returns it",
			input: map[string]any{
				"token": "${{ secrets.GH_TOKEN }}",
				"plain": "hello",
			},
			expected: []string{"${{ secrets.GH_TOKEN }}"},
		},
		{
			name: "slice with secret returns it",
			input: []any{
				"no secret here",
				"${{ secrets.MY_SECRET }}",
			},
			expected: []string{"${{ secrets.MY_SECRET }}"},
		},
		{
			name: "deeply nested secret is found",
			input: map[string]any{
				"env": map[string]any{
					"API_KEY": "${{ secrets.API_KEY }}",
				},
			},
			expected: []string{"${{ secrets.API_KEY }}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSecretsFromStepValue(tt.input)
			if len(tt.expected) == 0 {
				assert.Empty(t, result, "expected no secrets")
			} else {
				assert.Len(t, result, len(tt.expected), "unexpected number of secrets extracted")
				for _, expected := range tt.expected {
					assert.Contains(t, result, expected, "expected %q to be in results", expected)
				}
			}
		})
	}
}

func TestDeduplicateStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice returns empty",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates returns same",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "duplicates are removed preserving order",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sliceutil.Deduplicate(tt.input)
			assert.Equal(t, tt.expected, result, "unexpected deduplication result")
		})
	}
}
