//go:build integration

package workflow

import (
	"os"
	"os/exec"
	"testing"
)

// TestRepositoryFeaturesValidationIntegration tests the repository features validation
// with actual GitHub API calls. This test requires:
// 1. Running in a git repository with GitHub remote
// 2. GitHub CLI (gh) authenticated
func TestRepositoryFeaturesValidationIntegration(t *testing.T) {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not available, skipping integration test")
	}

	// Check if gh CLI is authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		t.Skip("gh CLI not authenticated, skipping integration test")
	}

	// Get current repository
	repo, err := getCurrentRepository()
	if err != nil {
		t.Skip("Could not determine current repository, skipping integration test")
	}

	t.Logf("Testing with repository: %s", repo)

	// Test checking discussions
	t.Run("check_discussions", func(t *testing.T) {
		hasDiscussions, err := checkRepositoryHasDiscussions(repo)
		if err != nil {
			t.Errorf("Failed to check discussions: %v", err)
		}
		t.Logf("Repository %s has discussions enabled: %v", repo, hasDiscussions)
	})

	// Test checking issues
	t.Run("check_issues", func(t *testing.T) {
		hasIssues, err := checkRepositoryHasIssues(repo)
		if err != nil {
			t.Errorf("Failed to check issues: %v", err)
		}
		t.Logf("Repository %s has issues enabled: %v", repo, hasIssues)

		// Issues should be enabled for githubnext/gh-aw
		if repo == "githubnext/gh-aw" && !hasIssues {
			t.Error("Expected githubnext/gh-aw to have issues enabled")
		}
	})

	// Test full validation with discussions
	t.Run("validate_with_discussions", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{},
			},
		}

		compiler := NewCompiler(true, "", "test")
		err := compiler.validateRepositoryFeatures(workflowData)

		hasDiscussions, checkErr := checkRepositoryHasDiscussions(repo)
		if checkErr != nil {
			t.Logf("Could not verify discussions status: %v", checkErr)
			return
		}

		if hasDiscussions && err != nil {
			t.Errorf("Expected no error when discussions are enabled, got: %v", err)
		} else if !hasDiscussions && err == nil {
			t.Error("Expected error when discussions are disabled, got none")
		}
	})

	// Test full validation with issues
	t.Run("validate_with_issues", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
		}

		compiler := NewCompiler(true, "", "test")
		err := compiler.validateRepositoryFeatures(workflowData)

		hasIssues, checkErr := checkRepositoryHasIssues(repo)
		if checkErr != nil {
			t.Logf("Could not verify issues status: %v", checkErr)
			return
		}

		if hasIssues && err != nil {
			t.Errorf("Expected no error when issues are enabled, got: %v", err)
		} else if !hasIssues && err == nil {
			t.Error("Expected error when issues are disabled, got none")
		}
	})
}

// TestCompileWorkflowWithRepositoryFeatureValidation tests compiling a workflow
// that requires repository features
func TestCompileWorkflowWithRepositoryFeatureValidation(t *testing.T) {
	// Check if gh CLI is available and authenticated
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not available, skipping integration test")
	}

	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		t.Skip("gh CLI not authenticated, skipping integration test")
	}

	// Get current repository
	repo, err := getCurrentRepository()
	if err != nil {
		t.Skip("Could not determine current repository, skipping integration test")
	}

	// Create a temporary workflow with create-discussion
	tempDir := t.TempDir()
	workflowPath := tempDir + "/test-discussion.md"

	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
safe-outputs:
  create-discussion:
    category: "General"
---

# Test Discussion Workflow

Test workflow for discussions validation.
`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Try to compile the workflow
	compiler := NewCompiler(true, "", "test")
	compiler.SetNoEmit(true) // Don't write lock file

	err = compiler.CompileWorkflow(workflowPath)

	// Check if discussions are enabled
	hasDiscussions, checkErr := checkRepositoryHasDiscussions(repo)
	if checkErr != nil {
		t.Logf("Could not verify discussions status: %v", checkErr)
		t.Logf("Compilation result: %v", err)
		return
	}

	if hasDiscussions {
		if err != nil {
			t.Errorf("Expected compilation to succeed when discussions are enabled, got error: %v", err)
		}
	} else {
		if err == nil {
			t.Error("Expected compilation to fail when discussions are disabled, but it succeeded")
		} else {
			t.Logf("Compilation correctly failed: %v", err)
		}
	}
}
