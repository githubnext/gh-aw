//go:build !integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafeOutputsAppConcurrencyWarning tests that a warning is emitted when
// safe-outputs.github-app, comment-based triggers, and cancel-in-progress: true
// are combined, and that no warning is emitted when any condition is missing.
func TestSafeOutputsAppConcurrencyWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "all three conditions: github-app + issue_comment + cancel-in-progress emits warning",
			content: `---
on:
  issue_comment:
    types: [created]
  workflow_dispatch:
engine: copilot
concurrency:
  group: gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.run_id }}
  cancel-in-progress: true
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  add-comment:
---

Post a comment.
`,
			expectWarning: true,
		},
		{
			name: "slash_command trigger + github-app + cancel-in-progress emits warning",
			content: `---
on:
  slash_command:
    name: test
    events: [issue_comment]
engine: copilot
concurrency:
  group: >-
    gh-aw-${{ github.workflow }}-${{
      github.event.issue.number ||
      github.event.inputs.issue_number ||
      github.run_id
    }}
  cancel-in-progress: true
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  add-comment:
---

Post a comment to the triggering issue.
`,
			expectWarning: true,
		},
		{
			name: "github-app + issue_comment but cancel-in-progress: false does not emit warning",
			content: `---
on:
  issue_comment:
    types: [created]
  workflow_dispatch:
engine: copilot
concurrency:
  group: gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.run_id }}
  cancel-in-progress: false
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  add-comment:
---

Post a comment.
`,
			expectWarning: false,
		},
		{
			name: "github-app + cancel-in-progress but no comment trigger does not emit warning",
			content: `---
on:
  workflow_dispatch:
engine: copilot
concurrency:
  group: gh-aw-${{ github.workflow }}-${{ github.event.inputs.issue_number || github.run_id }}
  cancel-in-progress: true
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  add-comment:
---

Post a comment.
`,
			expectWarning: false,
		},
		{
			name: "issue_comment + cancel-in-progress but no github-app does not emit warning",
			content: `---
on:
  issue_comment:
    types: [created]
  workflow_dispatch:
engine: copilot
concurrency:
  group: gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.run_id }}
  cancel-in-progress: true
safe-outputs:
  add-comment:
---

Post a comment.
`,
			expectWarning: false,
		},
		{
			name: "no safe-outputs at all does not emit warning",
			content: `---
on:
  issue_comment:
    types: [created]
  workflow_dispatch:
engine: copilot
concurrency:
  group: gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.run_id }}
  cancel-in-progress: true
---

Read comments.
`,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "safe-outputs-app-concurrency-warning-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			err := os.WriteFile(testFile, []byte(tt.content), 0600)
			require.NoError(t, err, "Should write test file")

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			compileErr := compiler.CompileWorkflow(testFile)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			io.Copy(&buf, r)
			stderrOutput := buf.String()

			require.NoError(t, compileErr, "Compilation should succeed")

			expectedPhrases := []string{
				"safe-outputs.github-app combined with comment triggers and cancel-in-progress: true",
				"self-cancellation",
				"passive",
				"safe-outputs.concurrency-group does NOT protect against this",
			}

			if tt.expectWarning {
				for _, phrase := range expectedPhrases {
					assert.Contains(t, stderrOutput, phrase,
						"Expected warning to contain %q, got stderr:\n%s", phrase, stderrOutput)
				}
				assert.Contains(t, stderrOutput, "⚠",
					"Expected warning indicator '⚠' in stderr output, got:\n%s", stderrOutput)
				assert.Positive(t, compiler.GetWarningCount(),
					"Expected warning count > 0")
			} else {
				// None of the self-cancel warning phrases should appear
				for _, phrase := range expectedPhrases {
					assert.NotContains(t, stderrOutput, phrase,
						"Did not expect warning containing %q, but got stderr:\n%s", phrase, stderrOutput)
				}
			}
		})
	}
}
