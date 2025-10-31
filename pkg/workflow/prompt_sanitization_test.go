package workflow

import (
	"testing"
)

func TestExtractGitHubContextExpressions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string // expression -> env var name
	}{
		{
			name:    "single github.actor expression",
			content: "User: ${{ github.actor }}",
			expected: map[string]string{
				"${{ github.actor }}": "GH_ACTOR",
			},
		},
		{
			name:    "multiple github.event expressions",
			content: "Run: ${{ github.event.workflow_run.id }} URL: ${{ github.event.workflow_run.html_url }}",
			expected: map[string]string{
				"${{ github.event.workflow_run.id }}":       "GH_EVENT_WORKFLOW_RUN_ID",
				"${{ github.event.workflow_run.html_url }}": "GH_EVENT_WORKFLOW_RUN_HTML_URL",
			},
		},
		{
			name:    "duplicate expressions",
			content: "${{ github.actor }} mentioned by ${{ github.actor }}",
			expected: map[string]string{
				"${{ github.actor }}": "GH_ACTOR",
			},
		},
		{
			name:    "mixed github and other expressions",
			content: "Run: ${{ github.run_id }} Needs: ${{ needs.activation.outputs.text }}",
			expected: map[string]string{
				"${{ github.run_id }}": "GH_RUN_ID",
			},
		},
		{
			name:     "no github expressions",
			content:  "Just plain text with ${{ needs.activation.outputs.text }}",
			expected: map[string]string{},
		},
		{
			name:    "github.repository expression",
			content: "Repo: ${{ github.repository }}",
			expected: map[string]string{
				"${{ github.repository }}": "GH_REPOSITORY",
			},
		},
		{
			name:    "expressions in template conditionals are skipped",
			content: "{{#if ${{ github.actor }} }}\nUser: ${{ github.actor }}\n{{/if}}",
			expected: map[string]string{
				// The expression in the content should be sanitized, but not the one in the conditional
				"${{ github.actor }}": "GH_ACTOR",
			},
		},
		{
			name:    "only content expressions are extracted, not conditional expressions",
			content: "{{#if ${{ github.event.issue.number }} }}\nIssue: ${{ github.event.issue.number }}\n{{/if}}",
			expected: map[string]string{
				// Only the one in content, not in the conditional header
				"${{ github.event.issue.number }}": "GH_EVENT_ISSUE_NUMBER",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGitHubContextExpressions(tt.content)

			// Check count matches
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d expressions, got %d", len(tt.expected), len(result))
			}

			// Check each expected expression
			for expr, expectedEnvVar := range tt.expected {
				if contextVar, exists := result[expr]; !exists {
					t.Errorf("Expected expression %q not found", expr)
				} else if contextVar.EnvVarName != expectedEnvVar {
					t.Errorf("For expression %q, expected env var %q, got %q",
						expr, expectedEnvVar, contextVar.EnvVarName)
				}
			}
		})
	}
}

func TestGithubExpressionToEnvVar(t *testing.T) {
	tests := []struct {
		expression string
		expected   string
	}{
		{"github.actor", "GH_ACTOR"},
		{"github.repository", "GH_REPOSITORY"},
		{"github.event.workflow_run.id", "GH_EVENT_WORKFLOW_RUN_ID"},
		{"github.event.workflow_run.html_url", "GH_EVENT_WORKFLOW_RUN_HTML_URL"},
		{"github.event.workflow_run.head_sha", "GH_EVENT_WORKFLOW_RUN_HEAD_SHA"},
		{"github.event.issue.number", "GH_EVENT_ISSUE_NUMBER"},
		{"github.run_id", "GH_RUN_ID"},
		{"github.run_number", "GH_RUN_NUMBER"},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result := githubExpressionToEnvVar(tt.expression)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestReplaceGitHubContextWithEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "replace single expression",
			content:  "User: ${{ github.actor }}",
			expected: "User: ${GH_ACTOR}",
		},
		{
			name:     "replace multiple expressions",
			content:  "Run: ${{ github.event.workflow_run.id }} URL: ${{ github.event.workflow_run.html_url }}",
			expected: "Run: ${GH_EVENT_WORKFLOW_RUN_ID} URL: ${GH_EVENT_WORKFLOW_RUN_HTML_URL}",
		},
		{
			name:     "replace duplicate expressions",
			content:  "${{ github.actor }} mentioned by ${{ github.actor }}",
			expected: "${GH_ACTOR} mentioned by ${GH_ACTOR}",
		},
		{
			name:     "preserve non-github expressions",
			content:  "Run: ${{ github.run_id }} Needs: ${{ needs.activation.outputs.text }}",
			expected: "Run: ${GH_RUN_ID} Needs: ${{ needs.activation.outputs.text }}",
		},
		{
			name: "multiline content",
			content: `## Current Context
- **Repository**: ${{ github.repository }}
- **Workflow Run**: ${{ github.event.workflow_run.id }}
- **Conclusion**: ${{ github.event.workflow_run.conclusion }}
- **Run URL**: ${{ github.event.workflow_run.html_url }}`,
			expected: `## Current Context
- **Repository**: ${GH_REPOSITORY}
- **Workflow Run**: ${GH_EVENT_WORKFLOW_RUN_ID}
- **Conclusion**: ${GH_EVENT_WORKFLOW_RUN_CONCLUSION}
- **Run URL**: ${GH_EVENT_WORKFLOW_RUN_HTML_URL}`,
		},
		{
			name: "preserve template conditional headers",
			content: `{{#if ${{ github.actor }} }}
User: ${{ github.actor }}
{{/if}}`,
			expected: `{{#if ${{ github.actor }} }}
User: ${GH_ACTOR}
{{/if}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First extract the context vars
			contextVars := extractGitHubContextExpressions(tt.content)

			// Then replace them
			result := replaceGitHubContextWithEnvVars(tt.content, contextVars)

			if result != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGenerateEnvVarDefinitions(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedLen int
		contains    []string
	}{
		{
			name:        "single expression",
			content:     "User: ${{ github.actor }}",
			expectedLen: 1,
			contains:    []string{"GH_ACTOR: ${{ github.actor }}"},
		},
		{
			name:        "multiple expressions",
			content:     "Run: ${{ github.event.workflow_run.id }} URL: ${{ github.event.workflow_run.html_url }}",
			expectedLen: 2,
			contains: []string{
				"GH_EVENT_WORKFLOW_RUN_HTML_URL: ${{ github.event.workflow_run.html_url }}",
				"GH_EVENT_WORKFLOW_RUN_ID: ${{ github.event.workflow_run.id }}",
			},
		},
		{
			name:        "no github expressions",
			content:     "Just text",
			expectedLen: 0,
			contains:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contextVars := extractGitHubContextExpressions(tt.content)
			lines := generateEnvVarDefinitions(contextVars)

			if len(lines) != tt.expectedLen {
				t.Errorf("Expected %d env var definitions, got %d", tt.expectedLen, len(lines))
			}

			// Check that all expected strings are present
			for _, expected := range tt.contains {
				found := false
				for _, line := range lines {
					if line == "          "+expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find %q in env var definitions", expected)
				}
			}
		})
	}
}
