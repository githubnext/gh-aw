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

func TestGradersExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "graders enabled produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
graders: {}
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "no graders does not produce experimental warning",
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
			tmpDir := testutil.TempDir(t, "graders-experimental-warning-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			oldStderr := os.Stderr
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stderr = w
			t.Cleanup(func() {
				os.Stderr = oldStderr
			})

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			compileErr := compiler.CompileWorkflow(testFile)

			_ = w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Fatal(copyErr)
			}
			stderrOutput := buf.String()

			if compileErr != nil {
				t.Errorf("expected compilation to succeed but it failed: %v", compileErr)
				return
			}

			expectedMessage := "Using experimental feature: graders"
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
