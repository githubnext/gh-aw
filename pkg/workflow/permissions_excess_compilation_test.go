package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExcessPermissionsCompilationBehavior(t *testing.T) {
	tests := []struct {
		name                   string
		content                string
		strictMode             bool
		expectCompileSuccess   bool
		expectWarning          bool
		expectedWarningMessage string
	}{
		{
			name: "Excess permissions in strict mode - should warn but compile",
			content: `---
on: push
permissions:
  contents: read
  issues: read
  actions: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [repos]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			strictMode:             true,
			expectCompileSuccess:   true,
			expectWarning:          true,
			expectedWarningMessage: "WARNING: Over-provisioned permissions",
		},
		{
			name: "Excess permissions in regular mode - should ignore silently",
			content: `---
on: push
permissions:
  contents: write
  issues: write
  actions: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [repos]
---

# Test Workflow`,
			strictMode:           false,
			expectCompileSuccess: true,
			expectWarning:        false,
		},
		{
			name: "No excess permissions in strict mode - should compile without warning",
			content: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [repos]
    read-only: true
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			strictMode:           true,
			expectCompileSuccess: true,
			expectWarning:        false,
		},
		{
			name: "No excess permissions in regular mode - should compile without warning",
			content: `---
on: push
permissions:
  contents: write
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [repos]
---

# Test Workflow`,
			strictMode:           false,
			expectCompileSuccess: true,
			expectWarning:        false,
		},
		{
			name: "Missing permissions in strict mode - should fail",
			content: `---
on: push
permissions:
  actions: read
engine: copilot
tools:
  github:
    toolsets: [repos]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			strictMode:           true,
			expectCompileSuccess: false,
			expectWarning:        false,
		},
		{
			name: "Missing permissions in regular mode - should fail",
			content: `---
on: push
permissions:
  actions: read
engine: copilot
tools:
  github:
    toolsets: [repos]
---

# Test Workflow`,
			strictMode:           false,
			expectCompileSuccess: false,
			expectWarning:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "excess-permissions-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(tt.strictMode)

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			err = compiler.CompileWorkflow(testFile)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr

			// Read captured output
			var capturedOutput strings.Builder
			buf := make([]byte, 1024)
			for {
				n, readErr := r.Read(buf)
				if n > 0 {
					capturedOutput.Write(buf[:n])
				}
				if readErr != nil {
					break
				}
			}
			r.Close()

			output := capturedOutput.String()

			// Check compilation result
			if tt.expectCompileSuccess && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if !tt.expectCompileSuccess && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			}

			// Check for warnings
			if tt.expectWarning {
				if !strings.Contains(output, "WARNING") {
					t.Errorf("Expected warning in output but none found. Output: %s", output)
				}
				if tt.expectedWarningMessage != "" && !strings.Contains(output, tt.expectedWarningMessage) {
					t.Errorf("Expected warning message containing '%s' but got: %s", tt.expectedWarningMessage, output)
				}
			} else {
				if strings.Contains(output, "WARNING") {
					t.Errorf("Did not expect warning but found one. Output: %s", output)
				}
			}
		})
	}
}

func TestExcessPermissionsWithAllToolset(t *testing.T) {
	// When using 'all' toolset, excess permission checking is disabled
	content := `---
on: push
permissions:
  contents: write
  issues: write
  actions: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [all]
---

# Test Workflow with All Toolset`

	tmpDir, err := os.MkdirTemp("", "all-toolset-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Test in both strict and regular modes
	for _, strictMode := range []bool{true, false} {
		t.Run(fmt.Sprintf("strict=%v", strictMode), func(t *testing.T) {
			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(strictMode)

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			err = compiler.CompileWorkflow(testFile)

			w.Close()
			os.Stderr = oldStderr

			var capturedOutput strings.Builder
			buf := make([]byte, 1024)
			for {
				n, readErr := r.Read(buf)
				if n > 0 {
					capturedOutput.Write(buf[:n])
				}
				if readErr != nil {
					break
				}
			}
			r.Close()

			output := capturedOutput.String()

			// Should not warn about excess permissions with 'all' toolset
			// (but might have missing permissions warnings)
			if strings.Contains(output, "Over-provisioned") {
				t.Errorf("Should not warn about over-provisioned permissions with 'all' toolset. Output: %s", output)
			}
		})
	}
}
