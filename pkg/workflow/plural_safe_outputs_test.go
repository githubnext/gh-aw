package workflow

import (
	"testing"
)

func TestPluralSafeOutputs(t *testing.T) {
	compiler := &Compiler{}

	t.Run("Singular forms should convert to max: 1", func(t *testing.T) {
		testSingular := map[string]any{
			"safe-outputs": map[string]any{
				"create-issue":        nil,
				"add-issue-comment":   nil,
				"create-pull-request": nil,
			},
		}

		config := compiler.extractSafeOutputsConfig(testSingular)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed")
		}
		if config.CreateIssues.Max != 1 {
			t.Errorf("Expected CreateIssues.Max to be 1 for singular form, got %d", config.CreateIssues.Max)
		}

		if config.AddIssueComments == nil {
			t.Fatal("Expected AddIssueComments to be parsed")
		}
		if config.AddIssueComments.Max != 1 {
			t.Errorf("Expected AddIssueComments.Max to be 1 for singular form, got %d", config.AddIssueComments.Max)
		}

		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests to be parsed")
		}
		if config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 for singular form, got %d", config.CreatePullRequests.Max)
		}
	})

	t.Run("Plural forms should default to max: 10", func(t *testing.T) {
		testPlural := map[string]any{
			"safe-outputs": map[string]any{
				"create-issues":       nil,
				"add-issue-comments":  nil,
				"create-pull-request": nil, // Note: singular, not plural
			},
		}

		config := compiler.extractSafeOutputsConfig(testPlural)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed")
		}
		if config.CreateIssues.Max != 10 {
			t.Errorf("Expected CreateIssues.Max to be 10 for plural form, got %d", config.CreateIssues.Max)
		}

		if config.AddIssueComments == nil {
			t.Fatal("Expected AddIssueComments to be parsed")
		}
		if config.AddIssueComments.Max != 10 {
			t.Errorf("Expected AddIssueComments.Max to be 10 for plural form, got %d", config.AddIssueComments.Max)
		}

		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests to be parsed")
		}
		if config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 for singular form, got %d", config.CreatePullRequests.Max)
		}
	})

	t.Run("Plural forms with explicit max should use provided value", func(t *testing.T) {
		testPluralMax := map[string]any{
			"safe-outputs": map[string]any{
				"create-issues": map[string]any{
					"max": 3,
				},
				"add-issue-comments": map[string]any{
					"max": 5,
				},
				"create-pull-request": map[string]any{
					// max parameter is ignored for pull requests
					"max": 2,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(testPluralMax)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed")
		}
		if config.CreateIssues.Max != 3 {
			t.Errorf("Expected CreateIssues.Max to be 3, got %d", config.CreateIssues.Max)
		}

		if config.AddIssueComments == nil {
			t.Fatal("Expected AddIssueComments to be parsed")
		}
		if config.AddIssueComments.Max != 5 {
			t.Errorf("Expected AddIssueComments.Max to be 5, got %d", config.AddIssueComments.Max)
		}

		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests to be parsed")
		}
		if config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 (max ignored for pull requests), got %d", config.CreatePullRequests.Max)
		}
	})

	t.Run("Mixed configurations should work correctly", func(t *testing.T) {
		testMixed := map[string]any{
			"safe-outputs": map[string]any{
				"create-issues": map[string]any{
					"title-prefix": "[Auto] ",
					"labels":       []any{"bug", "auto-generated"},
					"max":          2,
				},
				"create-pull-request": map[string]any{
					"title-prefix": "[Fix] ",
					"labels":       []any{"fix"},
					"draft":        true,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(testMixed)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		// Check plural create-issues
		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed")
		}
		if config.CreateIssues.Max != 2 {
			t.Errorf("Expected CreateIssues.Max to be 2, got %d", config.CreateIssues.Max)
		}
		if config.CreateIssues.TitlePrefix != "[Auto] " {
			t.Errorf("Expected CreateIssues.TitlePrefix to be '[Auto] ', got '%s'", config.CreateIssues.TitlePrefix)
		}
		if len(config.CreateIssues.Labels) != 2 || config.CreateIssues.Labels[0] != "bug" || config.CreateIssues.Labels[1] != "auto-generated" {
			t.Errorf("Expected CreateIssues.Labels to be ['bug', 'auto-generated'], got %v", config.CreateIssues.Labels)
		}

		// Check singular create-pull-request (should convert to max: 1)
		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests to be parsed")
		}
		if config.CreatePullRequests.Max != 1 {
			t.Errorf("Expected CreatePullRequests.Max to be 1 for singular form, got %d", config.CreatePullRequests.Max)
		}
		if config.CreatePullRequests.TitlePrefix != "[Fix] " {
			t.Errorf("Expected CreatePullRequests.TitlePrefix to be '[Fix] ', got '%s'", config.CreatePullRequests.TitlePrefix)
		}
		if len(config.CreatePullRequests.Labels) != 1 || config.CreatePullRequests.Labels[0] != "fix" {
			t.Errorf("Expected CreatePullRequests.Labels to be ['fix'], got %v", config.CreatePullRequests.Labels)
		}
		if config.CreatePullRequests.Draft == nil || *config.CreatePullRequests.Draft != true {
			t.Errorf("Expected CreatePullRequests.Draft to be true, got %v", config.CreatePullRequests.Draft)
		}
	})

	t.Run("Should prefer plural form when both singular and plural are present", func(t *testing.T) {
		testBoth := map[string]any{
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{
					"title-prefix": "[Singular] ",
				},
				"create-issues": map[string]any{
					"title-prefix": "[Plural] ",
					"max":          5,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(testBoth)
		if config == nil {
			t.Fatal("Expected config to be parsed")
		}

		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues to be parsed")
		}
		if config.CreateIssues.Max != 5 {
			t.Errorf("Expected CreateIssues.Max to be 5 (from plural form), got %d", config.CreateIssues.Max)
		}
		if config.CreateIssues.TitlePrefix != "[Plural] " {
			t.Errorf("Expected CreateIssues.TitlePrefix to be '[Plural] ' (from plural form), got '%s'", config.CreateIssues.TitlePrefix)
		}
	})
}
