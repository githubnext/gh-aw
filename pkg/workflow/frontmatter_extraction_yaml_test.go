//go:build !integration

package workflow

import "testing"

func TestIsGitHubAppNestedField(t *testing.T) {
	t.Run("supports ignore-if-missing field", func(t *testing.T) {
		if !isGitHubAppNestedField("ignore-if-missing: true") {
			t.Fatal("expected ignore-if-missing to be treated as on.github-app nested field")
		}
	})
}

func TestExtractPullRequestReviewerConfig(t *testing.T) {
	c := &Compiler{}
	t.Run("returns true for empty value", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_reviewer": nil,
			},
		}
		if !c.extractPullRequestReviewerConfig(frontmatter) {
			t.Fatal("expected empty pull_request_reviewer to be detected")
		}
	})
	t.Run("returns true for slash_command", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_reviewer": "slash_command",
			},
		}
		if !c.extractPullRequestReviewerConfig(frontmatter) {
			t.Fatal("expected pull_request_reviewer slash_command to be detected")
		}
	})
	t.Run("returns true for custom command name", func(t *testing.T) {
		frontmatter := map[string]any{
			"on": map[string]any{
				"pull_request_reviewer": "reviewer-command",
			},
		}
		if !c.extractPullRequestReviewerConfig(frontmatter) {
			t.Fatal("expected pull_request_reviewer custom command to be detected")
		}
	})
	t.Run("returns false for missing trigger", func(t *testing.T) {
		frontmatter := map[string]any{"on": map[string]any{"issues": map[string]any{}}}
		if c.extractPullRequestReviewerConfig(frontmatter) {
			t.Fatal("expected false when pull_request_reviewer is missing")
		}
	})
}
