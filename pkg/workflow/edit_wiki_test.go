package workflow

import (
	"strings"
	"testing"
)

func TestEditWikiConfigParsing(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("Should parse basic edit-wiki configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"edit-wiki": nil,
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.EditWiki == nil {
			t.Fatal("Expected EditWiki to be parsed")
		}

		// Check defaults
		if config.EditWiki.Max != 1 {
			t.Errorf("Expected default max to be 1, got %d", config.EditWiki.Max)
		}

		if len(config.EditWiki.Path) != 0 {
			t.Errorf("Expected default path to be empty, got %v", config.EditWiki.Path)
		}
	})

	t.Run("Should parse edit-wiki configuration with all options", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"edit-wiki": map[string]any{
					"path":         []any{"docs/", "wiki/"},
					"max":          3,
					"github-token": "${{ secrets.WIKI_PAT }}",
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.EditWiki == nil {
			t.Fatal("Expected EditWiki to be parsed")
		}

		if config.EditWiki.Max != 3 {
			t.Errorf("Expected max to be 3, got %d", config.EditWiki.Max)
		}

		expectedPaths := []string{"docs/", "wiki/"}
		if len(config.EditWiki.Path) != len(expectedPaths) {
			t.Errorf("Expected path length to be %d, got %d", len(expectedPaths), len(config.EditWiki.Path))
		}

		for i, expectedPath := range expectedPaths {
			if i < len(config.EditWiki.Path) && config.EditWiki.Path[i] != expectedPath {
				t.Errorf("Expected path[%d] to be %q, got %q", i, expectedPath, config.EditWiki.Path[i])
			}
		}

		if config.EditWiki.GitHubToken != "${{ secrets.WIKI_PAT }}" {
			t.Errorf("Expected github-token to be %q, got %q", "${{ secrets.WIKI_PAT }}", config.EditWiki.GitHubToken)
		}
	})

	t.Run("Should handle numeric max values of different types", func(t *testing.T) {
		testCases := []struct {
			name     string
			maxValue any
			expected int
		}{
			{"int", 5, 5},
			{"int64", int64(7), 7},
			{"uint64", uint64(3), 3},
			{"float64", float64(4), 4},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				frontmatter := map[string]any{
					"safe-outputs": map[string]any{
						"edit-wiki": map[string]any{
							"max": tc.maxValue,
						},
					},
				}

				config := compiler.extractSafeOutputsConfig(frontmatter)
				if config == nil {
					t.Fatal("Expected SafeOutputsConfig to be parsed")
				}

				if config.EditWiki == nil {
					t.Fatal("Expected EditWiki to be parsed")
				}

				if config.EditWiki.Max != tc.expected {
					t.Errorf("Expected max to be %d, got %d", tc.expected, config.EditWiki.Max)
				}
			})
		}
	})
}

func TestEditWikiJobCreation(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("Should build edit-wiki job with basic configuration", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Description: "Test workflow",
			SafeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{
					Max: 1,
				},
			},
		}

		job, err := compiler.buildEditWikiJob(workflowData, "main_job", false, nil)
		if err != nil {
			t.Fatalf("Expected no error building edit-wiki job, got: %v", err)
		}

		if job == nil {
			t.Fatal("Expected job to be created")
		}

		if job.Name != "edit_wiki" {
			t.Errorf("Expected job name to be 'edit_wiki', got %q", job.Name)
		}

		if job.RunsOn != "runs-on: ubuntu-latest" {
			t.Errorf("Expected job to run on ubuntu-latest, got %q", job.RunsOn)
		}

		if !strings.Contains(job.Permissions, "contents: write") {
			t.Error("Expected job to have contents: write permission")
		}

		if len(job.Needs) != 1 || job.Needs[0] != "main_job" {
			t.Errorf("Expected job to depend on main_job, got needs: %v", job.Needs)
		}

		// Check that the job has the expected steps
		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "Edit Wiki Pages") {
			t.Error("Expected job to have 'Edit Wiki Pages' step")
		}

		if !strings.Contains(stepsStr, "GITHUB_AW_AGENT_OUTPUT") {
			t.Error("Expected job to use GITHUB_AW_AGENT_OUTPUT environment variable")
		}

		if !strings.Contains(stepsStr, "GITHUB_WORKFLOW_NAME") {
			t.Error("Expected job to set GITHUB_WORKFLOW_NAME environment variable")
		}
	})

	t.Run("Should build edit-wiki job with path restrictions", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Description: "Test workflow",
			SafeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{
					Path: []string{"docs/", "wiki/help/"},
					Max:  2,
				},
			},
		}

		job, err := compiler.buildEditWikiJob(workflowData, "main_job", false, nil)
		if err != nil {
			t.Fatalf("Expected no error building edit-wiki job, got: %v", err)
		}

		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "GITHUB_AW_WIKI_ALLOWED_PATHS") {
			t.Error("Expected job to set GITHUB_AW_WIKI_ALLOWED_PATHS environment variable")
		}

		if !strings.Contains(stepsStr, "docs/,wiki/help/") {
			t.Error("Expected job to include allowed paths in environment variable")
		}

		if !strings.Contains(stepsStr, "GITHUB_AW_WIKI_MAX: 2") {
			t.Error("Expected job to set GITHUB_AW_WIKI_MAX environment variable")
		}
	})

	t.Run("Should build edit-wiki job with staged mode", func(t *testing.T) {
		staged := true
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Description: "Test workflow",
			SafeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{
					Max: 1,
				},
				Staged: &staged,
			},
		}

		job, err := compiler.buildEditWikiJob(workflowData, "main_job", false, nil)
		if err != nil {
			t.Fatalf("Expected no error building edit-wiki job, got: %v", err)
		}

		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"") {
			t.Error("Expected job to set staged mode environment variable")
		}
	})

	t.Run("Should build edit-wiki job with custom GitHub token", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Description: "Test workflow",
			SafeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{
					Max:         1,
					GitHubToken: "${{ secrets.WIKI_PAT }}",
				},
			},
		}

		job, err := compiler.buildEditWikiJob(workflowData, "main_job", false, nil)
		if err != nil {
			t.Fatalf("Expected no error building edit-wiki job, got: %v", err)
		}

		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "github-token: ${{ secrets.WIKI_PAT }}") {
			t.Error("Expected job to use custom GitHub token")
		}
	})

	t.Run("Should fail when SafeOutputs.EditWiki is nil", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Description: "Test workflow",
			SafeOutputs: &SafeOutputsConfig{
				// EditWiki is nil
			},
		}

		_, err := compiler.buildEditWikiJob(workflowData, "main_job", false, nil)
		if err == nil {
			t.Fatal("Expected error when EditWiki config is nil")
		}

		if !strings.Contains(err.Error(), "safe-outputs.edit-wiki configuration is required") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("Should fail when SafeOutputs is nil", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Description: "Test workflow",
			SafeOutputs: nil,
		}

		_, err := compiler.buildEditWikiJob(workflowData, "main_job", false, nil)
		if err == nil {
			t.Fatal("Expected error when SafeOutputs is nil")
		}

		if !strings.Contains(err.Error(), "safe-outputs.edit-wiki configuration is required") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})
}

func TestHasSafeOutputsEnabledWithEditWiki(t *testing.T) {
	t.Run("Should return true when EditWiki is configured", func(t *testing.T) {
		config := &SafeOutputsConfig{
			EditWiki: &EditWikiConfig{
				Max: 1,
			},
		}

		if !HasSafeOutputsEnabled(config) {
			t.Error("Expected HasSafeOutputsEnabled to return true when EditWiki is configured")
		}
	})

	t.Run("Should return false when no safe outputs are configured", func(t *testing.T) {
		config := &SafeOutputsConfig{}

		if HasSafeOutputsEnabled(config) {
			t.Error("Expected HasSafeOutputsEnabled to return false when no safe outputs are configured")
		}
	})

	t.Run("Should return false when SafeOutputsConfig is nil", func(t *testing.T) {
		if HasSafeOutputsEnabled(nil) {
			t.Error("Expected HasSafeOutputsEnabled to return false when SafeOutputsConfig is nil")
		}
	})
}