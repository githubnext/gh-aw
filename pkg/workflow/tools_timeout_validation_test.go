package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
)

func TestToolsTimeoutValidation(t *testing.T) {
	tests := []struct {
		name          string
		workflowMd    string
		shouldCompile bool
		errorContains string
	}{
		{
			name: "valid timeout - positive value",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  timeout: 60
  github:
---

# Test Timeout

Test workflow.
`,
			shouldCompile: true,
		},
		{
			name: "valid timeout - minimum value (1)",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  timeout: 1
  github:
---

# Test Timeout

Test workflow.
`,
			shouldCompile: true,
		},
		{
			name: "invalid timeout - zero",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  timeout: 0
  github:
---

# Test Timeout

Test workflow.
`,
			shouldCompile: false,
			errorContains: "minimum: got 0, want 1",
		},
		{
			name: "invalid timeout - negative",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  timeout: -5
  github:
---

# Test Timeout

Test workflow.
`,
			shouldCompile: false,
			errorContains: "minimum: got -5, want 1",
		},
		{
			name: "valid startup-timeout - positive value",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  startup-timeout: 120
  github:
---

# Test Startup Timeout

Test workflow.
`,
			shouldCompile: true,
		},
		{
			name: "valid startup-timeout - minimum value (1)",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  startup-timeout: 1
  github:
---

# Test Startup Timeout

Test workflow.
`,
			shouldCompile: true,
		},
		{
			name: "invalid startup-timeout - zero",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  startup-timeout: 0
  github:
---

# Test Startup Timeout

Test workflow.
`,
			shouldCompile: false,
			errorContains: "minimum: got 0, want 1",
		},
		{
			name: "invalid startup-timeout - negative",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  startup-timeout: -10
  github:
---

# Test Startup Timeout

Test workflow.
`,
			shouldCompile: false,
			errorContains: "minimum: got -10, want 1",
		},
		{
			name: "both timeouts valid",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  timeout: 60
  startup-timeout: 120
  github:
---

# Test Both Timeouts

Test workflow.
`,
			shouldCompile: true,
		},
		{
			name: "both timeouts invalid",
			workflowMd: `---
on: workflow_dispatch
engine: claude
tools:
  timeout: 0
  startup-timeout: -5
  github:
---

# Test Both Timeouts Invalid

Test workflow.
`,
			shouldCompile: false,
			errorContains: "minimum:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write to temporary file
			tmpFile, err := os.CreateTemp("", "test-timeout-validation-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer os.Remove(stringutil.MarkdownToLockFile(tmpFile.Name()))

			if _, err := tmpFile.WriteString(tt.workflowMd); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Compile the workflow
			compiler := NewCompiler()
			err = compiler.CompileWorkflow(tmpFile.Name())

			if tt.shouldCompile {
				if err != nil {
					t.Errorf("Expected workflow to compile successfully, but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected workflow compilation to fail, but it succeeded")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			}
		})
	}
}
