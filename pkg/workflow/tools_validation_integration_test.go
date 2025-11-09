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
			name: "valid bash with nil/null",
			frontmatter: `---
on: push
engine: claude
tools:
  bash:
---`,
			shouldCompile: true,
			description:   "bash with nil/null should compile",
		},
		// Note: Type errors (string, number, non-string array elements) are caught by JSON schema
		// validation before our custom validation runs. Those tests are in the unit tests.
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
			errorMsg:      "duplicate cache-memory ID 'duplicate'",
			description:   "cache-memory with duplicate IDs should fail",
		},
		// Note: Type errors (string values, wrong types for fields, retention-days range, 
		// unknown fields, empty arrays) are caught by JSON schema validation before our 
		// custom validation runs. Those tests are in the unit tests.
		//
		// Our validation catches semantic errors like duplicate IDs that the schema can't validate.
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
