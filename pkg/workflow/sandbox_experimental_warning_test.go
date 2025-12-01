package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestSandboxRuntimeExperimentalWarning tests that the sandbox-runtime feature
// emits an experimental warning when enabled.
func TestSandboxRuntimeExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "sandbox-runtime enabled produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
sandbox: sandbox-runtime
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "sandbox default does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
sandbox: default
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Test Workflow
`,
			expectWarning: false,
		},
		{
			name: "no sandbox config does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Test Workflow
`,
			expectWarning: false,
		},
		{
			name: "sandbox-runtime with custom config produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
sandbox:
  type: sandbox-runtime
  config:
    network:
      allowedDomains:
        - example.com
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Test Workflow
`,
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "sandbox-experimental-warning-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			compiler := NewCompiler(false, "", "test")
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

			expectedMessage := "Using experimental feature: sandbox-runtime firewall"

			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Expected warning containing '%s', got stderr:\n%s", expectedMessage, stderrOutput)
				}
			} else {
				if strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Did not expect warning '%s', but got stderr:\n%s", expectedMessage, stderrOutput)
				}
			}

			// Verify warning count includes sandbox-runtime warning
			if tt.expectWarning {
				warningCount := compiler.GetWarningCount()
				if warningCount == 0 {
					t.Error("Expected warning count > 0 but got 0")
				}
			}
		})
	}
}
