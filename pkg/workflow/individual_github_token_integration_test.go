//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestIndividualGitHubTokenIntegration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "individual-github-token-test")

	t.Run("create-issue uses individual github-token in generated workflow", func(t *testing.T) {
		testContent := `---
name: Test Individual GitHub Token for Issues
on:
  issues:
    types: [opened]
engine: claude
safe-outputs:
  github-token: ${{ secrets.GLOBAL_PAT }}
  create-issue:
    github-token: ${{ secrets.ISSUE_SPECIFIC_PAT }}
---

# Test Individual GitHub Token for Issues

This workflow tests that create-issue uses its own github-token.
`

		testFile := filepath.Join(tmpDir, "test-issue-token.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		// Read the generated YAML
		outputFile := filepath.Join(tmpDir, "test-issue-token.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that the safe_outputs job exists
		if !strings.Contains(yamlContent, "safe_outputs:") {
			t.Error("Expected safe_outputs job to be generated")
		}

		// Verify that the specific token is used for create_issue
		if !strings.Contains(yamlContent, "github-token: ${{ secrets.ISSUE_SPECIFIC_PAT }}") {
			t.Error("Expected safe_outputs job to use the issue-specific GitHub token")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}

		// Verify that the global token is not used in create_issue
		if strings.Contains(yamlContent, "github-token: ${{ secrets.GLOBAL_PAT }}") {
			// Check if it's in the safe_outputs job section specifically
			lines := strings.Split(yamlContent, "\n")
			inCreateIssueJob := false
			for _, line := range lines {
				if strings.Contains(line, "create_issue:") {
					inCreateIssueJob = true
					continue
				}
				if inCreateIssueJob && strings.HasPrefix(line, "  ") && strings.Contains(line, ":") && !strings.HasPrefix(line, "    ") {
					// We've moved to a new job
					inCreateIssueJob = false
				}
				if inCreateIssueJob && strings.Contains(line, "github-token: ${{ secrets.GLOBAL_PAT }}") {
					t.Error("safe_outputs job should not use the global GitHub token when individual token is specified")
				}
			}
		}
	})

	t.Run("create-pull-request fallback to global github-token when no individual token specified", func(t *testing.T) {
		testContent := `---
name: Test GitHub Token Fallback for PRs
on:
  issues:
    types: [opened]
engine: claude
safe-outputs:
  github-token: ${{ secrets.GLOBAL_PAT }}
  create-pull-request:
    draft: true
    # No github-token specified, should use global
  create-issue:
    github-token: ${{ secrets.ISSUE_SPECIFIC_PAT }}
---

# Test GitHub Token Fallback

This workflow tests that create-pull-request falls back to global github-token.
`

		testFile := filepath.Join(tmpDir, "test-pr-fallback.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		// Read the generated YAML
		outputFile := filepath.Join(tmpDir, "test-pr-fallback.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that both jobs exist and use correct tokens
		if !strings.Contains(yamlContent, "safe_outputs:") {
			t.Error("Expected create_pull_request job to be generated")
		}
		if !strings.Contains(yamlContent, "safe_outputs:") {
			t.Error("Expected safe_outputs job to be generated")
		}

		// Use simple string checks like the other working tests
		if !strings.Contains(yamlContent, "github-token: ${{ secrets.GLOBAL_PAT }}") {
			t.Error("Expected create_pull_request job to use global GitHub token (fallback)")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}

		if !strings.Contains(yamlContent, "github-token: ${{ secrets.ISSUE_SPECIFIC_PAT }}") {
			t.Error("Expected safe_outputs job to use individual GitHub token")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}
	})

	t.Run("add-labels uses individual github-token", func(t *testing.T) {
		testContent := `---
name: Test Individual GitHub Token for Labels
on:
  issues:
    types: [opened]
engine: claude
safe-outputs:
  github-token: ${{ secrets.GLOBAL_PAT }}
  add-labels:
    allowed: [bug, feature, enhancement]
    github-token: ${{ secrets.LABELS_PAT }}
---

# Test Individual GitHub Token for Labels

This workflow tests that add-labels uses its own github-token.
`

		testFile := filepath.Join(tmpDir, "test-labels-token.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		// Read the generated YAML
		outputFile := filepath.Join(tmpDir, "test-labels-token.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that the safe_outputs job is generated with handler manager step
		// (add-labels is now handled by the consolidated handler manager)
		if !strings.Contains(yamlContent, "id: process_safe_outputs") {
			t.Error("Expected safe_outputs job with process_safe_outputs step to be generated")
		}

		if !strings.Contains(yamlContent, "github-token: ${{ secrets.LABELS_PAT }}") {
			t.Error("Expected safe_outputs job to use the labels-specific GitHub token")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}
	})

	t.Run("backward compatibility - global github-token still works", func(t *testing.T) {
		testContent := `---
name: Test Backward Compatibility
on:
  issues:
    types: [opened]
engine: claude
safe-outputs:
  github-token: ${{ secrets.LEGACY_PAT }}
  create-issue:
    title-prefix: "[AUTO] "
    # No individual github-token, should use global
---

# Test Backward Compatibility

This workflow tests that the global github-token still works when no individual tokens are specified.
`

		testFile := filepath.Join(tmpDir, "test-backward-compatibility.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		// Read the generated YAML
		outputFile := filepath.Join(tmpDir, "test-backward-compatibility.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that the safe_outputs job uses the global token
		if !strings.Contains(yamlContent, "safe_outputs:") {
			t.Error("Expected safe_outputs job to be generated")
		}

		if !strings.Contains(yamlContent, "github-token: ${{ secrets.LEGACY_PAT }}") {
			t.Error("Expected safe_outputs job to use the global GitHub token for backward compatibility")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}
	})
}
