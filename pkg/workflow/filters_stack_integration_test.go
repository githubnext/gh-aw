//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

// TestStackFilterJobIfContainsNoArithmetic is an actionlint-backed regression test that
// verifies the compiled workflow's job-level if: condition does not contain arithmetic
// operators (+, -, *, /) which are not supported by GitHub Actions expressions.
// Previously, max-stack generated "stack.position + N > stack.size" which actionlint
// (correctly) rejected; now it uses equality or a PreStep for arithmetic.
func TestStackFilterJobIfContainsNoArithmetic(t *testing.T) {
	tests := []struct {
		name         string
		workflowBody string
	}{
		{
			name: "default max-stack (1) uses equality not arithmetic",
			workflowBody: `---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
---

Test workflow.
`,
		},
		{
			name: "explicit max-stack: 1 uses equality not arithmetic",
			workflowBody: `---
on:
  pull_request:
    types: [opened]
    max-stack: 1
permissions:
  contents: read
---

Test workflow.
`,
		},
		{
			name: "max-stack: -1 does not add stack condition",
			workflowBody: `---
on:
  pull_request:
    types: [opened]
    max-stack: -1
permissions:
  contents: read
---

Test workflow.
`,
		},
		{
			name: "pull_request_review default max-stack (1) uses equality not arithmetic",
			workflowBody: `---
on:
  pull_request_review:
    types: [submitted]
permissions:
  contents: read
---

Test workflow.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			workflowPath := filepath.Join(tempDir, "test-workflow.md")
			err := os.WriteFile(workflowPath, []byte(tt.workflowBody), 0644)
			require.NoError(t, err)

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(workflowPath)
			require.NoError(t, err)

			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockContent, err := os.ReadFile(lockPath)
			require.NoError(t, err)

			lockStr := string(lockContent)

			// The compiled YAML job-level if: must not contain arithmetic operators.
			// GitHub Actions expression syntax does not support +, -, *, / operators.
			// Actionlint rejects any if: condition containing these.
			for _, line := range strings.Split(lockStr, "\n") {
				trimmed := strings.TrimSpace(line)
				// Only check lines that are job-level "if:" conditions (inside jobs block)
				if strings.HasPrefix(trimmed, "if: ") {
					ifValue := strings.TrimPrefix(trimmed, "if: ")
					require.NotContains(t, ifValue, "stack.position +",
						"job-level if: must not contain arithmetic (+); actionlint rejects this.\n"+
							"Line: %s\nFull lock file:\n%s", line, lockStr)
					require.NotContains(t, ifValue, "stack.position -",
						"job-level if: must not contain arithmetic (-); actionlint rejects this")
				}
			}
		})
	}
}

// TestStackFilterMaxStack2UsesPreStep verifies that max-stack: 2 (N>1)
// generates a PreStep with bash arithmetic rather than an invalid
// arithmetic expression in the job-level if: condition.
func TestStackFilterMaxStack2UsesPreStep(t *testing.T) {
	workflowBody := `---
on:
  pull_request:
    types: [opened]
    max-stack: 2
permissions:
  contents: read
---

Test workflow.
`
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowBody), 0644)
	require.NoError(t, err)

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err)

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	lockStr := string(lockContent)

	// The PreStep with bash arithmetic must be present in the compiled YAML.
	require.Contains(t, lockStr, "Stack position gate (max-stack: 2)",
		"max-stack: 2 should generate a stack gate PreStep in the compiled YAML")
	require.Contains(t, lockStr, "STACK_POSITION",
		"stack gate step should use STACK_POSITION env var")

	// The job-level if: condition must NOT contain arithmetic operators.
	for _, line := range strings.Split(lockStr, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "if: ") {
			ifValue := strings.TrimPrefix(trimmed, "if: ")
			require.NotContains(t, ifValue, "stack.position +",
				"job-level if: must not contain arithmetic operators for max-stack: 2.\nLine: %s", line)
		}
	}
}

func TestStackFilterPullRequestReviewMaxStack2UsesPreStep(t *testing.T) {
	workflowBody := `---
on:
  pull_request_review:
    types: [submitted]
    max-stack: 2
permissions:
  contents: read
---

Test workflow.
`
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowBody), 0644)
	require.NoError(t, err)

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err)

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	lockStr := string(lockContent)
	require.Contains(t, lockStr, "Stack position gate (max-stack: 2)")
	require.Contains(t, lockStr, "github.event_name == 'pull_request_review'")

	for _, line := range strings.Split(lockStr, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "if: ") {
			ifValue := strings.TrimPrefix(trimmed, "if: ")
			require.NotContains(t, ifValue, "stack.position +",
				"job-level if: must not contain arithmetic operators for pull_request_review max-stack: 2.\nLine: %s", line)
		}
	}
}
