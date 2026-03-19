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

// TestGuardPolicyNoExperimentalWarning tests that the tools.github guard policy
// (repos/min-integrity) does not emit an experimental warning, as the feature
// is no longer considered experimental.
func TestGuardPolicyNoExperimentalWarning(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "guard policy enabled does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    repos: all
    min-integrity: unapproved
permissions:
  contents: read
---

# Test Workflow
`,
		},
		{
			name: "no guard policy does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
---

# Test Workflow
`,
		},
		{
			name: "github tool without guard policy does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    toolsets:
      - default
permissions:
  contents: read
---

# Test Workflow
`,
		},
		{
			name: "guard policy with repos array does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    repos:
      - owner/repo
    min-integrity: approved
permissions:
  contents: read
---

# Test Workflow
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "guard-policy-experimental-warning-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			err := compiler.CompileWorkflow(testFile)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			io.Copy(&buf, r)
			stderrOutput := buf.String()

			if err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
				return
			}

			unexpectedMessage := "Using experimental feature: tools.github guard policy (repos/min-integrity)"
			if strings.Contains(stderrOutput, unexpectedMessage) {
				t.Errorf("Did not expect experimental warning '%s', but got stderr:\n%s", unexpectedMessage, stderrOutput)
			}
		})
	}
}
