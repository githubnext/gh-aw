package workflow

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestSafeOutputsBackwardCompatibility(t *testing.T) {
	compiler := &Compiler{}

	t.Run("Legacy singular syntax should still work", func(t *testing.T) {
		content := `---
safe-outputs:
  create-issue:
    title-prefix: "[Auto] "
    labels: ["bug", "auto-generated"]
  add-issue-comment:
  create-pull-request:
    title-prefix: "[Fix] "
    draft: true
---

# Test workflow

This workflow should work with legacy syntax.
`

		// Parse the workflow content
		result, err := parser.ExtractFrontmatterFromContent(content)
		if err != nil {
			t.Fatalf("Failed to parse frontmatter: %v", err)
		}

		config := compiler.extractSafeOutputsConfig(result.Frontmatter)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		// Verify create-issue (singular) is converted to create-issues with max: 1
		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed from legacy create-issue")
		}
		if config.CreateIssues.Max != 1 {
			t.Errorf("Expected CreateIssues.Max to be 1 for legacy syntax, got %d", config.CreateIssues.Max)
		}
		if config.CreateIssues.TitlePrefix != "[Auto] " {
			t.Errorf("Expected TitlePrefix '[Auto] ', got '%s'", config.CreateIssues.TitlePrefix)
		}
		if len(config.CreateIssues.Labels) != 2 {
			t.Errorf("Expected 2 labels, got %d", len(config.CreateIssues.Labels))
		}

		// Verify add-issue-comment (singular) is converted to add-issue-comments with max: 1
		if config.AddIssueComments == nil {
			t.Fatal("Expected AddIssueComments to be parsed from legacy add-issue-comment")
		}
		if config.AddIssueComments.Max != 1 {
			t.Errorf("Expected AddIssueComments.Max to be 1 for legacy syntax, got %d", config.AddIssueComments.Max)
		}

		// Verify create-pull-request (singular) stays as max: 1
		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests to be parsed from create-pull-request")
		}
		if config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 for singular syntax, got %d", config.CreatePullRequests.Max)
		}
		if config.CreatePullRequests.TitlePrefix != "[Fix] " {
			t.Errorf("Expected TitlePrefix '[Fix] ', got '%s'", config.CreatePullRequests.TitlePrefix)
		}
		if config.CreatePullRequests.Draft == nil || *config.CreatePullRequests.Draft != true {
			t.Errorf("Expected Draft to be true, got %v", config.CreatePullRequests.Draft)
		}
	})

	t.Run("New plural syntax should work", func(t *testing.T) {
		content := `---
safe-outputs:
  create-issues:
    title-prefix: "[Batch] "
    labels: ["enhancement"]
    max: 5
  add-issue-comments:
    max: 3
  create-pull-request:
    title-prefix: "[Single] "
    draft: false
---

# Test workflow

This workflow uses the new plural syntax for issues and comments, singular for pull requests.
`

		// Parse the workflow content
		result, err := parser.ExtractFrontmatterFromContent(content)
		if err != nil {
			t.Fatalf("Failed to parse frontmatter: %v", err)
		}

		config := compiler.extractSafeOutputsConfig(result.Frontmatter)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		// Verify create-issues (plural) with explicit max
		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed")
		}
		if config.CreateIssues.Max != 5 {
			t.Errorf("Expected CreateIssues.Max to be 5, got %d", config.CreateIssues.Max)
		}
		if config.CreateIssues.TitlePrefix != "[Batch] " {
			t.Errorf("Expected TitlePrefix '[Batch] ', got '%s'", config.CreateIssues.TitlePrefix)
		}

		// Verify add-issue-comments (plural) with explicit max
		if config.AddIssueComments == nil {
			t.Fatal("Expected AddIssueComments to be parsed")
		}
		if config.AddIssueComments.Max != 3 {
			t.Errorf("Expected AddIssueComments.Max to be 3, got %d", config.AddIssueComments.Max)
		}

		// Verify create-pull-request (singular) is always max: 1
		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests to be parsed")
		}
		if config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 (singular), got %d", config.CreatePullRequests.Max)
		}
		if config.CreatePullRequests.Draft == nil || *config.CreatePullRequests.Draft != false {
			t.Errorf("Expected Draft to be false, got %v", config.CreatePullRequests.Draft)
		}
	})

	t.Run("Plural syntax without explicit max should default to 10", func(t *testing.T) {
		content := `---
safe-outputs:
  create-issues:
  add-issue-comments:
  create-pull-request:
---

# Test workflow

This workflow uses plural syntax without explicit max values (except pull request which is always singular).
`

		// Parse the workflow content
		result, err := parser.ExtractFrontmatterFromContent(content)
		if err != nil {
			t.Fatalf("Failed to parse frontmatter: %v", err)
		}

		config := compiler.extractSafeOutputsConfig(result.Frontmatter)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		// Issues and comments should default to max: 10, pull requests always max: 1
		if config.CreateIssues == nil || config.CreateIssues.Max != 10 {
			t.Errorf("Expected CreateIssues.Max to be 10, got %d", config.CreateIssues.Max)
		}
		if config.AddIssueComments == nil || config.AddIssueComments.Max != 10 {
			t.Errorf("Expected AddIssueComments.Max to be 10, got %d", config.AddIssueComments.Max)
		}
		if config.CreatePullRequests == nil || config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 (singular), got %d", config.CreatePullRequests.Max)
		}
	})
}

func TestWorkflowCompilationWithPluralSafeOutputs(t *testing.T) {
	t.Run("Workflow should parse plural safe-outputs configuration", func(t *testing.T) {
		content := `---
safe-outputs:
  create-issues:
    max: 2
  add-issue-comments:
    max: 4
---

# Test workflow

This workflow uses plural safe-outputs and should compile successfully.

Analyze the repository and create issues for any problems found.
`

		// Parse just the frontmatter content
		result, err := parser.ExtractFrontmatterFromContent(content)
		if err != nil {
			t.Fatalf("Failed to parse frontmatter: %v", err)
		}

		// Extract safe-outputs configuration
		compiler := &Compiler{}
		config := compiler.extractSafeOutputsConfig(result.Frontmatter)

		// Verify the configuration was parsed correctly
		if config == nil {
			t.Fatal("Expected SafeOutputs to be parsed")
		}
		if config.CreateIssues == nil || config.CreateIssues.Max != 2 {
			t.Errorf("Expected CreateIssues.Max to be 2, got %v", config.CreateIssues)
		}
		if config.AddIssueComments == nil || config.AddIssueComments.Max != 4 {
			t.Errorf("Expected AddIssueComments.Max to be 4, got %v", config.AddIssueComments)
		}
	})
}
