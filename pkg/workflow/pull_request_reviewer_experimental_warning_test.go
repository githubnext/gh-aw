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

func TestPullRequestReviewerExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "pull_request_reviewer enabled produces experimental warning",
			content: `---
on:
  pull_request_reviewer:
permissions:
  contents: read
engine: copilot
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "no pull_request_reviewer does not produce experimental warning",
			content: `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Test Workflow
`,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "pull-request-reviewer-experimental-warning-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0600); err != nil {
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
				t.Fatalf("Expected compilation to succeed but it failed: %v", err)
			}

			expectedMessage := "Using experimental feature: pull_request_reviewer"
			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Expected warning containing %q, got stderr:\n%s", expectedMessage, stderrOutput)
				}
				if compiler.GetWarningCount() == 0 {
					t.Error("Expected warning count > 0 but got 0")
				}
			} else if strings.Contains(stderrOutput, expectedMessage) {
				t.Errorf("Did not expect warning %q, but got stderr:\n%s", expectedMessage, stderrOutput)
			}
		})
	}
}
