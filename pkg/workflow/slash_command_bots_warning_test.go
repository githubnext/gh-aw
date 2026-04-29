//go:build integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestSlashCommandBotsWarning tests that combining slash_command and bots triggers
// emits a compile-time warning about the potential conflict.
func TestSlashCommandBotsWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "slash_command with bots produces conflict warning",
			content: `---
on:
  slash_command:
    name: rust-review
    events: [pull_request, pull_request_comment]
  bots:
    - "copilot[bot]"
engine: copilot
permissions:
  contents: read
  pull-requests: read
  issues: read
---

# Rust Review Workflow
Review Rust code on demand.
`,
			expectWarning: true,
		},
		{
			name: "slash_command without bots does not produce warning",
			content: `---
on:
  slash_command:
    name: rust-review
    events: [pull_request, pull_request_comment]
engine: copilot
permissions:
  contents: read
  pull-requests: read
  issues: read
---

# Rust Review Workflow
Review Rust code on demand.
`,
			expectWarning: false,
		},
		{
			name: "bots without slash_command does not produce warning",
			content: `---
on:
  pull_request:
    types: [opened]
  bots:
    - "dependabot[bot]"
engine: copilot
permissions:
  contents: read
  pull-requests: read
---

# Dependabot Workflow
Handle dependabot PRs.
`,
			expectWarning: false,
		},
		{
			name: "slash_command with multiple bots produces conflict warning",
			content: `---
on:
  slash_command:
    name: review
  bots:
    - "copilot[bot]"
    - "renovate[bot]"
engine: copilot
permissions:
  contents: read
  pull-requests: read
  issues: read
---

# Review Workflow
Review code on demand.
`,
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "slash-command-bots-warning-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			oldStderr := os.Stderr
			r, w, pipeErr := os.Pipe()
			if pipeErr != nil {
				t.Fatal(pipeErr)
			}
			os.Stderr = w
			t.Cleanup(func() {
				os.Stderr = oldStderr
				_ = w.Close()
				_ = r.Close()
			})

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			err := compiler.CompileWorkflow(testFile)

			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			os.Stderr = oldStderr
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatal(err)
			}
			stderrOutput := buf.String()

			if err != nil {
				t.Errorf("expected compilation to succeed but it failed: %v", err)
				return
			}

			expectedPhrase := "Both slash_command and bots triggers are configured"
			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedPhrase) {
					t.Errorf("expected warning containing %q, got stderr:\n%s", expectedPhrase, stderrOutput)
				}
				if compiler.GetWarningCount() == 0 {
					t.Error("expected warning count > 0 but got 0")
				}
				return
			}

			if strings.Contains(stderrOutput, expectedPhrase) {
				t.Errorf("did not expect warning %q, but got stderr:\n%s", expectedPhrase, stderrOutput)
			}
		})
	}
}
