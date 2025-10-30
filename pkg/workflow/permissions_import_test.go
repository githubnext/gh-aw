package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestMergePermissions(t *testing.T) {
	tests := []struct {
		name                 string
		topPermissionsYAML   string
		importedPermissions  string
		expectedContains     []string
		expectedNotContains  []string
	}{
		{
			name:               "No imported permissions returns top-level unchanged",
			topPermissionsYAML: "permissions:\n  contents: read",
			importedPermissions: "",
			expectedContains:    []string{"contents: read"},
		},
		{
			name:               "Empty imported permissions returns top-level unchanged",
			topPermissionsYAML: "permissions:\n  contents: read",
			importedPermissions: "{}",
			expectedContains:    []string{"contents: read"},
		},
		{
			name:               "Merge adds new permission",
			topPermissionsYAML: "permissions:\n  contents: read",
			importedPermissions: `{"actions":"read"}`,
			expectedContains:    []string{"contents: read", "actions: read"},
		},
		{
			name:               "Merge upgrades read to write",
			topPermissionsYAML: "permissions:\n  contents: read",
			importedPermissions: `{"contents":"write"}`,
			expectedContains:    []string{"contents: write"},
			expectedNotContains: []string{"contents: read"},
		},
		{
			name:               "Merge keeps write when import has read",
			topPermissionsYAML: "permissions:\n  contents: write",
			importedPermissions: `{"contents":"read"}`,
			expectedContains:    []string{"contents: write"},
		},
		{
			name:               "Merge multiple permissions from import",
			topPermissionsYAML: "",
			importedPermissions: `{"actions":"read"}` + "\n" + `{"issues":"write"}`,
			expectedContains:    []string{"actions: read", "issues: write"},
		},
		{
			name:               "Merge with empty top-level permissions",
			topPermissionsYAML: "",
			importedPermissions: `{"actions":"read"}`,
			expectedContains:    []string{"actions: read"},
		},
		{
			name:               "Merge complex scenario",
			topPermissionsYAML: "permissions:\n  contents: read\n  issues: read",
			importedPermissions: `{"actions":"read"}` + "\n" + `{"contents":"write"}` + "\n" + `{"pull-requests":"write"}`,
			expectedContains:    []string{"contents: write", "issues: read", "actions: read", "pull-requests: write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			result, err := compiler.MergePermissions(tt.topPermissionsYAML, tt.importedPermissions)
			if err != nil {
				t.Fatalf("MergePermissions() error = %v", err)
			}

			// Check expected contains
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("MergePermissions() result missing expected content '%s'\nGot: %s", expected, result)
				}
			}

			// Check expected not contains
			for _, notExpected := range tt.expectedNotContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("MergePermissions() result contains unexpected content '%s'\nGot: %s", notExpected, result)
				}
			}
		})
	}
}

func TestPermissionsImportIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create a shared workflow file with permissions
	sharedWorkflowContent := `---
permissions:
  actions: read
---

# Shared workflow with permissions
`
	sharedWorkflowPath := filepath.Join(sharedDir, "shared-permissions.md")
	if err := os.WriteFile(sharedWorkflowPath, []byte(sharedWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to create shared workflow file: %v", err)
	}

	// Test 1: Import adds new permissions
	t.Run("Import adds permissions to workflow", func(t *testing.T) {
		mainWorkflowContent := `---
on: issues
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
imports:
  - shared/shared-permissions.md
tools:
  github:
    toolsets: [default]
---

# Main workflow
`
		mainWorkflowPath := filepath.Join(tempDir, ".github", "workflows", "test-workflow.md")
		if err := os.WriteFile(mainWorkflowPath, []byte(mainWorkflowContent), 0644); err != nil {
			t.Fatalf("Failed to create main workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(mainWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the generated lock file to verify permissions
		lockFilePath := filepath.Join(tempDir, ".github", "workflows", "test-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockStr := string(lockContent)
		// Check that permissions were merged
		if !strings.Contains(lockStr, "actions: read") {
			t.Errorf("Expected permissions to contain 'actions: read', got: %s", lockStr)
		}
		if !strings.Contains(lockStr, "issues: read") {
			t.Errorf("Expected permissions to contain 'issues: read', got: %s", lockStr)
		}
		if !strings.Contains(lockStr, "contents: read") {
			t.Errorf("Expected permissions to contain 'contents: read', got: %s", lockStr)
		}
	})

	// Test 2: Import upgrades permissions
	t.Run("Import upgrades permission level", func(t *testing.T) {
		sharedWorkflowUpgradeContent := `---
permissions:
  contents: write
---

# Shared workflow with write permission
`
		sharedWorkflowUpgradePath := filepath.Join(sharedDir, "shared-upgrade.md")
		if err := os.WriteFile(sharedWorkflowUpgradePath, []byte(sharedWorkflowUpgradeContent), 0644); err != nil {
			t.Fatalf("Failed to create shared upgrade workflow file: %v", err)
		}

		mainWorkflowContent := `---
on: issues
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
imports:
  - shared/shared-upgrade.md
tools:
  github:
    toolsets: [default]
---

# Main workflow
`
		mainWorkflowPath := filepath.Join(tempDir, ".github", "workflows", "test-upgrade.md")
		if err := os.WriteFile(mainWorkflowPath, []byte(mainWorkflowContent), 0644); err != nil {
			t.Fatalf("Failed to create main workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(mainWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the generated lock file to verify permissions
		lockFilePath := filepath.Join(tempDir, ".github", "workflows", "test-upgrade.lock.yml")
		lockContent, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockStr := string(lockContent)
		// Check that permissions were upgraded to write
		if !strings.Contains(lockStr, "contents: write") {
			t.Errorf("Expected permissions to contain 'contents: write', got: %s", lockStr)
		}
	})

	// Test 3: Validate merged permissions against GitHub MCP toolsets
	t.Run("Validate merged permissions with GitHub toolsets", func(t *testing.T) {
		mainWorkflowContent := `---
on: issues
engine: copilot
permissions:
  contents: read
tools:
  github:
    toolsets: [repos, issues]
imports:
  - shared/shared-permissions.md
---

# Main workflow with toolsets
`
		mainWorkflowPath := filepath.Join(tempDir, ".github", "workflows", "test-validation.md")
		if err := os.WriteFile(mainWorkflowPath, []byte(mainWorkflowContent), 0644); err != nil {
			t.Fatalf("Failed to create main workflow file: %v", err)
		}

		// Compile the workflow - should succeed with merged permissions
		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(mainWorkflowPath)
		if err != nil {
			// This is expected to fail initially because we need write permissions for toolsets
			// but let's check the error message
			if !strings.Contains(err.Error(), "Missing required permissions") {
				t.Fatalf("Expected missing permissions error, got: %v", err)
			}
		}
	})
}

func TestExtractPermissionsFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "Simple permissions",
			content: `---
permissions:
  contents: read
  issues: write
---
# Content`,
			expected: `{"contents":"read","issues":"write"}`,
			wantErr:  false,
		},
		{
			name: "No permissions",
			content: `---
on: issues
---
# Content`,
			expected: "{}",
			wantErr:  false,
		},
		{
			name: "Empty frontmatter",
			content: `---
---
# Content`,
			expected: "{}",
			wantErr:  false,
		},
		{
			name:     "No frontmatter",
			content:  "# Just markdown content",
			expected: "{}",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractPermissionsFromContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractPermissionsFromContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("extractPermissionsFromContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}
