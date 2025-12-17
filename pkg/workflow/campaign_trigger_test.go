package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestCampaignTriggerEvents tests that workflows with both opened and labeled
// event types compile correctly. This specifically tests the fix for issue #6721
// where the campaign generator needs to respond to both opened (issue creation)
// and labeled (when GitHub applies labels from issue forms).
func TestCampaignTriggerEvents(t *testing.T) {
	tmpDir := testutil.TempDir(t, "campaign-trigger-test")
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		frontmatter string
		checkFor    string
	}{
		{
			name: "issues with opened and labeled types",
			frontmatter: `---
on:
  issues:
    types: [opened, labeled]

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot
tools:
  github:
    toolsets: [default]

if: contains(github.event.issue.labels.*.name, 'campaign')
---`,
			checkFor: "- labeled",
		},
		{
			name: "issues with labeled type only",
			frontmatter: `---
on:
  issues:
    types: [labeled]

permissions:
  contents: read
  issues: read

engine: copilot
tools:
  github:
    toolsets: [default]
---`,
			checkFor: "- labeled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := tmpDir + "/test-campaign-trigger.md"
			content := tt.frontmatter + "\n\n# Test Workflow\n\nTest campaign trigger events."
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

			// Verify the labeled type is present
			if !strings.Contains(lockContent, tt.checkFor) {
				t.Errorf("Expected lock file to contain '%s', but it doesn't", tt.checkFor)
			}

			// Verify the types are in the correct YAML structure
			if !strings.Contains(lockContent, "types:") {
				t.Error("Expected 'types:' field in lock file")
			}

			// Clean up
			os.Remove(testFile)
			os.Remove(lockFile)
		})
	}
}

// TestCampaignGeneratorWorkflow specifically tests the campaign-generator workflow
// to ensure it compiles correctly with only the labeled event type.
// This prevents the workflow from being skipped when triggered by the 'opened' event
// before labels are applied by GitHub issue forms.
func TestCampaignGeneratorWorkflow(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test compilation of the actual campaign-generator workflow
	workflowPath := "../../.github/workflows/campaign-generator.md"

	// Check if file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skip("campaign-generator.md not found, skipping test")
	}

	// Compile the workflow
	err := compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile campaign-generator workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	lockContent := string(lockBytes)

	// Verify only labeled event type is present (not opened)
	// The 'opened' event causes the workflow to skip because labels aren't applied yet
	if !strings.Contains(lockContent, "- labeled") {
		t.Error("Expected 'labeled' event type in campaign-generator lock file")
	}
	if strings.Contains(lockContent, "- opened") {
		t.Error("Should not have 'opened' event type in campaign-generator lock file (causes skip before labels are applied)")
	}

	// Verify the label condition is present
	if !strings.Contains(lockContent, "contains(github.event.issue.labels.*.name, 'campaign')") {
		t.Error("Expected label condition in campaign-generator lock file")
	}
}
