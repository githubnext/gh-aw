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
			name: "steps with secret in env field only in strict mode is allowed",
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
			expectError: false,
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
			name: "multiple secrets in env bindings only are allowed in strict mode",
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
			expectError: false,
		},
		{
			name: "secrets in both env and run fields in strict mode fails for run only",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Step with mixed secrets",
						"env": map[string]any{
							"SAFE_KEY": "${{ secrets.SAFE_KEY }}",
						},
						"run": "echo ${{ secrets.LEAKED }}",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
		{
			name: "secrets in env only across multiple steps are allowed in strict mode",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Step 1",
						"run":  "my-tool scan",
						"env": map[string]any{
							"SONAR_TOKEN": "${{ secrets.SONAR_TOKEN }}",
						},
					},
					map[string]any{
						"name": "Step 2",
						"run":  "other-tool check",
						"env": map[string]any{
							"CORONA_TOKEN": "${{ secrets.CORONA_TOKEN }}",
							"SI_TOKEN":     "${{ secrets.SI_TOKEN }}",
						},
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "post-steps with secret in env only is allowed in strict mode",
			frontmatter: map[string]any{
				"post-steps": []any{
					map[string]any{
						"name": "Notify",
						"run":  "send-notification",
						"env": map[string]any{
							"SLACK_TOKEN": "${{ secrets.SLACK_TOKEN }}",
						},
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "pre-steps with secret in env only is allowed in strict mode",
			frontmatter: map[string]any{
				"pre-steps": []any{
					map[string]any{
						"name": "Run pre-check with credentials",
						"env": map[string]any{
							"CIAM_CLIENT_ID": "${{ secrets.CIAM_CLIENT_ID }}",
						},
						"run": "ciam-auth verify",
					},
				},
			},
			strictMode:  true,
			expectError: false,
		},
		{
			name: "malformed string env with secret is blocked in strict mode",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Malformed env",
						"env":  "${{ secrets.TOKEN }}",
						"run":  "echo hi",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
		{
			name: "malformed slice env with secret is blocked in strict mode",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Malformed slice env",
						"env": []any{
							"${{ secrets.ARRAY_TOKEN }}",
						},
						"run": "echo hi",
					},
				},
			},
			strictMode:  true,
			expectError: true,
			errorMsg:    "strict mode: secrets expressions detected in 'steps' section",
		},
		{
			name: "env-bound secret with GITHUB_ENV write in run is blocked in strict mode",
			frontmatter: map[string]any{
				"steps": []any{
					map[string]any{
						"name": "Leaky step",
						"env": map[string]any{
							"TOKEN": "${{ secrets.TOKEN }}",
						},
						"run": `echo "TOKEN=${TOKEN}" >> "$GITHUB_ENV"`,
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

func TestClassifyStepSecrets(t *testing.T) {
	tests := []struct {
		name           string
		step           any
		expectedUnsafe []string
		expectedEnv    []string
	}{
		{
			name:           "non-map step classifies all as unsafe",
			step:           "echo ${{ secrets.TOKEN }}",
			expectedUnsafe: []string{"${{ secrets.TOKEN }}"},
			expectedEnv:    nil,
		},
		{
			name: "secret in run field is unsafe",
			step: map[string]any{
				"name": "Run step",
				"run":  "echo ${{ secrets.API_KEY }}",
			},
			expectedUnsafe: []string{"${{ secrets.API_KEY }}"},
			expectedEnv:    nil,
		},
		{
			name: "secret in env field is classified as env",
			step: map[string]any{
				"name": "Env step",
				"env": map[string]any{
					"TOKEN": "${{ secrets.TOKEN }}",
				},
				"run": "echo hi",
			},
			expectedUnsafe: nil,
			expectedEnv:    []string{"${{ secrets.TOKEN }}"},
		},
		{
			name: "secrets in both env and run are classified separately",
			step: map[string]any{
				"name": "Mixed step",
				"env": map[string]any{
					"SAFE": "${{ secrets.SAFE }}",
				},
				"run": "curl ${{ secrets.LEAKED }}",
			},
			expectedUnsafe: []string{"${{ secrets.LEAKED }}"},
			expectedEnv:    []string{"${{ secrets.SAFE }}"},
		},
		{
			name: "secret in with field is unsafe",
			step: map[string]any{
				"uses": "some/action@v1",
				"with": map[string]any{
					"token": "${{ secrets.MY_TOKEN }}",
				},
			},
			expectedUnsafe: []string{"${{ secrets.MY_TOKEN }}"},
			expectedEnv:    nil,
		},
		{
			name: "step with no secrets returns empty",
			step: map[string]any{
				"name": "Plain step",
				"run":  "echo hello",
			},
			expectedUnsafe: nil,
			expectedEnv:    nil,
		},
		{
			name: "secret in malformed string env is unsafe",
			step: map[string]any{
				"name": "Malformed env step",
				"env":  "${{ secrets.TOKEN }}",
				"run":  "echo hi",
			},
			expectedUnsafe: []string{"${{ secrets.TOKEN }}"},
			expectedEnv:    nil,
		},
		{
			name: "secret in malformed slice env is unsafe",
			step: map[string]any{
				"name": "Malformed env slice step",
				"env": []any{
					"${{ secrets.ARRAY_TOKEN }}",
				},
				"run": "echo hi",
			},
			expectedUnsafe: []string{"${{ secrets.ARRAY_TOKEN }}"},
			expectedEnv:    nil,
		},
		{
			name: "env-bound secret with GITHUB_ENV in run is reclassified as unsafe",
			step: map[string]any{
				"name": "Leaky step",
				"env": map[string]any{
					"TOKEN": "${{ secrets.TOKEN }}",
				},
				"run": `echo "TOKEN=${TOKEN}" >> "$GITHUB_ENV"`,
			},
			expectedUnsafe: []string{"${{ secrets.TOKEN }}"},
			expectedEnv:    nil,
		},
		{
			name: "env-bound secret without GITHUB_ENV reference stays env-bound",
			step: map[string]any{
				"name": "Safe step",
				"env": map[string]any{
					"TOKEN": "${{ secrets.TOKEN }}",
				},
				"run": "my-tool --authenticate",
			},
			expectedUnsafe: nil,
			expectedEnv:    []string{"${{ secrets.TOKEN }}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsafe, env := classifyStepSecrets(tt.step)
			if len(tt.expectedUnsafe) == 0 {
				assert.Empty(t, unsafe, "expected no unsafe secrets")
			} else {
				assert.Equal(t, tt.expectedUnsafe, unsafe, "unexpected unsafe secrets")
			}
			if len(tt.expectedEnv) == 0 {
				assert.Empty(t, env, "expected no env secrets")
			} else {
				assert.Equal(t, tt.expectedEnv, env, "unexpected env secrets")
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

func TestFilterBuiltinTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "GITHUB_TOKEN is filtered out",
			input:    []string{"${{ secrets.GITHUB_TOKEN }}"},
			expected: []string{},
		},
		{
			name:     "user secret is kept",
			input:    []string{"${{ secrets.API_KEY }}"},
			expected: []string{"${{ secrets.API_KEY }}"},
		},
		{
			name:     "GITHUB_TOKEN_SUFFIX is NOT filtered (precise match)",
			input:    []string{"${{ secrets.GITHUB_TOKEN_SUFFIX }}"},
			expected: []string{"${{ secrets.GITHUB_TOKEN_SUFFIX }}"},
		},
		{
			name:     "mixed expression with GITHUB_TOKEN and other secret is NOT filtered",
			input:    []string{"${{ secrets.GITHUB_TOKEN && secrets.OTHER }}"},
			expected: []string{"${{ secrets.GITHUB_TOKEN && secrets.OTHER }}"},
		},
		{
			name:     "expression with only GITHUB_TOKEN references is filtered",
			input:    []string{"${{ secrets.GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterBuiltinTokens(tt.input)
			if len(tt.expected) == 0 {
				assert.Empty(t, result, "expected all to be filtered")
			} else {
				assert.Equal(t, tt.expected, result, "unexpected filter result")
			}
		})
	}
}

func TestStepReferencesGitHubEnv(t *testing.T) {
	tests := []struct {
		name     string
		stepMap  map[string]any
		expected bool
	}{
		{
			name: "run with GITHUB_ENV reference",
			stepMap: map[string]any{
				"run": `echo "KEY=val" >> "$GITHUB_ENV"`,
				"env": map[string]any{"K": "v"},
			},
			expected: true,
		},
		{
			name: "run without GITHUB_ENV reference",
			stepMap: map[string]any{
				"run": "my-tool scan",
				"env": map[string]any{"K": "v"},
			},
			expected: false,
		},
		{
			name: "GITHUB_ENV in env field is ignored",
			stepMap: map[string]any{
				"run": "my-tool scan",
				"env": map[string]any{"GITHUB_ENV": "/tmp/env"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stepReferencesGitHubEnv(tt.stepMap)
			assert.Equal(t, tt.expected, result, "unexpected GITHUB_ENV detection result")
		})
	}
}
