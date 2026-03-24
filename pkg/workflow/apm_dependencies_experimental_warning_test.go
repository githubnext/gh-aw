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

// TestAPMDependenciesExperimentalWarning tests that the dependencies (APM) feature
// emits an experimental warning when enabled.
func TestAPMDependenciesExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "dependencies array format produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
dependencies:
  - microsoft/apm-sample-package
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
			name: "dependencies object format produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
dependencies:
  packages:
    - microsoft/apm-sample-package
    - github/awesome-copilot/skills/review-and-refactor
  isolated: false
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
			name: "dependencies with multiple packages produces experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
dependencies:
  - microsoft/apm-sample-package
  - github/awesome-copilot/skills/review-and-refactor
  - microsoft/apm-sample-package#v2.0
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
			name: "no dependencies does not produce experimental warning",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "apm-dependencies-experimental-warning-test")

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

			expectedMessage := "Using experimental feature: dependencies (APM)"

			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Expected warning containing '%s', got stderr:\n%s", expectedMessage, stderrOutput)
				}
			} else {
				if strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Did not expect warning '%s', but got stderr:\n%s", expectedMessage, stderrOutput)
				}
			}

			// Verify warning count includes dependencies warning
			if tt.expectWarning {
				warningCount := compiler.GetWarningCount()
				if warningCount == 0 {
					t.Error("Expected warning count > 0 but got 0")
				}
			}
		})
	}
}
