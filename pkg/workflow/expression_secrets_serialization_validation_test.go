//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindSecretsSerializationExpressions tests the helper that detects
// ${{ toJSON(secrets) }} patterns in raw string content.
func TestFindSecretsSerializationExpressions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLen  int
		wantExpr []string
	}{
		{
			name:    "no expressions",
			content: "This is a plain markdown file.",
			wantLen: 0,
		},
		{
			name:    "safe single secret expression",
			content: "Token: ${{ secrets.MY_SECRET }}",
			wantLen: 0,
		},
		{
			name:    "safe toJSON on non-secrets context",
			content: "Inputs: ${{ toJSON(inputs) }}",
			wantLen: 0,
		},
		{
			name:    "toJSON on a specific secret property",
			content: "${{ toJSON(secrets.MY_SECRET) }}",
			wantLen: 0,
		},
		{
			name:     "toJSON(secrets) — all secrets serialized",
			content:  "Here are all secrets: ${{ toJSON(secrets) }}",
			wantLen:  1,
			wantExpr: []string{"${{ toJSON(secrets) }}"},
		},
		{
			name: "toJSON(secrets) — case-insensitive function name",
			// Split literal to avoid secret-scanning false positive on the lowercase variant.
			content:  "${{" + " tojson(secrets) }}",
			wantLen:  1,
			wantExpr: []string{"${{" + " tojson(secrets) }}"},
		},
		{
			name:     "toJSON(secrets) — whitespace inside call",
			content:  "${{  toJSON(  secrets  )  }}",
			wantLen:  1,
			wantExpr: []string{"${{  toJSON(  secrets  )  }}"},
		},
		{
			name:    "quoted string literal is not flagged",
			content: "${{ 'toJSON(secrets)' }}",
			wantLen: 0,
		},
		{
			name:    "toJSON(secrets) — duplicate expression deduplication",
			content: "${{ toJSON(secrets) }} and again ${{ toJSON(secrets) }}",
			// findSecretsSerializationExpressions returns raw matches; deduplication
			// happens in validateSecretsSerializationExpressions.
			wantLen: 2,
		},
		{
			name:    "multiple expressions — only dangerous one flagged",
			content: "Safe: ${{ secrets.MY_TOKEN }}\nDangerous: ${{ toJSON(secrets) }}",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSecretsSerializationExpressions(tt.content)
			assert.Len(t, got, tt.wantLen, "unexpected number of matches")
			for _, want := range tt.wantExpr {
				assert.Contains(t, got, want, "expected expression not found in results")
			}
		})
	}
}

// TestValidateSecretsSerializationExpressions tests the Compiler method that
// emits errors in strict mode and warnings in non-strict mode.
func TestValidateSecretsSerializationExpressions(t *testing.T) {
	tests := []struct {
		name            string
		markdownBody    string
		frontmatterYAML string
		rawFrontmatter  map[string]any
		strictMode      bool
		wantError       bool
		errorContains   string
		wantWarning     bool
	}{
		{
			name:         "no expressions — always passes",
			markdownBody: "Tell me the current date.",
			strictMode:   true,
			wantError:    false,
		},
		{
			name:         "safe expression — always passes",
			markdownBody: "Issue: ${{ github.event.issue.number }}",
			strictMode:   true,
			wantError:    false,
		},
		{
			name:         "specific secret reference — always passes",
			markdownBody: "Token: ${{ secrets.MY_TOKEN }}",
			strictMode:   true,
			wantError:    false,
		},
		{
			name:           "toJSON(secrets) in markdown body — strict mode errors",
			markdownBody:   "All secrets: ${{ toJSON(secrets) }}",
			rawFrontmatter: map[string]any{},
			strictMode:     true,
			wantError:      true,
			errorContains:  "strict mode",
		},
		{
			name:           "toJSON(secrets) in markdown body — strict mode error message",
			markdownBody:   "Dump: ${{ toJSON(secrets) }}",
			rawFrontmatter: map[string]any{},
			strictMode:     true,
			wantError:      true,
			errorContains:  "secrets serialization expression(s) detected",
		},
		{
			name:           "toJSON(secrets) in markdown body — non-strict emits warning",
			markdownBody:   "All secrets: ${{ toJSON(secrets) }}",
			rawFrontmatter: map[string]any{"strict": false},
			strictMode:     false,
			wantError:      false,
			wantWarning:    true,
		},
		{
			name:            "toJSON(secrets) in frontmatter YAML — strict mode errors",
			frontmatterYAML: "env:\n  ALL_SECRETS: ${{ toJSON(secrets) }}\n",
			rawFrontmatter:  map[string]any{},
			strictMode:      true,
			wantError:       true,
			errorContains:   "strict mode",
		},
		{
			name:            "toJSON(secrets) in frontmatter YAML — non-strict emits warning",
			frontmatterYAML: "env:\n  ALL_SECRETS: ${{ toJSON(secrets) }}\n",
			rawFrontmatter:  map[string]any{"strict": false},
			strictMode:      false,
			wantError:       false,
			wantWarning:     true,
		},
		{
			name: "toJSON(secrets) case-insensitive — strict mode errors",
			// Split literal to avoid secret-scanning false positive on the uppercase variant.
			markdownBody:   "Dump: ${{" + " TOJSON(SECRETS) }}",
			rawFrontmatter: map[string]any{},
			strictMode:     true,
			wantError:      true,
			errorContains:  "secrets serialization expression(s) detected",
		},
		{
			name:           "frontmatter strict:false overrides strict mode compiler flag",
			markdownBody:   "All: ${{ toJSON(secrets) }}",
			rawFrontmatter: map[string]any{"strict": false},
			strictMode:     false,
			wantError:      false,
			wantWarning:    true,
		},
		{
			name:           "toJSON(steps) is safe and not flagged",
			markdownBody:   "Steps: ${{ toJSON(steps) }}",
			rawFrontmatter: map[string]any{},
			strictMode:     true,
			wantError:      false,
		},
		{
			name:           "toJSON on specific secret property is safe",
			markdownBody:   "${{ toJSON(secrets.MY_KEY) }}",
			rawFrontmatter: map[string]any{},
			strictMode:     true,
			wantError:      false,
		},
		{
			name:           "quoted toJSON(secrets) string literal is safe",
			markdownBody:   "${{ 'toJSON(secrets)' }}",
			rawFrontmatter: map[string]any{},
			strictMode:     true,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			workflowData := &WorkflowData{
				MarkdownContent: tt.markdownBody,
				FrontmatterYAML: tt.frontmatterYAML,
				RawFrontmatter:  tt.rawFrontmatter,
			}

			err := compiler.validateSecretsSerializationExpressions(workflowData)

			if tt.wantError {
				require.Error(t, err, "expected an error but got none")
				if tt.errorContains != "" {
					require.ErrorContains(t, err, tt.errorContains,
						"error should contain expected message",
					)
				}
			} else {
				require.NoError(t, err, "expected no error")
			}

			if tt.wantWarning {
				assert.Equal(t, 1, compiler.GetWarningCount(), "expected exactly one warning")
			} else {
				assert.Equal(t, 0, compiler.GetWarningCount(), "expected no warnings")
			}
		})
	}
}

// TestValidateSecretsSerializationViaValidateExpressions ensures the new check
// is wired into the validateExpressions entry point.
func TestValidateSecretsSerializationViaValidateExpressions(t *testing.T) {
	compiler := NewCompiler()
	compiler.strictMode = true

	workflowData := &WorkflowData{
		Name:            "Test",
		MarkdownContent: "Expose everything: ${{ toJSON(secrets) }}",
		RawFrontmatter:  map[string]any{},
	}

	err := compiler.validateExpressions(workflowData, "/tmp/test.md")
	require.Error(t, err, "expected validateExpressions to surface secrets serialization error")
	require.ErrorContains(t, err, "secrets serialization expression(s) detected")
}

func TestValidateSecretsSerializationNonStrictViaValidateExpressions(t *testing.T) {
	t.Run("toJSON(secrets) only warns in non-strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = false

		workflowData := &WorkflowData{
			Name:            "Test",
			MarkdownContent: "Expose everything: ${{ toJSON(secrets) }}",
			RawFrontmatter:  map[string]any{"strict": false},
		}

		err := compiler.validateExpressions(workflowData, "/tmp/test.md")
		require.NoError(t, err, "non-strict toJSON(secrets) should warn but not error")
		assert.Equal(t, 1, compiler.GetWarningCount(), "expected one warning")
	})

	t.Run("neutralization keeps other unsafe operands visible to allowlist", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.strictMode = false

		workflowData := &WorkflowData{
			Name:            "Test",
			MarkdownContent: "${{ github.event.constructor || toJSON(secrets) }}",
			RawFrontmatter:  map[string]any{"strict": false},
		}

		err := compiler.validateExpressions(workflowData, "/tmp/test.md")
		require.Error(t, err, "dangerous constructor operand should still be rejected")
		require.ErrorContains(t, err, "constructor")
		assert.Equal(t, 1, compiler.GetWarningCount(), "secrets serialization warning should still be emitted")
	})
}
