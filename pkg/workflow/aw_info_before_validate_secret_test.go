//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestValidateSecretBeforeAwInfo verifies that the validate-secret step in the activation job
// appears before the generate_aw_info step in the agent job in the generated workflow.
// The validate-secret step runs in the activation job, which executes before the agent job.
func TestValidateSecretBeforeAwInfo(t *testing.T) {
	tests := []struct {
		name            string
		workflowContent string
		engine          string
	}{
		{
			name: "copilot engine",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
---

# Test Copilot Workflow

This workflow tests that validate-secret appears before generate_aw_info.
`,
			engine: "copilot",
		},
		{
			name: "claude engine",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test Claude Workflow

This workflow tests that validate-secret appears before generate_aw_info.
`,
			engine: "claude",
		},
		{
			name: "codex engine",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: codex
---

# Test Codex Workflow

This workflow tests that validate-secret appears before generate_aw_info.
`,
			engine: "codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := testutil.TempDir(t, "aw-info-order-test")

			// Create test file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read generated lock file: %v", err)
			}

			lockStr := string(lockContent)

			// Find the positions of both steps
			validateSecretPos := strings.Index(lockStr, "id: validate-secret")
			awInfoPos := strings.Index(lockStr, "id: generate_aw_info")

			// Both steps should exist
			if validateSecretPos == -1 {
				t.Error("Expected 'id: validate-secret' to be present in generated workflow")
			}
			if awInfoPos == -1 {
				t.Error("Expected 'id: generate_aw_info' to be present in generated workflow")
			}

			// validate-secret (activation job) must come before generate_aw_info (agent job)
			if validateSecretPos != -1 && awInfoPos != -1 {
				if validateSecretPos > awInfoPos {
					t.Errorf("Step ordering error: validate-secret (pos %d) should come before generate_aw_info (pos %d)",
						validateSecretPos, awInfoPos)
				} else {
					t.Logf("✓ Step ordering correct: validate-secret (pos %d) comes before generate_aw_info (pos %d)",
						validateSecretPos, awInfoPos)
				}
			}
		})
	}
}
