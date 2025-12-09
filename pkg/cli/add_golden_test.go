//go:build integration

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestAddCommandGolden tests the 'gh aw add' command with golden file snapshots.
// Golden tests verify that generated scaffolding matches expected output,
// helping ensure that changes to generated files are intentional and reviewed.
//
// This test uses a simpler approach: it directly adds pre-created workflows
// without going through the package installation system, focusing purely on
// testing the file generation and compilation aspects.
func TestAddCommandGolden(t *testing.T) {
	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	tests := []struct {
		name           string
		sourceWorkflow string   // Source workflow content to add
		workflowName   string   // Name for the workflow file
		nameFlag       string   // -n flag value (custom name)
		dirFlag        string   // --dir flag value (subdirectory)
		numberFlag     int      // -c flag value (number of copies)
		goldenFiles    []string // List of files to snapshot
		skipReason     string   // If non-empty, skip this test
	}{
		{
			name:         "add_simple_workflow",
			workflowName: "simple-workflow",
			sourceWorkflow: `---
on:
  issues:
    types: [opened]
permissions:
  issues: read
engine: copilot
timeout-minutes: 5
safe-outputs:
  add-comment:
---

# Simple Test Workflow

This is a simple test workflow for golden testing.

When an issue is opened, add a comment saying "Hello from simple workflow!".
`,
			numberFlag: 1,
			goldenFiles: []string{
				"simple-workflow.md",
				"simple-workflow.lock.yml",
			},
		},
		{
			name:         "add_workflow_with_custom_name",
			workflowName: "original-name",
			nameFlag:     "custom-name",
			sourceWorkflow: `---
on:
  issues:
    types: [opened]
permissions:
  issues: read
engine: copilot
timeout-minutes: 5
safe-outputs:
  add-comment:
---

# Original Workflow Name

This workflow will be added with a custom name.
`,
			numberFlag: 1,
			goldenFiles: []string{
				"custom-name.md",
				"custom-name.lock.yml",
			},
		},
		{
			name:         "add_workflow_to_subdirectory",
			workflowName: "shared-workflow",
			dirFlag:      "shared",
			sourceWorkflow: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
timeout-minutes: 5
---

# Shared Workflow

This is a shared workflow that will be added to a subdirectory.
`,
			numberFlag: 1,
			goldenFiles: []string{
				"shared/shared-workflow.md",
				"shared/shared-workflow.lock.yml",
			},
		},
		{
			name:         "add_workflow_with_numbered_copies",
			workflowName: "multi-copy",
			sourceWorkflow: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
timeout-minutes: 5
---

# Multi Copy Workflow

This workflow will be added with multiple copies.
`,
			numberFlag: 3,
			goldenFiles: []string{
				"multi-copy-1.md",
				"multi-copy-1.lock.yml",
				"multi-copy-2.md",
				"multi-copy-2.lock.yml",
				"multi-copy-3.md",
				"multi-copy-3.lock.yml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			// Create temporary test repository
			testRepoDir := testutil.TempDir(t, "golden-test-*")

			// Initialize as git repository
			gitInit := exec.Command("git", "init")
			gitInit.Dir = testRepoDir
			if err := gitInit.Run(); err != nil {
				t.Fatalf("Failed to git init: %v", err)
			}

			// Configure git
			exec.Command("git", "-C", testRepoDir, "config", "user.name", "Test User").Run()
			exec.Command("git", "-C", testRepoDir, "config", "user.email", "test@example.com").Run()

			// Create initial commit
			readmeContent := "# Test Repository\n"
			readmePath := filepath.Join(testRepoDir, "README.md")
			if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
				t.Fatalf("Failed to write README: %v", err)
			}

			exec.Command("git", "-C", testRepoDir, "add", ".").Run()
			exec.Command("git", "-C", testRepoDir, "commit", "-m", "Initial commit").Run()

			// Change to test repo directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(testRepoDir); err != nil {
				t.Fatalf("Failed to change to test repo dir: %v", err)
			}

			// Determine the target workflow directory
			var workflowsDir string
			if tt.dirFlag != "" {
				workflowsDir = filepath.Join(testRepoDir, ".github", "workflows", tt.dirFlag)
			} else {
				workflowsDir = filepath.Join(testRepoDir, ".github", "workflows")
			}

			// Create workflows directory
			if err := os.MkdirAll(workflowsDir, 0755); err != nil {
				t.Fatalf("Failed to create workflows dir: %v", err)
			}

			// Determine the final workflow name
			finalName := tt.workflowName
			if tt.nameFlag != "" {
				finalName = tt.nameFlag
			}

			// Create workflow files based on number flag
			for i := 1; i <= tt.numberFlag; i++ {
				var workflowPath string
				content := tt.sourceWorkflow

				if tt.numberFlag == 1 {
					workflowPath = filepath.Join(workflowsDir, finalName+".md")
				} else {
					workflowPath = filepath.Join(workflowsDir, fmt.Sprintf("%s-%d.md", finalName, i))

					// Update H1 title for numbered workflows
					lines := strings.Split(content, "\n")
					for j, line := range lines {
						if strings.HasPrefix(strings.TrimSpace(line), "# ") {
							// Extract the title part and add number
							title := strings.TrimSpace(line[2:])
							lines[j] = fmt.Sprintf("# %s %d", title, i)
							break
						}
					}
					content = strings.Join(lines, "\n")
				}

				// Write workflow file
				if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write workflow file: %v", err)
				}

				// Compile the workflow
				compiler := workflow.NewCompiler(false, "", "test")
				if err := CompileWorkflowWithValidation(compiler, workflowPath, false, false, false, false, false, false); err != nil {
					t.Fatalf("Failed to compile workflow: %v", err)
				}
			}

			// Check each golden file
			for _, goldenFile := range tt.goldenFiles {
				generatedPath := filepath.Join(testRepoDir, ".github", "workflows", goldenFile)
				goldenPath := filepath.Join(originalDir, "testdata", "add_golden", tt.name, goldenFile)

				// Read generated content
				generatedContent, err := os.ReadFile(generatedPath)
				if err != nil {
					t.Fatalf("Failed to read generated file %s: %v", goldenFile, err)
				}

				// Normalize content for comparison
				normalizedContent := normalizeGoldenContent(string(generatedContent))

				if updateGolden {
					// Update golden file
					goldenDir := filepath.Dir(goldenPath)
					if err := os.MkdirAll(goldenDir, 0755); err != nil {
						t.Fatalf("Failed to create golden dir: %v", err)
					}

					if err := os.WriteFile(goldenPath, []byte(normalizedContent), 0644); err != nil {
						t.Fatalf("Failed to write golden file: %v", err)
					}
					t.Logf("Updated golden file: %s", goldenPath)
				} else {
					// Compare with golden file
					goldenContent, err := os.ReadFile(goldenPath)
					if err != nil {
						t.Fatalf("Failed to read golden file %s: %v\nRun with UPDATE_GOLDEN=1 to generate golden files", goldenPath, err)
					}

					if normalizedContent != string(goldenContent) {
						t.Errorf("Generated file %s does not match golden file.\nGenerated:\n%s\n\nGolden:\n%s\n\nRun with UPDATE_GOLDEN=1 to update golden files",
							goldenFile, normalizedContent, string(goldenContent))
					}
				}
			}
		})
	}
}

// normalizeGoldenContent normalizes file content for golden file comparison
// by removing dynamic content like timestamps, commit SHAs, etc.
func normalizeGoldenContent(content string) string {
	lines := strings.Split(content, "\n")
	var normalized []string

	for _, line := range lines {
		// Skip comment lines that contain timestamps or version info
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Skip lines with "Generated by", timestamps, or version info
			if strings.Contains(line, "Generated by") ||
				strings.Contains(line, "gh-aw version") ||
				strings.Contains(line, "2024") || // Years in timestamps
				strings.Contains(line, "2025") {
				continue
			}
		}

		// Normalize source field that contains commit SHAs
		if strings.Contains(line, "source:") && strings.Contains(line, "@") {
			// Replace commit SHA with placeholder
			// Format: source: owner/repo/workflow.md@SHA
			if idx := strings.LastIndex(line, "@"); idx != -1 {
				line = line[:idx] + "@<commit-sha>"
			}
		}

		normalized = append(normalized, line)
	}

	return strings.Join(normalized, "\n")
}
