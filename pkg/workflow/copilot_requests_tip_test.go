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

func TestCopilotRequestsEnableTip(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expectTip bool
	}{
		{
			name: "copilot engine without copilot-requests emits tip",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
---

# Test Workflow
`,
			expectTip: true,
		},
		{
			name: "copilot engine with copilot-requests write does not emit tip",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
  copilot-requests: write
---

# Test Workflow
`,
			expectTip: false,
		},
		{
			name: "copilot engine with copilot-requests none does not emit tip",
			content: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
  copilot-requests: none
---

# Test Workflow
`,
			expectTip: false,
		},
		{
			name: "non-copilot engine does not emit tip",
			content: `---
on: workflow_dispatch
engine: claude
permissions:
  contents: read
---

# Test Workflow
`,
			expectTip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "copilot-requests-tip-test")
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
				t.Fatalf("Expected compilation to succeed but it failed: %v", err)
			}

			const tipText = "Tip: set permissions.copilot-requests: write to use GitHub Actions token-based inference"
			if tt.expectTip && !strings.Contains(stderrOutput, tipText) {
				t.Fatalf("Expected copilot-requests tip in stderr, got:\n%s", stderrOutput)
			}
			if !tt.expectTip && strings.Contains(stderrOutput, tipText) {
				t.Fatalf("Did not expect copilot-requests tip in stderr, got:\n%s", stderrOutput)
			}
		})
	}
}
