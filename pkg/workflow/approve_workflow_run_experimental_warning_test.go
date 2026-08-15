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

func TestApproveWorkflowRunExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "approve-workflow-run enabled produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  approve-workflow-run:
    github-token: ${{ secrets.APPROVE_WORKFLOW_RUN_TOKEN }}
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "no approve-workflow-run does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
---

# Test Workflow
`,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "approve-workflow-run-experimental-warning-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			err := compiler.CompileWorkflow(testFile)

			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderrOutput := buf.String()

			if err != nil {
				t.Errorf("expected compilation to succeed but it failed: %v", err)
				return
			}

			expectedMessage := "Using experimental feature: approve-workflow-run"
			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("expected warning containing %q, got stderr:\n%s", expectedMessage, stderrOutput)
				}
				if compiler.GetWarningCount() == 0 {
					t.Error("expected warning count > 0 but got 0")
				}
				return
			}

			if strings.Contains(stderrOutput, expectedMessage) {
				t.Errorf("did not expect warning %q, but got stderr:\n%s", expectedMessage, stderrOutput)
			}
		})
	}
}
