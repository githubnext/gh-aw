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

// TestDispatchRepositoryExperimentalWarning tests that the dispatch_repository feature
// emits an experimental warning when enabled.
func TestDispatchRepositoryExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "dispatch_repository enabled produces experimental warning",
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
			expectWarning: true,
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
			expectWarning: false,
		},
		{
			name: "dispatch_repository with allowed_repositories produces experimental warning",
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
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "dispatch-repository-experimental-warning-test")

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

			expectedMessage := "Using experimental feature: dispatch_repository"

			if tt.expectWarning {
				if !strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Expected warning containing '%s', got stderr:\n%s", expectedMessage, stderrOutput)
				}
			} else {
				if strings.Contains(stderrOutput, expectedMessage) {
					t.Errorf("Did not expect warning '%s', but got stderr:\n%s", expectedMessage, stderrOutput)
				}
			}

			// Verify warning count includes dispatch_repository warning
			if tt.expectWarning {
				warningCount := compiler.GetWarningCount()
				if warningCount == 0 {
					t.Error("Expected warning count > 0 but got 0")
				}
			}
		})
	}
}
