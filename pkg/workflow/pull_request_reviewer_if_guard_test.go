//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestPullRequestReviewerIfGuard verifies that the compiled if: condition for
// pull_request_reviewer workflows uses an allowlist (not a denylist) so that
// comment events (pull_request_review_comment, issue_comment, etc.) cannot
// accidentally trigger the reviewer job – either directly or via workflow_dispatch
// from the central router.
func TestPullRequestReviewerIfGuard(t *testing.T) {
	tmpDir := testutil.TempDir(t, "reviewer-if-guard-test")

	tests := []struct {
		name               string
		content            string
		expectedIfContains string
		notExpectedIfParts []string
	}{
		{
			name: "reviewer workflow produces allowlist if guard",
			content: `---
on:
  pull_request_reviewer:
permissions:
  contents: read
engine: copilot
---

# Reviewer Workflow
`,
			expectedIfContains: "github.event_name == 'workflow_dispatch' || ((github.event_name == 'pull_request' || github.event_name == 'pull_request_review') && github.event.pull_request.state != 'closed')",
			notExpectedIfParts: []string{
				// The old denylist guard must be absent – it allowed comment events through.
				"github.event_name != 'pull_request' && github.event_name != 'pull_request_review'",
			},
		},
		{
			name: "reviewer workflow if guard does not allow comment events",
			content: `---
on:
  pull_request_reviewer: custom-reviewer
permissions:
  contents: read
engine: copilot
---

# Custom Reviewer Workflow
`,
			// The allowlist guard must be present and include workflow_dispatch.
			// Verifies the guard does not use the old denylist form that admitted comments.
			expectedIfContains: "github.event_name == 'workflow_dispatch' || ((github.event_name == 'pull_request' || github.event_name == 'pull_request_review') && github.event.pull_request.state != 'closed')",
			notExpectedIfParts: []string{
				// The old denylist form must not appear in the compiled reviewer guard.
				"github.event_name != 'pull_request' && github.event_name != 'pull_request_review'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("CompileWorkflow failed: %v", err)
			}

			lockFile := stringutil.MarkdownToLockFile(testFile)
			raw, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockContent := string(raw)
			// Normalize whitespace for substring matching across line-wrapped YAML.
			normalised := strings.Join(strings.Fields(lockContent), " ")

			if tt.expectedIfContains != "" {
				normExpected := strings.Join(strings.Fields(tt.expectedIfContains), " ")
				if !strings.Contains(normalised, normExpected) {
					t.Errorf("Expected lock file to contain if guard:\n  %s\nbut it was not found.\nLock content:\n%s", tt.expectedIfContains, lockContent)
				}
			}

			for _, bad := range tt.notExpectedIfParts {
				normBad := strings.Join(strings.Fields(bad), " ")
				if strings.Contains(normalised, normBad) {
					t.Errorf("Lock file must NOT contain:\n  %s\nbut it was found.\nLock content:\n%s", bad, lockContent)
				}
			}
		})
	}
}
