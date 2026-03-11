//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestBotsFieldExtraction tests the extraction of the bots field from frontmatter
func TestBotsFieldExtraction(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-bots-test")

	compiler := NewCompiler()

	tests := []struct {
		name         string
		frontmatter  string
		filename     string
		expectedBots []string
	}{
		{
			name: "workflow with bots array",
			frontmatter: `---
on:
  issues:
    types: [opened]
  bots: ["dependabot[bot]", "renovate[bot]"]
---

# Test Workflow
Test workflow content.`,
			filename:     "bots-array.md",
			expectedBots: []string{"dependabot[bot]", "renovate[bot]"},
		},
		{
			name: "workflow with single bot",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
  bots: ["github-actions[bot]"]
---

# Test Workflow
Test workflow content.`,
			filename:     "single-bot.md",
			expectedBots: []string{"github-actions[bot]"},
		},
		{
			name: "workflow without bots field",
			frontmatter: `---
on:
  push:
    branches: [main]
---

# Test Workflow
Test workflow content.`,
			filename:     "no-bots.md",
			expectedBots: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write the workflow file
			workflowPath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(workflowPath, []byte(tt.frontmatter), 0644)
			if err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Parse the workflow
			workflowData, err := compiler.ParseWorkflowFile(workflowPath)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Check the extracted bots
			if len(workflowData.Bots) != len(tt.expectedBots) {
				t.Errorf("Expected %d bots, got %d", len(tt.expectedBots), len(workflowData.Bots))
			}

			for i, expectedBot := range tt.expectedBots {
				if i >= len(workflowData.Bots) {
					t.Errorf("Expected bot '%s' at index %d, but only got %d bots", expectedBot, i, len(workflowData.Bots))
					continue
				}
				if workflowData.Bots[i] != expectedBot {
					t.Errorf("Expected bot '%s' at index %d, got '%s'", expectedBot, i, workflowData.Bots[i])
				}
			}
		})
	}
}

// TestBotsEnvironmentVariableGeneration tests that bots are passed via environment variable
func TestBotsEnvironmentVariableGeneration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-bots-env-test")

	compiler := NewCompiler()

	frontmatter := `---
on:
  issues:
    types: [opened]
  roles: [triage]
  bots: ["dependabot[bot]", "renovate[bot]"]
---

# Test Workflow with Bots
Test workflow content.`

	workflowPath := filepath.Join(tmpDir, "workflow-with-bots.md")
	err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	outputPath := filepath.Join(tmpDir, "workflow-with-bots.lock.yml")
	compiledContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledContent)

	// Check that the bots environment variable is set
	if !strings.Contains(compiledStr, "GH_AW_ALLOWED_BOTS: dependabot[bot],renovate[bot]") {
		t.Errorf("Expected compiled workflow to contain GH_AW_ALLOWED_BOTS environment variable")
	}

	// Also check that roles are still present
	if !strings.Contains(compiledStr, "GH_AW_REQUIRED_ROLES: triage") {
		t.Errorf("Expected compiled workflow to contain GH_AW_REQUIRED_ROLES environment variable")
	}
}

// TestBotsWithDefaultRoles tests that bots work with default roles
func TestBotsWithDefaultRoles(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-bots-default-roles-test")

	compiler := NewCompiler()

	frontmatter := `---
on:
  pull_request:
    types: [opened]
  bots: ["dependabot[bot]"]
---

# Test Workflow
Test workflow content with bot and default roles.`

	workflowPath := filepath.Join(tmpDir, "workflow-bots-default-roles.md")
	err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	outputPath := filepath.Join(tmpDir, "workflow-bots-default-roles.lock.yml")
	compiledContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledContent)

	// Check that default roles are present (admin, maintainer, write)
	if !strings.Contains(compiledStr, "GH_AW_REQUIRED_ROLES: admin,maintainer,write") {
		t.Errorf("Expected compiled workflow to contain default GH_AW_REQUIRED_ROLES")
	}

	// Check that bots environment variable is set
	if !strings.Contains(compiledStr, "GH_AW_ALLOWED_BOTS: dependabot[bot]") {
		t.Errorf("Expected compiled workflow to contain GH_AW_ALLOWED_BOTS environment variable")
	}
}

// TestWorkflowRunAutoBot tests that github-actions[bot] is automatically added to bots
// when the workflow has a workflow_run trigger, fixing the pre_activation role check failure.
func TestWorkflowRunAutoBot(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-run-auto-bot-test")

	compiler := NewCompiler()

	t.Run("workflow_run_adds_github_actions_bot", func(t *testing.T) {
		frontmatter := `---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]
engine: copilot
---

# Test Workflow
Test workflow content.`

		workflowPath := filepath.Join(tmpDir, "workflow-run.md")
		err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Parse the workflow
		workflowData, err := compiler.ParseWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to parse workflow: %v", err)
		}

		// github-actions[bot] should be automatically added
		if !slices.Contains(workflowData.Bots, "github-actions[bot]") {
			t.Errorf("Expected github-actions[bot] to be automatically added to bots for workflow_run trigger, got: %v", workflowData.Bots)
		}
	})

	t.Run("workflow_run_compiled_includes_github_actions_bot_env_var", func(t *testing.T) {
		frontmatter := `---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]
engine: copilot
---

# Test Workflow
Test workflow content.`

		workflowPath := filepath.Join(tmpDir, "workflow-run-compile.md")
		err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		err = compiler.CompileWorkflow(workflowPath)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		outputPath := filepath.Join(tmpDir, "workflow-run-compile.lock.yml")
		compiledContent, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}

		compiledStr := string(compiledContent)

		// The compiled workflow should include github-actions[bot] in GH_AW_ALLOWED_BOTS
		if !strings.Contains(compiledStr, "GH_AW_ALLOWED_BOTS: github-actions[bot]") {
			t.Errorf("Expected compiled workflow to include github-actions[bot] in GH_AW_ALLOWED_BOTS, got content:\n%s", compiledStr)
		}
	})

	t.Run("workflow_run_with_existing_bots_does_not_duplicate", func(t *testing.T) {
		frontmatter := `---
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]
  bots: ["github-actions[bot]", "dependabot[bot]"]
engine: copilot
---

# Test Workflow
Test workflow content.`

		workflowPath := filepath.Join(tmpDir, "workflow-run-existing-bots.md")
		err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		workflowData, err := compiler.ParseWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to parse workflow: %v", err)
		}

		// Verify github-actions[bot] is present and no duplicate was added
		if !slices.Contains(workflowData.Bots, "github-actions[bot]") {
			t.Errorf("Expected github-actions[bot] to be present in bots: %v", workflowData.Bots)
		}
		// Total bots should be exactly 2 (no duplicate)
		if len(workflowData.Bots) != 2 {
			t.Errorf("Expected exactly 2 bots (no duplicate), got %d: %v", len(workflowData.Bots), workflowData.Bots)
		}
	})

	t.Run("non_workflow_run_does_not_add_github_actions_bot", func(t *testing.T) {
		frontmatter := `---
on:
  issues:
    types: [opened]
engine: copilot
---

# Test Workflow
Test workflow content.`

		workflowPath := filepath.Join(tmpDir, "not-workflow-run.md")
		err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		workflowData, err := compiler.ParseWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to parse workflow: %v", err)
		}

		// github-actions[bot] should NOT be added for non-workflow_run triggers
		if slices.Contains(workflowData.Bots, "github-actions[bot]") {
			t.Errorf("Expected github-actions[bot] NOT to be added for non-workflow_run trigger, but it was in: %v", workflowData.Bots)
		}
	})
}

// TestBotsWithRolesAll tests that bots field works even when roles: all is set
func TestBotsWithRolesAll(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-bots-roles-all-test")

	compiler := NewCompiler()

	frontmatter := `---
on:
  issues:
    types: [opened]
  roles: all
  bots: ["dependabot[bot]"]
---

# Test Workflow
Test workflow content.`

	workflowPath := filepath.Join(tmpDir, "workflow-bots-roles-all.md")
	err := os.WriteFile(workflowPath, []byte(frontmatter), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	outputPath := filepath.Join(tmpDir, "workflow-bots-roles-all.lock.yml")
	compiledContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledContent)

	// When roles: all is set, no check_membership job should be generated
	// so the bots environment variable shouldn't appear
	if strings.Contains(compiledStr, "check_membership") {
		t.Errorf("Expected no check_membership job when roles: all is set")
	}
}
