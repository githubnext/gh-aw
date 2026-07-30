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

// TestCopilotSDKNoExperimentalWarning tests that the copilot-sdk feature
// does not emit an experimental warning, as the feature is no longer
// considered experimental.
func TestCopilotSDKNoExperimentalWarning(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "copilot-sdk enabled does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine:
  id: copilot
  copilot-sdk: true
permissions:
  contents: read
---

# Test Workflow
`,
		},
		{
			name: "copilot-sdk disabled does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine:
  id: copilot
  copilot-sdk: false
permissions:
  contents: read
---

# Test Workflow
`,
		},
		{
			name: "no copilot-sdk does not produce experimental warning",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
---

# Test Workflow
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "copilot-sdk-experimental-warning-test")

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

			unexpectedMessage := "Using experimental feature: engine.copilot-sdk"
			if strings.Contains(stderrOutput, unexpectedMessage) {
				t.Errorf("Did not expect experimental warning '%s', but got stderr:\n%s", unexpectedMessage, stderrOutput)
			}
		})
	}
}
