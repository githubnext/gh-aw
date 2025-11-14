package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestActivationJobTimestampCheck tests that the activation job uses GitHub API
// for timestamp checking instead of actions/checkout
func TestActivationJobTimestampCheck(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		description string
	}{
		{
			name: "basic workflow uses API for timestamp check",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: claude
---`,
			description: "Activation job should use GitHub API for timestamp check, not checkout",
		},
		{
			name: "workflow without contents permission uses API",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
engine: claude
---`,
			description: "Activation job should use API even when main job doesn't have contents permission",
		},
		{
			name: "workflow with reaction uses API for timestamp check",
			frontmatter: `---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  issues: write
engine: claude
---`,
			description: "Activation job with reaction should use API for timestamp check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "activation-timestamp-test")
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

			// Verify checkout step is NOT present
			if strings.Contains(activationJobSection, "actions/checkout@") {
				t.Errorf("%s: Activation job should NOT contain checkout step\nSection:\n%s",
					tt.description, activationJobSection)
			}

			// Verify checkout-related configuration is NOT present
			if strings.Contains(activationJobSection, "sparse-checkout:") {
				t.Errorf("%s: Should NOT use sparse-checkout (no checkout needed)", tt.description)
			}

			if strings.Contains(activationJobSection, "fetch-depth:") {
				t.Errorf("%s: Should NOT have fetch-depth (no checkout needed)", tt.description)
			}

			if strings.Contains(activationJobSection, "persist-credentials:") {
				t.Errorf("%s: Should NOT have persist-credentials (no checkout needed)", tt.description)
			}

			// Verify timestamp check step is present
			if !strings.Contains(activationJobSection, "Check workflow file timestamps") {
				t.Errorf("%s: Should contain timestamp check step", tt.description)
			}

			// Verify it uses GitHub API (github.rest.repos)
			if !strings.Contains(activationJobSection, "github.rest.repos") {
				t.Errorf("%s: Should use GitHub API (github.rest.repos) for timestamp check\nSection:\n%s",
					tt.description, activationJobSection)
			}

			// Verify it references getContent and listCommits
			if !strings.Contains(activationJobSection, "getContent") {
				t.Errorf("%s: Should call github.rest.repos.getContent", tt.description)
			}

			if !strings.Contains(activationJobSection, "listCommits") {
				t.Errorf("%s: Should call github.rest.repos.listCommits", tt.description)
			}
		})
	}
}
