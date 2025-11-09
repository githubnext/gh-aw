package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashValidationIntegration(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   string
		shouldCompile bool
		errorMsg      string
		description   string
	}{
		{
			name: "valid bash with array",
			frontmatter: `---
on: push
engine: claude
tools:
  bash:
    - "git:*"
    - "npm:*"
---`,
			shouldCompile: true,
			description:   "valid bash array should compile",
		},
		{
			name: "valid bash with true",
			frontmatter: `---
on: push
engine: claude
tools:
  bash: true
---`,
			shouldCompile: true,
			description:   "bash: true should compile",
		},
		{
			name: "valid bash with false",
			frontmatter: `---
on: push
engine: claude
tools:
  bash: false
---`,
			shouldCompile: true,
			description:   "bash: false should compile",
		},
		{
			name: "invalid bash with string",
			frontmatter: `---
on: push
engine: claude
tools:
  bash: "git:*"
---`,
			shouldCompile: false,
			errorMsg:      "bash: must be null, boolean, or array",
			description:   "bash with string value should fail",
		},
		{
			name: "invalid bash with number",
			frontmatter: `---
on: push
engine: claude
tools:
  bash: 123
---`,
			shouldCompile: false,
			errorMsg:      "bash: must be null, boolean, or array",
			description:   "bash with number value should fail",
		},
		{
			name: "invalid bash array with non-string element",
			frontmatter: `---
on: push
engine: claude
tools:
  bash:
    - "git:*"
    - 123
---`,
			shouldCompile: false,
			errorMsg:      "bash: bash tool array element at index 1 is not a string",
			description:   "bash array with non-string element should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tmpDir, err := os.MkdirTemp("", "TestBashValidationIntegration")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create a test workflow file
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			content := tt.frontmatter + "\n\n# Test Workflow\n\nThis is a test workflow.\n"
			if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			compiler.SetNoEmit(true) // Don't write .lock.yml file
			err = compiler.CompileWorkflow(workflowPath)

			if tt.shouldCompile {
				if err != nil {
					t.Errorf("Expected compilation to succeed for %s, but got error: %v", tt.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected compilation to fail for %s, but it succeeded", tt.description)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain '%s'\nGot: %s", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestCacheMemoryValidationIntegration(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   string
		shouldCompile bool
		errorMsg      string
		description   string
	}{
		{
			name: "valid cache-memory with true",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory: true
---`,
			shouldCompile: true,
			description:   "cache-memory: true should compile",
		},
		{
			name: "valid cache-memory with object",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory:
    key: custom-key
    retention-days: 7
---`,
			shouldCompile: true,
			description:   "cache-memory with valid object should compile",
		},
		{
			name: "valid cache-memory with array",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session
---`,
			shouldCompile: true,
			description:   "cache-memory with valid array should compile",
		},
		{
			name: "invalid cache-memory with string",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory: "invalid"
---`,
			shouldCompile: false,
			errorMsg:      "cache-memory: must be null, boolean, object, or array",
			description:   "cache-memory with string value should fail",
		},
		{
			name: "invalid cache-memory retention-days below range",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory:
    retention-days: 0
---`,
			shouldCompile: false,
			errorMsg:      "cache-memory: 'retention-days' must be between 1 and 90 days",
			description:   "cache-memory with retention-days below 1 should fail",
		},
		{
			name: "invalid cache-memory retention-days above range",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory:
    retention-days: 91
---`,
			shouldCompile: false,
			errorMsg:      "cache-memory: 'retention-days' must be between 1 and 90 days",
			description:   "cache-memory with retention-days above 90 should fail",
		},
		{
			name: "invalid cache-memory with unknown field",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory:
    invalid-field: value
---`,
			shouldCompile: false,
			errorMsg:      "cache-memory: unknown field 'invalid-field'",
			description:   "cache-memory with unknown field should fail",
		},
		{
			name: "invalid cache-memory empty array",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory: []
---`,
			shouldCompile: false,
			errorMsg:      "cache-memory: cache-memory array cannot be empty",
			description:   "cache-memory with empty array should fail",
		},
		{
			name: "invalid cache-memory array with duplicate IDs",
			frontmatter: `---
on: push
engine: claude
tools:
  cache-memory:
    - id: duplicate
      key: key1
    - id: duplicate
      key: key2
---`,
			shouldCompile: false,
			errorMsg:      "cache-memory: duplicate cache-memory ID 'duplicate'",
			description:   "cache-memory with duplicate IDs should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tmpDir, err := os.MkdirTemp("", "TestCacheMemoryValidationIntegration")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create a test workflow file
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			content := tt.frontmatter + "\n\n# Test Workflow\n\nThis is a test workflow.\n"
			if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			compiler.SetNoEmit(true) // Don't write .lock.yml file
			err = compiler.CompileWorkflow(workflowPath)

			if tt.shouldCompile {
				if err != nil {
					t.Errorf("Expected compilation to succeed for %s, but got error: %v", tt.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected compilation to fail for %s, but it succeeded", tt.description)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain '%s'\nGot: %s", tt.errorMsg, err.Error())
				}
			}
		})
	}
}
