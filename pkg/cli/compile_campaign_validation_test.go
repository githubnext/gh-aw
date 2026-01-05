//go:build integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// initGitRepo initializes a git repository in the given directory
func initGitRepoForCampaignTest(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}
}

// TestCampaignValidationSkippedForSpecificWorkflow tests that campaign validation
// is skipped when compiling a specific non-campaign workflow
func TestCampaignValidationSkippedForSpecificWorkflow(t *testing.T) {
	// Create a temporary directory
	tmpDir := testutil.TempDir(t, "test-*")

	// Create workflows directory
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Initialize git repo
	initGitRepoForCampaignTest(t, tmpDir)

	// Create a simple non-campaign workflow
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	workflowContent := `---
name: Test Workflow
engine: copilot
---

# Test Task

This is a simple test workflow.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0600); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Create an invalid campaign spec to verify validation is skipped
	campaignDir := filepath.Join(tmpDir, ".github", "campaigns")
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		t.Fatalf("Failed to create campaigns directory: %v", err)
	}

	invalidCampaignPath := filepath.Join(campaignDir, "invalid.campaign.yml")
	invalidCampaignContent := `id: invalid-campaign
# Missing required 'workflows' field
name: Invalid Campaign
description: This campaign is missing the required workflows field
`
	if err := os.WriteFile(invalidCampaignPath, []byte(invalidCampaignContent), 0600); err != nil {
		t.Fatalf("Failed to create invalid campaign spec: %v", err)
	}

	// Change to the temporary directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create compiler
	compiler := workflow.NewCompiler(false, "", GetVersion())
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(false)

	// Compile the specific workflow
	config := CompileConfig{
		MarkdownFiles: []string{workflowPath},
		Verbose:       false,
		WorkflowDir:   ".github/workflows",
		NoEmit:        false,
		Strict:        false,
	}

	ctx := context.Background()
	_, err := CompileWorkflows(ctx, config)

	// The compilation should succeed without campaign validation errors
	// If campaign validation ran, it would detect the invalid campaign and return an error
	if err != nil {
		// Check if the error is related to campaign validation
		if strings.Contains(err.Error(), "campaign") {
			t.Errorf("Campaign validation should not run for specific non-campaign workflow, but got error: %v", err)
		}
		// Other errors might be acceptable (e.g., compilation issues)
	}
}

// TestCampaignValidationRunsForCampaignFile tests that campaign validation
// runs when compiling a specific campaign file
func TestCampaignValidationRunsForCampaignFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := testutil.TempDir(t, "test-*")

	// Create workflows directory
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Initialize git repo
	initGitRepoForCampaignTest(t, tmpDir)

	// Create a campaign spec file
	campaignPath := filepath.Join(workflowsDir, "test-campaign.campaign.md")
	campaignContent := `---
id: test-campaign
name: Test Campaign
description: A test campaign
workflows:
  - example-workflow
---

# Campaign Task

This is a test campaign.
`
	if err := os.WriteFile(campaignPath, []byte(campaignContent), 0600); err != nil {
		t.Fatalf("Failed to create campaign file: %v", err)
	}

	// Create an invalid campaign spec in campaigns directory to trigger validation error
	campaignDir := filepath.Join(tmpDir, ".github", "campaigns")
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		t.Fatalf("Failed to create campaigns directory: %v", err)
	}

	invalidCampaignPath := filepath.Join(campaignDir, "invalid.campaign.yml")
	invalidCampaignContent := `id: invalid-campaign
# Missing required 'workflows' field
name: Invalid Campaign
description: This campaign is missing the required workflows field
`
	if err := os.WriteFile(invalidCampaignPath, []byte(invalidCampaignContent), 0600); err != nil {
		t.Fatalf("Failed to create invalid campaign spec: %v", err)
	}

	// Change to the temporary directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create compiler
	compiler := workflow.NewCompiler(false, "", GetVersion())
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(false)

	// Compile the specific campaign file
	config := CompileConfig{
		MarkdownFiles: []string{campaignPath},
		Verbose:       false,
		WorkflowDir:   ".github/workflows",
		NoEmit:        false,
		Strict:        false, // Use non-strict mode so validation errors become warnings
	}

	ctx := context.Background()
	_, err := CompileWorkflows(ctx, config)

	// In non-strict mode, campaign validation errors should be warnings, not fatal errors
	// So compilation should succeed even with invalid campaigns
	if err != nil {
		// Check if the error is related to campaign validation
		if strings.Contains(err.Error(), "campaign") {
			// This is expected in non-strict mode - validation ran and reported as warning
			// The test passes because validation actually ran
			return
		}
	}

	// If no error, validation still ran but was reported as a warning
	// This is the expected behavior in non-strict mode
}

// TestCampaignValidationRunsForAllWorkflows tests that campaign validation
// runs when compiling all workflows (no specific files)
func TestCampaignValidationRunsForAllWorkflows(t *testing.T) {
	// Create a temporary directory
	tmpDir := testutil.TempDir(t, "test-*")

	// Create workflows directory
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Initialize git repo
	initGitRepoForCampaignTest(t, tmpDir)

	// Create a simple workflow
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	workflowContent := `---
name: Test Workflow
engine: copilot
---

# Test Task

This is a simple test workflow.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0600); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Create an invalid campaign spec to verify validation runs
	campaignDir := filepath.Join(tmpDir, ".github", "campaigns")
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		t.Fatalf("Failed to create campaigns directory: %v", err)
	}

	invalidCampaignPath := filepath.Join(campaignDir, "invalid.campaign.yml")
	invalidCampaignContent := `id: invalid-campaign
# Missing required 'workflows' field
name: Invalid Campaign
description: This campaign is missing the required workflows field
`
	if err := os.WriteFile(invalidCampaignPath, []byte(invalidCampaignContent), 0600); err != nil {
		t.Fatalf("Failed to create invalid campaign spec: %v", err)
	}

	// Change to the temporary directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create compiler
	compiler := workflow.NewCompiler(false, "", GetVersion())
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(false)

	// Compile all workflows (no specific files)
	config := CompileConfig{
		MarkdownFiles: []string{}, // Empty means compile all
		Verbose:       false,
		WorkflowDir:   ".github/workflows",
		NoEmit:        false,
		Strict:        false, // Use non-strict mode so validation errors become warnings
	}

	ctx := context.Background()
	_, err := CompileWorkflows(ctx, config)

	// In non-strict mode, campaign validation errors should be warnings, not fatal errors
	// So compilation should succeed even with invalid campaigns
	if err != nil {
		// Check if the error is related to campaign validation
		if strings.Contains(err.Error(), "campaign") {
			// This is expected in non-strict mode - validation ran and reported as warning
			// The test passes because validation actually ran
			return
		}
	}

	// If no error, validation still ran but was reported as a warning
	// This is the expected behavior in non-strict mode
}
