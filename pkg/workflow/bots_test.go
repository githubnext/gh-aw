//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name: "workflow with top-level bots array (shorthand for on.bots)",
			frontmatter: `---
bots: ["dependabot[bot]", "renovate[bot]"]
on:
  issues:
    types: [opened]
---

# Test Workflow
Test workflow content with top-level bots.`,
			filename:     "top-level-bots-array.md",
			expectedBots: []string{"dependabot[bot]", "renovate[bot]"},
		},
		{
			name: "workflow with top-level single bot",
			frontmatter: `---
bots: ["github-actions[bot]"]
on:
  pull_request:
    types: [opened]
---

# Test Workflow
Test workflow content with top-level single bot.`,
			filename:     "top-level-single-bot.md",
			expectedBots: []string{"github-actions[bot]"},
		},
		{
			name: "on.bots takes precedence over top-level bots",
			frontmatter: `---
bots: ["top-level-bot[bot]"]
on:
  issues:
    types: [opened]
  bots: ["on-bots-bot[bot]"]
---

# Test Workflow
Test that on.bots takes precedence over top-level bots.`,
			filename:     "on-bots-precedence.md",
			expectedBots: []string{"on-bots-bot[bot]"},
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

	// Check that the bots environment variable is set (value is %q-quoted)
	if !strings.Contains(compiledStr, `GH_AW_ALLOWED_BOTS: "dependabot[bot],renovate[bot]"`) {
		t.Errorf("Expected compiled workflow to contain GH_AW_ALLOWED_BOTS environment variable")
	}

	// Also check that roles are still present
	if !strings.Contains(compiledStr, `GH_AW_REQUIRED_ROLES: "triage"`) {
		t.Errorf("Expected compiled workflow to contain GH_AW_REQUIRED_ROLES environment variable")
	}
}

// TestTopLevelBotsCompilation tests that top-level bots (shorthand for on.bots) are compiled correctly
func TestTopLevelBotsCompilation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-top-level-bots-test")

	compiler := NewCompiler()

	frontmatter := `---
bots: ["dependabot[bot]", "renovate[bot]"]
on:
  issues:
    types: [opened]
  roles: [triage]
---

# Test Workflow with Top-Level Bots
Test workflow content.`

	workflowPath := filepath.Join(tmpDir, "workflow-top-level-bots.md")
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
	outputPath := filepath.Join(tmpDir, "workflow-top-level-bots.lock.yml")
	compiledContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that the top-level bots are picked up and produce the GH_AW_ALLOWED_BOTS env var
	if !strings.Contains(compiledStr, `GH_AW_ALLOWED_BOTS: "dependabot[bot],renovate[bot]"`) {
		t.Errorf("Expected compiled workflow to contain GH_AW_ALLOWED_BOTS with top-level bots; compiled output:\n%s", compiledStr)
	}
}

// TestOnBotsPrecedenceOverTopLevelBots verifies that on.bots takes precedence over top-level bots
// when both are present, ensuring backward compatibility.
func TestOnBotsPrecedenceOverTopLevelBots(t *testing.T) {
	tmpDir := testutil.TempDir(t, "workflow-on-bots-precedence-test")

	compiler := NewCompiler()

	frontmatter := `---
bots: ["top-level-bot[bot]"]
on:
  issues:
    types: [opened]
  bots: ["on-bots-bot[bot]"]
---

# Test Workflow on.bots Precedence
Test that on.bots takes precedence over top-level bots.`

	workflowPath := filepath.Join(tmpDir, "workflow-on-bots-precedence.md")
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
	outputPath := filepath.Join(tmpDir, "workflow-on-bots-precedence.lock.yml")
	compiledContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledContent)

	// Only on.bots value should appear in GH_AW_ALLOWED_BOTS
	if !strings.Contains(compiledStr, `GH_AW_ALLOWED_BOTS: "on-bots-bot[bot]"`) {
		t.Errorf("Expected GH_AW_ALLOWED_BOTS to contain only on.bots value; compiled output:\n%s", compiledStr)
	}

	// Top-level bot should NOT appear in GH_AW_ALLOWED_BOTS
	if strings.Contains(compiledStr, "top-level-bot") {
		t.Errorf("Expected top-level bots to be ignored when on.bots is present; compiled output:\n%s", compiledStr)
	}
}

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
	if !strings.Contains(compiledStr, `GH_AW_REQUIRED_ROLES: "admin,maintainer,write"`) {
		t.Errorf("Expected compiled workflow to contain default GH_AW_REQUIRED_ROLES")
	}

	// Check that bots environment variable is set (value is %q-quoted)
	if !strings.Contains(compiledStr, `GH_AW_ALLOWED_BOTS: "dependabot[bot]"`) {
		t.Errorf("Expected compiled workflow to contain GH_AW_ALLOWED_BOTS environment variable")
	}
}

// TestMergeBots tests the mergeBots helper function
func TestMergeBots(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name     string
		top      []string
		imported []string
		expected []string
	}{
		{
			name:     "top only",
			top:      []string{"dependabot[bot]"},
			imported: nil,
			expected: []string{"dependabot[bot]"},
		},
		{
			name:     "imported only",
			top:      nil,
			imported: []string{"renovate[bot]"},
			expected: []string{"renovate[bot]"},
		},
		{
			name:     "both with no overlap",
			top:      []string{"dependabot[bot]"},
			imported: []string{"renovate[bot]"},
			expected: []string{"dependabot[bot]", "renovate[bot]"},
		},
		{
			name:     "both with duplicates deduped",
			top:      []string{"dependabot[bot]", "renovate[bot]"},
			imported: []string{"renovate[bot]", "github-actions[bot]"},
			expected: []string{"dependabot[bot]", "renovate[bot]", "github-actions[bot]"},
		},
		{
			name:     "both nil",
			top:      nil,
			imported: nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.mergeBots(tt.top, tt.imported)
			assert.Equal(t, tt.expected, result, "mergeBots result mismatch")
		})
	}
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

// TestBotsImportMerge tests that bots from imported workflows are merged with top-level bots
// in the compiled output (regression test for the fix in compiler_orchestrator_workflow.go).
func TestBotsImportMerge(t *testing.T) {
	compiler := NewCompiler()

	t.Run("imported_bots_merged_with_top_level_bots", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "bots-import-merge-test")

		// Shared workflow defines a bot at the top level (the format used by the importer)
		sharedContent := `---
on: issues
bots:
  - "renovate[bot]"
---
`
		sharedPath := filepath.Join(tmpDir, "shared-bots.md")
		err := os.WriteFile(sharedPath, []byte(sharedContent), 0644)
		require.NoError(t, err, "Failed to write shared workflow file")

		// Main workflow defines its own bot inside on.bots and imports the shared file
		mainContent := `---
on:
  issues:
    types: [opened]
  bots: ["dependabot[bot]"]
imports:
  - shared-bots.md
---

# Main workflow importing bots from a shared file.
`
		mainPath := filepath.Join(tmpDir, "main-workflow.md")
		err = os.WriteFile(mainPath, []byte(mainContent), 0644)
		require.NoError(t, err, "Failed to write main workflow file")

		err = compiler.CompileWorkflow(mainPath)
		require.NoError(t, err, "Compilation failed")

		lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(mainPath))
		require.NoError(t, err, "Failed to read lock file")
		lockStr := string(lockContent)

		// Both top-level and imported bots must appear in the compiled output
		assert.Contains(t, lockStr, `GH_AW_ALLOWED_BOTS: "dependabot[bot],renovate[bot]"`,
			"Expected compiled workflow to contain both top-level and imported bots")
	})

	t.Run("imported_bots_only_no_top_level_bots", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "bots-import-only-test")

		sharedContent := `---
on: issues
bots:
  - "github-actions[bot]"
---
`
		sharedPath := filepath.Join(tmpDir, "shared-bots-only.md")
		err := os.WriteFile(sharedPath, []byte(sharedContent), 0644)
		require.NoError(t, err, "Failed to write shared workflow file")

		// Main workflow has no on.bots of its own; relies solely on the import
		mainContent := `---
on:
  issues:
    types: [opened]
imports:
  - shared-bots-only.md
---

# Main workflow with bots defined only in the import.
`
		mainPath := filepath.Join(tmpDir, "main-no-bots.md")
		err = os.WriteFile(mainPath, []byte(mainContent), 0644)
		require.NoError(t, err, "Failed to write main workflow file")

		err = compiler.CompileWorkflow(mainPath)
		require.NoError(t, err, "Compilation failed")

		lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(mainPath))
		require.NoError(t, err, "Failed to read lock file")
		lockStr := string(lockContent)

		// Imported bot must appear even when the main workflow has no on.bots
		assert.Contains(t, lockStr, `GH_AW_ALLOWED_BOTS: "github-actions[bot]"`,
			"Expected compiled workflow to contain bots from import when main workflow has none")
	})

	t.Run("duplicate_bots_across_top_level_and_import_deduped", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "bots-import-dedup-test")

		sharedContent := `---
on: issues
bots:
  - "dependabot[bot]"
  - "renovate[bot]"
---
`
		sharedPath := filepath.Join(tmpDir, "shared-bots-dup.md")
		err := os.WriteFile(sharedPath, []byte(sharedContent), 0644)
		require.NoError(t, err, "Failed to write shared workflow file")

		// Main workflow overlaps with the shared workflow's bots
		mainContent := `---
on:
  issues:
    types: [opened]
  bots: ["dependabot[bot]", "github-actions[bot]"]
imports:
  - shared-bots-dup.md
---

# Main workflow with overlapping bots in import.
`
		mainPath := filepath.Join(tmpDir, "main-dup-bots.md")
		err = os.WriteFile(mainPath, []byte(mainContent), 0644)
		require.NoError(t, err, "Failed to write main workflow file")

		err = compiler.CompileWorkflow(mainPath)
		require.NoError(t, err, "Compilation failed")

		lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(mainPath))
		require.NoError(t, err, "Failed to read lock file")
		lockStr := string(lockContent)

		// dependabot[bot] appears in both top-level and import — must be deduplicated
		assert.Contains(t, lockStr, `GH_AW_ALLOWED_BOTS: "dependabot[bot],github-actions[bot],renovate[bot]"`,
			"Expected compiled workflow to deduplicate bots from top-level and import")
	})
}
