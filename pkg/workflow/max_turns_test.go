package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaxTurnsCompilation(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedMaxTurns string
		shouldInclude    bool
	}{
		{
			name: "workflow with max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
max-turns: 3
tools:
  github:
    allowed: [get_issue]
---

# Test Max Turns

This workflow tests the max-turns feature.`,
			expectedMaxTurns: "max_turns: 3",
			shouldInclude:    true,
		},
		{
			name: "workflow without max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [get_issue]
---

# Test Without Max Turns

This workflow should not include max-turns.`,
			expectedMaxTurns: "",
			shouldInclude:    false,
		},
		{
			name: "workflow with max-turns and timeout",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
max-turns: 10
timeout_minutes: 15
tools:
  github:
    allowed: [get_issue]
---

# Test Max Turns and Timeout

This workflow tests max-turns with timeout.`,
			expectedMaxTurns: "max_turns: 10",
			shouldInclude:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir, err := os.MkdirTemp("", "max-turns-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create the test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "")
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			if tt.shouldInclude {
				// Verify max_turns is included in the generated workflow
				if !strings.Contains(lockContentStr, tt.expectedMaxTurns) {
					t.Errorf("Expected max_turns to be included in generated workflow. Expected: %s\nActual content:\n%s", tt.expectedMaxTurns, lockContentStr)
				}

				// Verify it's in the correct context (under the Claude action inputs)
				if !strings.Contains(lockContentStr, "anthropics/claude-code-base-action") {
					t.Error("Expected to find Claude action in generated workflow")
				}

				// Look for max_turns in the inputs section
				lines := strings.Split(lockContentStr, "\n")
				foundAction := false
				foundMaxTurns := false
				for i, line := range lines {
					if strings.Contains(line, "anthropics/claude-code-base-action") {
						foundAction = true
					}
					if foundAction && strings.Contains(line, "with:") {
						// Look for max_turns in the subsequent lines
						for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], "          "); j++ {
							if strings.Contains(lines[j], "max_turns:") {
								foundMaxTurns = true
								break
							}
						}
						break
					}
				}

				if !foundMaxTurns {
					t.Error("Expected to find max_turns in the action inputs section")
				}
			} else {
				// Verify max_turns is NOT included when not specified
				if strings.Contains(lockContentStr, "max_turns:") {
					t.Error("Expected max_turns NOT to be included when not specified in frontmatter")
				}
			}
		})
	}
}

func TestMaxTurnsValidation(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid integer max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
max-turns: 5
---

# Valid Max Turns`,
			expectError: false,
		},
		{
			name: "invalid string max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
max-turns: "invalid"
---

# Invalid Max Turns`,
			expectError: true,
		},
		{
			name: "zero max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
max-turns: 0
---

# Zero Max Turns`,
			expectError: false, // Zero should be valid (might mean unlimited)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir, err := os.MkdirTemp("", "max-turns-validation-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create the test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "")
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			}
		})
	}
}
