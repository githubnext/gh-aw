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

// TestGuardPolicyExperimentalWarning tests that the tools.github guard policy
// (repos/min-integrity) emits an experimental warning when enabled.
func TestGuardPolicyExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "guard policy enabled produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    repos: all
    min-integrity: reader
permissions:
  contents: read
---

# Test Workflow
`,
			expectWarning: true,
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
			expectWarning: false,
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
			expectWarning: false,
		},
		{
			name: "guard policy with repos array produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    repos:
      - owner/repo
    min-integrity: writer
permissions:
  contents: read
---

# Test Workflow
`,
			expectWarning: true,
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

			expectedMessage := "Using experimental feature: tools.github guard policy (repos/min-integrity)"

			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Expected warning containing '%s', got stderr:\n%s", expectedMessage, stderrOutput)
				}
			} else {
				if strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Did not expect warning '%s', but got stderr:\n%s", expectedMessage, stderrOutput)
				}
			}

			// Verify warning count includes guard policy warning
			if tt.expectWarning {
				warningCount := compiler.GetWarningCount()
				if warningCount == 0 {
					t.Error("Expected warning count > 0 but got 0")
				}
			}
		})
	}
}
