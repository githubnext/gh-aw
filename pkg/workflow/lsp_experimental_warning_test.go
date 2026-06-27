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

// TestLSPExperimentalWarning tests that using the lsp frontmatter field
// emits an experimental feature warning at compile time.
func TestLSPExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "lsp field produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
lsp:
  typescript:
    command: typescript-language-server
    args: ["--stdio"]
    fileExtensions:
      ".ts": typescript
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "no lsp field does not produce experimental warning",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "lsp-experimental-warning-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
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
			io.Copy(&buf, r)
			stderrOutput := buf.String()

			if err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
				return
			}

			expectedMessage := "Using experimental feature: lsp"

			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Expected warning containing '%s', got stderr:\n%s", expectedMessage, stderrOutput)
				}
				if compiler.GetWarningCount() == 0 {
					t.Error("Expected warning count > 0 but got 0")
				}
				return
			}

			if strings.Contains(stderrOutput, expectedMessage) {
				t.Errorf("Did not expect warning '%s', but got stderr:\n%s", expectedMessage, stderrOutput)
			}
		})
	}
}
