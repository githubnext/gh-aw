//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLabelNames(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []string
	}{
		{
			name: "single label name as string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"types": []any{"labeled"},
					},
					"labels": "panel-review",
				},
			},
			expected: []string{"panel-review"},
		},
		{
			name: "multiple label names as array",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"types": []any{"labeled"},
					},
					"labels": []any{"panel-review", "needs-triage"},
				},
			},
			expected: []string{"panel-review", "needs-triage"},
		},
		{
			name: "no labels field returns nil",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"types": []any{"labeled"},
					},
				},
			},
			expected: nil,
		},
		{
			name:        "no on section returns nil",
			frontmatter: map[string]any{},
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractLabelNames(tt.frontmatter)
			assert.Equal(t, tt.expected, result, "extractLabelNames should return expected label names")
		})
	}
}

func TestBuildLabelNamesCondition(t *testing.T) {
	tests := []struct {
		name       string
		labelNames []string
		expected   string
	}{
		{
			name:       "single label name",
			labelNames: []string{"panel-review"},
			expected:   "github.event.label == null || github.event.label.name == 'panel-review'",
		},
		{
			name:       "multiple label names",
			labelNames: []string{"panel-review", "needs-triage"},
			expected:   "github.event.label == null || github.event.label.name == 'panel-review' || github.event.label.name == 'needs-triage'",
		},
		{
			name:       "label name with single quote",
			labelNames: []string{"can't-repro"},
			expected:   "github.event.label == null || github.event.label.name == 'can''t-repro'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLabelNamesCondition(tt.labelNames)
			assert.Equal(t, tt.expected, result, "buildLabelNamesCondition should return expected condition")
		})
	}
}

// TestLabelNamesPreActivationFilter verifies that on.labels generates a job-level
// if: condition on the pre_activation job that skips the workflow when the triggering
// label does not match (gray ⊘ rather than red ❌).
func TestLabelNamesPreActivationFilter(t *testing.T) {
	tmpDir := testutil.TempDir(t, "labels-filter-test")
	compiler := NewCompiler()

	tests := []struct {
		name         string
		frontmatter  string
		expectedIf   string
		shouldHaveIf bool
	}{
		{
			name: "pull_request_target with single labels",
			frontmatter: `---
on:
  pull_request_target:
    types: [labeled]
  labels: panel-review

permissions:
  contents: read
  pull-requests: read
  issues: read

strict: false
tools:
  github:
    allowed: [get_pull_request]
---`,
			expectedIf:   "github.event.label == null || github.event.label.name == 'panel-review'",
			shouldHaveIf: true,
		},
		{
			name: "pull_request_target with multiple labels",
			frontmatter: `---
on:
  pull_request_target:
    types: [labeled]
  labels: [panel-review, needs-triage]

permissions:
  contents: read
  pull-requests: read
  issues: read

strict: false
tools:
  github:
    allowed: [get_pull_request]
---`,
			expectedIf:   "github.event.label == null || github.event.label.name == 'panel-review' || github.event.label.name == 'needs-triage'",
			shouldHaveIf: true,
		},
		{
			// Negative test: no on.labels specified → the label-filter condition should not appear.
			// expectedIf is set to a substring of the filter expression to confirm its absence.
			name: "pull_request_target without labels has no label-filter if condition",
			frontmatter: `---
on:
  pull_request_target:
    types: [labeled]

permissions:
  contents: read
  pull-requests: read
  issues: read

strict: false
tools:
  github:
    allowed: [get_pull_request]
---`,
			expectedIf:   "github.event.label == null",
			shouldHaveIf: false,
		},
		{
			name: "issues with labels generates pre-activation if condition",
			frontmatter: `---
on:
  issues:
    types: [labeled]
  labels: [bug, enhancement]

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			expectedIf:   "github.event.label == null || github.event.label.name == 'bug' || github.event.label.name == 'enhancement'",
			shouldHaveIf: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := tmpDir + "/test-" + strings.ReplaceAll(tt.name, " ", "-") + ".md"
			content := tt.frontmatter + "\n\n# Test Workflow\n\nTest labels filter."
			require.NoError(t, os.WriteFile(testFile, []byte(content), 0644), "should write test file")

			err := compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "should compile workflow successfully")

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockBytes, err := os.ReadFile(lockFile)
			require.NoError(t, err, "should read lock file")
			lockContent := string(lockBytes)

			// Clean up
			os.Remove(testFile)
			os.Remove(lockFile)

			if tt.shouldHaveIf {
				assert.Contains(t, lockContent, tt.expectedIf,
					"pre_activation job should have if condition matching label filter")
			} else {
				assert.NotContains(t, lockContent, tt.expectedIf,
					"pre_activation job should not have label-name if condition when labels not specified")
			}
		})
	}
}
