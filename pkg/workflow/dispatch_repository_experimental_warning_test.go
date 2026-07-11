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

// TestDispatchRepositoryNoExperimentalWarning tests that dispatch-repository no longer emits an experimental warning.
func TestDispatchRepositoryNoExperimentalWarning(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "dispatch_repository enabled does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch_repository:
    trigger_ci:
      description: Trigger CI
      workflow: ci.yml
      event_type: ci_trigger
      repository: org/target-repo
---

# Test Workflow
`,
		},
		{
			name: "no dispatch_repository does not produce experimental warning",
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
			name: "dispatch_repository with allowed_repositories does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch_repository:
    notify_service:
      workflow: notify.yml
      event_type: notify_event
      allowed_repositories:
        - org/service-repo
        - org/backup-repo
---

# Test Workflow
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "dispatch-repository-no-experimental-warning-test")

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

			expectedMessage := "Using experimental feature: dispatch-repository"

			if strings.Contains(stderrOutput, expectedMessage) {
				t.Errorf("Did not expect warning '%s', but got stderr:\n%s", expectedMessage, stderrOutput)
			}

			// Verify warning count does not include dispatch_repository warning
			if compiler.GetWarningCount() != 0 {
				t.Errorf("Expected warning count to be 0, got %d", compiler.GetWarningCount())
			}
		})
	}
}
