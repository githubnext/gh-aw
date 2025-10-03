package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestLabelFilter tests the label name filter functionality for labeled/unlabeled events
func TestLabelFilter(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "label-filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name         string
		frontmatter  string
		expectedIf   string // Expected if condition in the generated lock file
		shouldHaveIf bool   // Whether an if condition should be present
	}{
		{
			name: "issues with labeled and single label name",
			frontmatter: `---
on:
  issues:
    types: [opened, labeled]
    names: bug

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "github.event.label.name == 'bug'",
			shouldHaveIf: true,
		},
		{
			name: "issues with labeled and multiple label names",
			frontmatter: `---
on:
  issues:
    types: [labeled]
    names: [bug, enhancement, feature]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "github.event.label.name == 'bug'",
			shouldHaveIf: true,
		},
		{
			name: "issues with unlabeled and label names",
			frontmatter: `---
on:
  issues:
    types: [unlabeled]
    names: [wontfix, duplicate]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "github.event.label.name == 'wontfix'",
			shouldHaveIf: true,
		},
		{
			name: "issues with both labeled and unlabeled",
			frontmatter: `---
on:
  issues:
    types: [labeled, unlabeled]
    names: [priority, urgent]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "github.event.label.name == 'priority'",
			shouldHaveIf: true,
		},
		{
			name: "pull_request with labeled and label names",
			frontmatter: `---
on:
  pull_request:
    types: [opened, labeled]
    names: ready-for-review

permissions:
  contents: read
  pull-requests: write

tools:
  github:
    allowed: [get_pull_request]
---`,
			expectedIf:   "github.event.label.name == 'ready-for-review'",
			shouldHaveIf: true,
		},
		{
			name: "issues without labeled/unlabeled types",
			frontmatter: `---
on:
  issues:
    types: [opened, edited]
    names: bug

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "",
			shouldHaveIf: false,
		},
		{
			name: "issues with labeled but no names field",
			frontmatter: `---
on:
  issues:
    types: [labeled]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "",
			shouldHaveIf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := tmpDir + "/test-label-filter.md"
			content := tt.frontmatter + "\n\n# Test Workflow\n\nTest label filtering."
			if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			lockBytes, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatal(err)
			}
			lockContent := string(lockBytes)

			// Check if the condition is present
			if tt.shouldHaveIf {
				if !strings.Contains(lockContent, "if:") {
					t.Errorf("Expected 'if:' condition to be present in generated workflow")
				}

				// Check if the expected condition fragment is present
				if tt.expectedIf != "" && !strings.Contains(lockContent, tt.expectedIf) {
					t.Errorf("Expected condition to contain '%s', got:\n%s", tt.expectedIf, lockContent)
				}
			} else {
				// For cases where we don't expect an if condition from label filtering
				// (but might have other conditions), we just verify compilation succeeded
			}

			// Clean up test file
			os.Remove(testFile)
			os.Remove(lockFile)
		})
	}
}

// TestLabelFilterCommentedOut tests that the names field is commented out in the final YAML
func TestLabelFilterCommentedOut(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "label-filter-comment-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	frontmatter := `---
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, enhancement]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`

	testFile := tmpDir + "/test-comment.md"
	content := frontmatter + "\n\n# Test Workflow\n\nTest comment."
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	lockContent := string(lockBytes)

	// Check that the names field is commented out
	if !strings.Contains(lockContent, "# names:") || !strings.Contains(lockContent, "Label filtering applied") {
		t.Error("Expected 'names:' field to be commented out with 'Label filtering applied' note")
	}

	// Check that the names array items are commented out
	if !strings.Contains(lockContent, "# - bug") || !strings.Contains(lockContent, "# - enhancement") {
		t.Error("Expected names array items to be commented out")
	}
}
