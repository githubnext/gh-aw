//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestExpressionExtractor_ExtractExpressions(t *testing.T) {
	tests := []struct {
		name            string
		markdown        string
		wantCount       int
		wantExpressions []string
	}{
		{
			name:      "no expressions",
			markdown:  "This is plain text without any expressions",
			wantCount: 0,
		},
		{
			name:            "single simple expression",
			markdown:        "Repository: ${{ github.repository }}",
			wantCount:       1,
			wantExpressions: []string{"github.repository"},
		},
		{
			name:            "multiple expressions",
			markdown:        "Repo: ${{ github.repository }}, Actor: ${{ github.actor }}, Run: ${{ github.run_id }}",
			wantCount:       3,
			wantExpressions: []string{"github.repository", "github.actor", "github.run_id"},
		},
		{
			name:            "duplicate expressions",
			markdown:        "First: ${{ github.repository }}, Second: ${{ github.repository }}",
			wantCount:       1,
			wantExpressions: []string{"github.repository"},
		},
		{
			name:            "expression with operators",
			markdown:        "Issue: ${{ github.event.issue.number || github.event.pull_request.number }}",
			wantCount:       1,
			wantExpressions: []string{"github.event.issue.number || github.event.pull_request.number"},
		},
		{
			name:            "expression in URL",
			markdown:        "Link: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}",
			wantCount:       3,
			wantExpressions: []string{"github.server_url", "github.repository", "github.run_id"},
		},
		{
			name:            "needs.activation.outputs.text gets transformed",
			markdown:        "Content: ${{ needs.activation.outputs.text }}",
			wantCount:       1,
			wantExpressions: []string{"steps.sanitized.outputs.text"},
		},
		{
			name:            "expression with whitespace",
			markdown:        "Value: ${{  github.actor  }}",
			wantCount:       1,
			wantExpressions: []string{"github.actor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExpressionExtractor()
			mappings, err := extractor.ExtractExpressions(tt.markdown)

			if err != nil {
				t.Errorf("ExtractExpressions() error = %v", err)
				return
			}

			if len(mappings) != tt.wantCount {
				t.Errorf("ExtractExpressions() got %d mappings, want %d", len(mappings), tt.wantCount)
			}

			// Verify expected expressions are present
			for _, wantExpr := range tt.wantExpressions {
				found := false
				for _, mapping := range mappings {
					if mapping.Content == wantExpr {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractExpressions() missing expected expression: %s", wantExpr)
				}
			}
		})
	}
}

func TestExpressionExtractor_GenerateEnvVarName(t *testing.T) {
	extractor := NewExpressionExtractor()

	tests := []struct {
		name     string
		content  string
		wantName string // expected env var name for simple expressions
	}{
		{
			name:     "simple expression",
			content:  "github.repository",
			wantName: "GH_AW_GITHUB_REPOSITORY",
		},
		{
			name:     "expression with underscore",
			content:  "github.run_id",
			wantName: "GH_AW_GITHUB_RUN_ID",
		},
		{
			name:     "nested expression",
			content:  "github.event.issue.number",
			wantName: "GH_AW_GITHUB_EVENT_ISSUE_NUMBER",
		},
		{
			name:     "needs output",
			content:  "needs.activation.outputs.text",
			wantName: "GH_AW_NEEDS_ACTIVATION_OUTPUTS_TEXT",
		},
		{
			name:    "complex expression with operators",
			content: "github.event.issue.number || github.event.pull_request.number",
			// Falls back to hash-based name for complex expressions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVar := extractor.generateEnvVarName(tt.content)

			// Check that env var has correct prefix
			if !strings.HasPrefix(envVar, "GH_AW_") {
				t.Errorf("generateEnvVarName() = %s, want prefix GH_AW_", envVar)
			}

			// Check that env var is uppercase
			if envVar != strings.ToUpper(envVar) {
				t.Errorf("generateEnvVarName() = %s, want uppercase", envVar)
			}

			// Check expected name for simple expressions
			if tt.wantName != "" && envVar != tt.wantName {
				t.Errorf("generateEnvVarName() = %s, want %s", envVar, tt.wantName)
			}

			// For complex expressions, check that it falls back to hash-based name
			if tt.wantName == "" && !strings.HasPrefix(envVar, "GH_AW_EXPR_") {
				t.Errorf("generateEnvVarName() = %s, want hash-based name with prefix GH_AW_EXPR_", envVar)
			}

			// Check that same content generates same env var (deterministic)
			envVar2 := extractor.generateEnvVarName(tt.content)
			if envVar != envVar2 {
				t.Errorf("generateEnvVarName() not deterministic: %s != %s", envVar, envVar2)
			}
		})
	}
}

func TestExpressionExtractor_ReplaceExpressionsWithEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "no expressions",
			markdown: "This is plain text",
			want:     "This is plain text",
		},
		{
			name:     "single expression",
			markdown: "Repository: ${{ github.repository }}",
			want:     "", // Will be replaced with env var, we check structure below
		},
		{
			name:     "multiple expressions",
			markdown: "Repo: ${{ github.repository }}, Actor: ${{ github.actor }}",
			want:     "", // Will be replaced with env vars
		},
		{
			name:     "duplicate expressions use same env var",
			markdown: "First: ${{ github.repository }}, Second: ${{ github.repository }}",
			want:     "", // Both should be replaced with same env var
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExpressionExtractor()
			_, err := extractor.ExtractExpressions(tt.markdown)
			if err != nil {
				t.Errorf("ExtractExpressions() error = %v", err)
				return
			}

			result := extractor.ReplaceExpressionsWithEnvVars(tt.markdown)

			// Verify that original expressions are gone
			if strings.Contains(result, "${{") {
				t.Errorf("ReplaceExpressionsWithEnvVars() still contains ${{ expressions: %s", result)
			}

			// Verify that we have env var references if there were expressions
			mappings := extractor.GetMappings()
			if len(mappings) > 0 {
				// Check that at least one env var reference is present
				hasEnvVarRef := false
				for _, mapping := range mappings {
					if strings.Contains(result, "__"+mapping.EnvVar+"__") {
						hasEnvVarRef = true
						break
					}
				}
				if !hasEnvVarRef {
					t.Errorf("ReplaceExpressionsWithEnvVars() missing env var placeholder references: %s", result)
				}
			}

			// Special case: check that duplicate expressions use the same env var
			if tt.name == "duplicate expressions use same env var" {
				mappings := extractor.GetMappings()
				if len(mappings) != 1 {
					t.Errorf("Expected 1 mapping for duplicate expressions, got %d", len(mappings))
				}
				// Count occurrences of the env var in the result
				envVarRef := "__" + mappings[0].EnvVar + "__"
				count := strings.Count(result, envVarRef)
				if count != 2 {
					t.Errorf("Expected env var to appear 2 times, got %d: %s", count, result)
				}
			}
		})
	}
}

func TestExpressionExtractor_CompleteWorkflow(t *testing.T) {
	markdown := `# Test Workflow

Repository: ${{ github.repository }}
Actor: ${{ github.actor }}
Run ID: ${{ github.run_id }}

Link: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}`

	extractor := NewExpressionExtractor()

	// Extract expressions
	mappings, err := extractor.ExtractExpressions(markdown)
	if err != nil {
		t.Fatalf("ExtractExpressions() error = %v", err)
	}

	// Should have 4 unique expressions
	expectedCount := 4
	if len(mappings) != expectedCount {
		t.Errorf("Expected %d mappings, got %d", expectedCount, len(mappings))
	}

	// Replace expressions
	result := extractor.ReplaceExpressionsWithEnvVars(markdown)

	// Verify no original expressions remain
	if strings.Contains(result, "${{") {
		t.Errorf("Result still contains ${{ expressions: %s", result)
	}

	// Verify all env vars are referenced
	for _, mapping := range mappings {
		envVarRef := "__" + mapping.EnvVar + "__"
		if !strings.Contains(result, envVarRef) {
			t.Errorf("Result missing env var placeholder reference %s: %s", envVarRef, result)
		}
	}

	// Verify the structure is intact (just with different placeholders)
	if !strings.Contains(result, "Repository:") {
		t.Errorf("Result missing 'Repository:' text")
	}
	if !strings.Contains(result, "Actor:") {
		t.Errorf("Result missing 'Actor:' text")
	}
	if !strings.Contains(result, "Link:") {
		t.Errorf("Result missing 'Link:' text")
	}
}

func TestExpressionExtractor_NoCollisions(t *testing.T) {
	// Test that different expressions get different env vars
	expressions := []string{
		"github.repository",
		"github.actor",
		"github.run_id",
		"github.event.issue.number",
		"needs.activation.outputs.text",
	}

	extractor := NewExpressionExtractor()
	envVars := make(map[string]bool)

	for _, expr := range expressions {
		envVar := extractor.generateEnvVarName(expr)
		if envVars[envVar] {
			t.Errorf("Collision detected: %s generated duplicate env var %s", expr, envVar)
		}
		envVars[envVar] = true
	}

	// Verify we have as many unique env vars as expressions
	if len(envVars) != len(expressions) {
		t.Errorf("Expected %d unique env vars, got %d", len(expressions), len(envVars))
	}
}

func TestTransformActivationOutputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "transform text output",
			input:    "needs.activation.outputs.text",
			expected: "steps.sanitized.outputs.text",
		},
		{
			name:     "transform title output",
			input:    "needs.activation.outputs.title",
			expected: "steps.sanitized.outputs.title",
		},
		{
			name:     "transform body output",
			input:    "needs.activation.outputs.body",
			expected: "steps.sanitized.outputs.body",
		},
		{
			name:     "no transformation for other outputs",
			input:    "needs.activation.outputs.comment_id",
			expected: "needs.activation.outputs.comment_id",
		},
		{
			name:     "no transformation for other jobs",
			input:    "needs.pre_activation.outputs.activated",
			expected: "needs.pre_activation.outputs.activated",
		},
		{
			name:     "expression with operators",
			input:    "needs.activation.outputs.text || 'default'",
			expected: "steps.sanitized.outputs.text || 'default'",
		},
		{
			name:     "multiple transformations in same expression",
			input:    "needs.activation.outputs.title && needs.activation.outputs.body",
			expected: "steps.sanitized.outputs.title && steps.sanitized.outputs.body",
		},
		{
			name:     "no transformation needed",
			input:    "github.repository",
			expected: "github.repository",
		},
		{
			name:     "no transformation for partial match",
			input:    "needs.activation.outputs.text_custom",
			expected: "needs.activation.outputs.text_custom",
		},
		{
			name:     "transform with trailing operator",
			input:    "needs.activation.outputs.text && true",
			expected: "steps.sanitized.outputs.text && true",
		},
		{
			name:     "transform with trailing parenthesis",
			input:    "func(needs.activation.outputs.text)",
			expected: "func(steps.sanitized.outputs.text)",
		},
		{
			name:     "partial match followed by valid match",
			input:    "needs.activation.outputs.text_custom || needs.activation.outputs.text",
			expected: "needs.activation.outputs.text_custom || steps.sanitized.outputs.text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformActivationOutputs(tt.input)
			if result != tt.expected {
				t.Errorf("transformActivationOutputs() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpressionExtractor_ActivationOutputTransformation(t *testing.T) {
	// Test that the extraction process applies the transformation
	markdown := `# Test

Triggering content: ${{ needs.activation.outputs.text }}
Title: ${{ needs.activation.outputs.title }}
Body: ${{ needs.activation.outputs.body }}
Other: ${{ needs.activation.outputs.comment_id }}
`

	extractor := NewExpressionExtractor()
	mappings, err := extractor.ExtractExpressions(markdown)

	if err != nil {
		t.Fatalf("ExtractExpressions() error = %v", err)
	}

	// Build a map for lookup
	contentMap := make(map[string]*ExpressionMapping)
	for _, mapping := range mappings {
		contentMap[mapping.Content] = mapping
	}

	// Verify transformations were applied
	tests := []struct {
		original    string
		transformed string
		shouldExist bool
	}{
		{
			original:    "needs.activation.outputs.text",
			transformed: "steps.sanitized.outputs.text",
			shouldExist: true,
		},
		{
			original:    "needs.activation.outputs.title",
			transformed: "steps.sanitized.outputs.title",
			shouldExist: true,
		},
		{
			original:    "needs.activation.outputs.body",
			transformed: "steps.sanitized.outputs.body",
			shouldExist: true,
		},
		{
			original:    "needs.activation.outputs.comment_id",
			transformed: "needs.activation.outputs.comment_id",
			shouldExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.transformed, func(t *testing.T) {
			// The original expression should NOT be in the content map
			if _, found := contentMap[tt.original]; found && tt.original != tt.transformed {
				t.Errorf("Found untransformed expression %q in mappings (should be %q)", tt.original, tt.transformed)
			}

			// The transformed expression should be in the content map
			if tt.shouldExist {
				if _, found := contentMap[tt.transformed]; !found {
					t.Errorf("Expected transformed expression %q in mappings", tt.transformed)
				}
			}
		})
	}
}
