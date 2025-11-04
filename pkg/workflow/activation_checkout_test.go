package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestActivationJobCheckoutStep tests that the activation job always includes
// a shallow checkout step for the timestamp check
func TestActivationJobCheckoutStep(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		description string
	}{
		{
			name: "basic workflow includes activation checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: claude
---`,
			description: "Activation job should include shallow checkout for timestamp check",
		},
		{
			name: "workflow without contents permission includes activation checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
engine: claude
---`,
			description: "Activation job should include checkout even when main job doesn't need it",
		},
		{
			name: "workflow with reaction includes activation checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  issues: write
engine: claude
---`,
			description: "Activation job with reaction should include checkout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "activation-checkout-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testContent := tt.frontmatter + "\n\n# Test Workflow\n\nTest workflow content."
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")

			// Compile the workflow
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Calculate the lock file path
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

			// Read the generated lock file
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			// Verify activation job exists
			if !strings.Contains(lockContentStr, "activation:") {
				t.Error("Expected activation job to be present")
			}

			// Extract the activation job section
			activationJobStart := strings.Index(lockContentStr, "activation:")
			if activationJobStart == -1 {
				t.Fatal("Activation job not found in compiled workflow")
			}

			// Find the next job or end of file
			activationJobEnd := len(lockContentStr)
			nextJobIdx := strings.Index(lockContentStr[activationJobStart+11:], "\n  ")
			if nextJobIdx != -1 {
				searchStart := activationJobStart + 11 + nextJobIdx
				for idx := searchStart; idx < len(lockContentStr); idx++ {
					if lockContentStr[idx] == '\n' {
						lineStart := idx + 1
						if lineStart < len(lockContentStr) && lineStart+2 < len(lockContentStr) {
							if lockContentStr[lineStart:lineStart+2] == "  " && lockContentStr[lineStart+2] != ' ' {
								colonIdx := strings.Index(lockContentStr[lineStart:], ":")
								if colonIdx > 0 && colonIdx < 50 {
									activationJobEnd = idx
									break
								}
							}
						}
					}
				}
			}

			activationJobSection := lockContentStr[activationJobStart:activationJobEnd]

			// Verify checkout step is present
			if !strings.Contains(activationJobSection, "actions/checkout@") {
				t.Errorf("%s: Activation job should contain checkout step\nSection:\n%s",
					tt.description, activationJobSection)
			}

			// Verify it's a sparse checkout
			if !strings.Contains(activationJobSection, "sparse-checkout:") {
				t.Errorf("%s: Checkout should use sparse-checkout", tt.description)
			}

			// Verify it checks out .github/workflows
			if !strings.Contains(activationJobSection, ".github/workflows") {
				t.Errorf("%s: Should checkout .github/workflows directory", tt.description)
			}

			// Verify shallow clone
			if !strings.Contains(activationJobSection, "fetch-depth: 1") {
				t.Errorf("%s: Should use shallow clone (fetch-depth: 1)", tt.description)
			}

			// Verify persist-credentials: false
			if !strings.Contains(activationJobSection, "persist-credentials: false") {
				t.Errorf("%s: Should set persist-credentials: false", tt.description)
			}

			// Verify sparse-checkout-cone-mode: false
			if !strings.Contains(activationJobSection, "sparse-checkout-cone-mode: false") {
				t.Errorf("%s: Should set sparse-checkout-cone-mode: false", tt.description)
			}

			// Verify timestamp check step is present after checkout
			if !strings.Contains(activationJobSection, "Check workflow file timestamps") {
				t.Errorf("%s: Should contain timestamp check step", tt.description)
			}
		})
	}
}
