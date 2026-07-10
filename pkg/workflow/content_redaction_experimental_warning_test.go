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

func TestContentRedactionExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "content-redaction enabled produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  add-comment:
  content-redaction:
    policies: "Do not disclose security vulnerabilities"
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "no content-redaction does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  add-comment:
---

# Test Workflow
`,
			expectWarning: false,
		},
		{
			name: "content-redaction with inline string produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  add-comment:
  content-redaction: "Never disclose internal credentials"
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "content-redaction with array produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  add-comment:
  content-redaction:
    - "https://corp.example.com/policy.md"
    - "Never disclose CVE IDs before fix"
---

# Test Workflow
`,
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "content-redaction-experimental-warning-test")

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
				t.Errorf("expected compilation to succeed but it failed: %v", err)
				return
			}

			expectedMessage := "Using experimental feature: content-redaction"
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
