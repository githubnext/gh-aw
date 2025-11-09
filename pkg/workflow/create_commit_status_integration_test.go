package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCommitStatusIntegration(t *testing.T) {
	tests := []struct {
		name                string
		workflowContent     string
		expectedJobName     string
		expectError         bool
		expectedPermissions string
	}{
		{
			name: "basic create-commit-status workflow",
			workflowContent: `---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
safe-outputs:
  create-commit-status:
---

# PR Status Check

Create a commit status indicating whether the PR looks good.
`,
			expectedJobName:     "create_commit_status",
			expectError:         false,
			expectedPermissions: "statuses: write",
		},
		{
			name: "create-commit-status with custom context",
			workflowContent: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-commit-status:
    context: "ci/custom-check"
---

# Custom Status Check

Perform a custom validation and create a commit status.
`,
			expectedJobName:     "create_commit_status",
			expectError:         false,
			expectedPermissions: "statuses: write",
		},
		{
			name: "create-commit-status with custom github-token",
			workflowContent: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-commit-status:
    github-token: "${{ secrets.CUSTOM_TOKEN }}"
---

# Status Check with Custom Token

Create a commit status using a custom GitHub token.
`,
			expectedJobName:     "create_commit_status",
			expectError:         false,
			expectedPermissions: "statuses: write",
		},
		{
			name: "create-commit-status with allowed-domains",
			workflowContent: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-commit-status:
    allowed-domains: ["example.com", "*.trusted.org"]
---

# Status Check with Allowed Domains

Create a commit status with validated target URLs.
`,
			expectedJobName:     "create_commit_status",
			expectError:         false,
			expectedPermissions: "statuses: write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir, err := os.MkdirTemp("", "commit-status-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Read the compiled output
			outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
			compiledContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read compiled output: %v", err)
			}

			yamlContent := string(compiledContent)

			// Verify the job exists in the YAML
			if !strings.Contains(yamlContent, tt.expectedJobName+":") {
				t.Errorf("Expected job '%s' not found in YAML", tt.expectedJobName)
			}

			// Verify permissions
			if !strings.Contains(yamlContent, tt.expectedPermissions) {
				t.Errorf("Expected permission '%s' not found in YAML", tt.expectedPermissions)
			}

			// Verify the job has the create_commit_status step
			if !strings.Contains(yamlContent, "Create Commit Status") {
				t.Error("Expected 'Create Commit Status' step not found in YAML")
			}

			// Verify outputs are defined
			if !strings.Contains(yamlContent, "status_created") {
				t.Error("Expected 'status_created' output not found in YAML")
			}

			if !strings.Contains(yamlContent, "status_url") {
				t.Error("Expected 'status_url' output not found in YAML")
			}

			// Verify the MCP tool is registered in the config
			if !strings.Contains(yamlContent, "create_commit_status") {
				t.Error("Expected 'create_commit_status' tool not found in config")
			}

			// If this is the allowed-domains test, verify the env var is set
			if strings.Contains(tt.name, "allowed-domains") {
				if !strings.Contains(yamlContent, "GH_AW_COMMIT_STATUS_ALLOWED_DOMAINS") {
					t.Error("Expected 'GH_AW_COMMIT_STATUS_ALLOWED_DOMAINS' environment variable not found")
				}
				if !strings.Contains(yamlContent, "example.com") || !strings.Contains(yamlContent, "*.trusted.org") {
					t.Error("Expected allowed domains not found in environment variable")
				}
			}
		})
	}
}

func TestCreateCommitStatusPromptGeneration(t *testing.T) {
	workflowContent := `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-commit-status:
---

# Status Check

Create a commit status.
`

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "commit-status-prompt-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	yamlContent := string(compiledContent)

	// Verify prompt includes instructions about create-commit-status
	if !strings.Contains(yamlContent, "Creating Commit Status") {
		t.Error("Expected prompt to mention 'Creating Commit Status'")
	}

	if !strings.Contains(yamlContent, "create-commit-status tool") {
		t.Error("Expected prompt to mention 'create-commit-status tool'")
	}
}

func TestCreateCommitStatusWithOtherSafeOutputs(t *testing.T) {
	workflowContent := `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-commit-status:
  add-comment:
---

# Combined Status and Comment

Create a commit status and add a comment to the PR.
`

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "commit-status-combined-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	yamlContent := string(compiledContent)

	// Verify both jobs exist
	if !strings.Contains(yamlContent, "create_commit_status:") {
		t.Error("Expected 'create_commit_status' job not found")
	}

	if !strings.Contains(yamlContent, "add_comment:") {
		t.Error("Expected 'add_comment' job not found")
	}

	// Verify prompt mentions both
	if !strings.Contains(yamlContent, "Creating Commit Status") {
		t.Error("Expected prompt to mention 'Creating Commit Status'")
	}

	// Note: "Adding Comment" appears as "Comment" in the capabilities section
	if !strings.Contains(yamlContent, "Comment") {
		t.Error("Expected prompt to mention commenting capability")
	}
}
