package workflow

import (
	"testing"
)

func TestUpdateReleasesConfigParsing(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Test basic update-releases configuration
	outputMap := map[string]any{
		"update-releases": map[string]any{
			"max":        5,
			"target":     "triggering",
			"release-id": "${{ github.event.release.id }}",
		},
	}

	config := compiler.parseUpdateReleasesConfig(outputMap)
	if config == nil {
		t.Fatal("Expected UpdateReleasesConfig to be parsed")
	}

	if config.Max != 5 {
		t.Errorf("Expected Max to be 5, got %d", config.Max)
	}

	if config.Target != "triggering" {
		t.Errorf("Expected Target to be 'triggering', got %s", config.Target)
	}

	if config.ReleaseID != "${{ github.event.release.id }}" {
		t.Errorf("Expected ReleaseID to be '${{ github.event.release.id }}', got %s", config.ReleaseID)
	}
}

func TestUpdateReleasesConfigDefaults(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Test minimal configuration with defaults
	outputMap := map[string]any{
		"update-releases": map[string]any{},
	}

	config := compiler.parseUpdateReleasesConfig(outputMap)
	if config == nil {
		t.Fatal("Expected UpdateReleasesConfig to be parsed")
	}

	if config.Max != 1 {
		t.Errorf("Expected default Max to be 1, got %d", config.Max)
	}

	if config.Target != "" {
		t.Errorf("Expected default Target to be empty, got %s", config.Target)
	}

	if config.ReleaseID != "" {
		t.Errorf("Expected default ReleaseID to be empty, got %s", config.ReleaseID)
	}
}

func TestUpdateReleasesConfigGitHubToken(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Test configuration with GitHub token
	outputMap := map[string]any{
		"update-releases": map[string]any{
			"github-token": "${{ secrets.CUSTOM_PAT }}",
		},
	}

	config := compiler.parseUpdateReleasesConfig(outputMap)
	if config == nil {
		t.Fatal("Expected UpdateReleasesConfig to be parsed")
	}

	if config.GitHubToken != "${{ secrets.CUSTOM_PAT }}" {
		t.Errorf("Expected GitHubToken to be '${{ secrets.CUSTOM_PAT }}', got %s", config.GitHubToken)
	}
}
