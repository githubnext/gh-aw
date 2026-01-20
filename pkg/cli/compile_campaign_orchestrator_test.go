package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestGenerateAndCompileCampaignOrchestrator(t *testing.T) {
	tmpDir := t.TempDir()

	campaignSpecPath := filepath.Join(tmpDir, "test-campaign.campaign.md")

	spec := &campaign.CampaignSpec{
		ID:          "test-campaign",
		Name:        "Test Campaign",
		Description: "A test campaign",
		Workflows:   []string{"example-workflow"},
		MemoryPaths: []string{"memory/campaigns/test-campaign/**"},
	}

	compiler := workflow.NewCompiler(false, "", GetVersion())
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(false)
	compiler.SetStrictMode(false)

	orchestratorPath, err := generateAndCompileCampaignOrchestrator(GenerateCampaignOrchestratorOptions{
		Compiler:             compiler,
		Spec:                 spec,
		CampaignSpecPath:     campaignSpecPath,
		Verbose:              false,
		NoEmit:               false,
		RunZizmorPerFile:     false,
		RunPoutinePerFile:    false,
		RunActionlintPerFile: false,
		Strict:               false,
		ValidateActionSHAs:   false,
	})
	if err != nil {
		t.Fatalf("generateAndCompileCampaignOrchestrator() error: %v", err)
	}

	expectedPath := strings.TrimSuffix(campaignSpecPath, ".campaign.md") + ".campaign.g.md"
	if orchestratorPath != expectedPath {
		t.Fatalf("unexpected orchestrator path: got %q, want %q", orchestratorPath, expectedPath)
	}

	if _, statErr := os.Stat(orchestratorPath); statErr != nil {
		t.Fatalf("expected orchestrator markdown to exist, stat error: %v", statErr)
	}

	// For campaign orchestrators (*.campaign.g.md), the lock file should be *.campaign.lock.yml
	lockPath := strings.TrimSuffix(campaignSpecPath, ".campaign.md") + ".campaign.lock.yml"
	if _, statErr := os.Stat(lockPath); statErr != nil {
		t.Fatalf("expected orchestrator lock file to exist at %s, stat error: %v", lockPath, statErr)
	}

	// Verify that the generated orchestrator has the required permissions
	lockContent, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		t.Fatalf("failed to read lock file: %v", readErr)
	}
	lockStr := string(lockContent)

	if !strings.Contains(lockStr, "engine_id: \"claude\"") {
		t.Errorf("expected lock file to use claude engine, got: %s", lockPath)
	}

	requiredPermissions := []string{
		"contents: read",
		"issues: read",
	}

	for _, perm := range requiredPermissions {
		if !strings.Contains(lockStr, perm) {
			t.Errorf("expected lock file to contain permission %q", perm)
		}
	}

	// Note: Issue/project write operations are handled via safe-outputs which mint
	// app tokens with appropriate permissions, not direct workflow permissions.

	// Read the generated markdown file to verify the Source comment contains a relative path
	mdContent, readErr := os.ReadFile(orchestratorPath)
	if readErr != nil {
		t.Fatalf("failed to read generated markdown file: %v", readErr)
	}
	mdStr := string(mdContent)

	if !strings.Contains(mdStr, "engine: claude") {
		t.Errorf("expected generated markdown to set engine: claude")
	}

	// Verify that the Source comment exists and contains a relative path (not absolute)
	if !strings.Contains(mdStr, "<!-- Source:") {
		t.Errorf("expected generated markdown to contain Source comment")
	}

	// Extract the source path from the comment
	if strings.Contains(mdStr, "<!-- Source:") {
		// Find the source path in the comment
		startIdx := strings.Index(mdStr, "<!-- Source:")
		endIdx := strings.Index(mdStr[startIdx:], "-->")
		if endIdx > 0 {
			sourceComment := mdStr[startIdx : startIdx+endIdx]
			// Verify it's not an absolute path (no leading / or drive letter)
			if strings.Contains(sourceComment, "<!-- Source: /") ||
				(len(sourceComment) > 15 && sourceComment[13] == ':' && sourceComment[14] == '\\') {
				t.Errorf("Source comment contains absolute path: %q", sourceComment)
			}
			// Verify it contains the campaign filename
			if !strings.Contains(sourceComment, "test-campaign.campaign.md") {
				t.Errorf("Source comment doesn't contain expected filename: %q", sourceComment)
			}
		}
	}
}

// TestCampaignSourceCommentStability verifies that the source path in the generated
// campaign orchestrator is stable regardless of the current working directory
func TestCampaignSourceCommentStability(t *testing.T) {
	// Create a temporary git repository structure with .github directory
	tmpDir := t.TempDir()
	githubDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		t.Fatalf("failed to create .github/workflows directory: %v", err)
	}

	// Create a campaign spec in the .github/workflows directory
	campaignSpecPath := filepath.Join(githubDir, "test-campaign.campaign.md")

	spec := &campaign.CampaignSpec{
		ID:          "test-campaign",
		Name:        "Test Campaign",
		Description: "A test campaign for path stability",
		Workflows:   []string{"example-workflow"},
		MemoryPaths: []string{"memory/campaigns/test-campaign/**"},
	}

	compiler := workflow.NewCompiler(false, "", GetVersion())
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(false)
	compiler.SetStrictMode(false)

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		// Restore original working directory
		if err := os.Chdir(originalWd); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()

	// Test 1: Generate from the tmp directory root
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to tmp directory: %v", err)
	}

	orchestratorPath1, err := generateAndCompileCampaignOrchestrator(GenerateCampaignOrchestratorOptions{
		Compiler:             compiler,
		Spec:                 spec,
		CampaignSpecPath:     campaignSpecPath,
		Verbose:              false,
		NoEmit:               false,
		RunZizmorPerFile:     false,
		RunPoutinePerFile:    false,
		RunActionlintPerFile: false,
		Strict:               false,
		ValidateActionSHAs:   false,
	})
	if err != nil {
		t.Fatalf("first generation error: %v", err)
	}

	mdContent1, err := os.ReadFile(orchestratorPath1)
	if err != nil {
		t.Fatalf("failed to read first generated file: %v", err)
	}
	sourcePath1 := extractSourcePath(t, string(mdContent1))

	// Test 2: Generate from a subdirectory
	pkgDir := filepath.Join(tmpDir, "pkg", "cli")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg/cli directory: %v", err)
	}
	if err := os.Chdir(pkgDir); err != nil {
		t.Fatalf("failed to change to pkg/cli directory: %v", err)
	}

	// Remove the previously generated file to regenerate it
	if err := os.Remove(orchestratorPath1); err != nil {
		t.Fatalf("failed to remove first generated file: %v", err)
	}

	orchestratorPath2, err := generateAndCompileCampaignOrchestrator(GenerateCampaignOrchestratorOptions{
		Compiler:             compiler,
		Spec:                 spec,
		CampaignSpecPath:     campaignSpecPath,
		Verbose:              false,
		NoEmit:               false,
		RunZizmorPerFile:     false,
		RunPoutinePerFile:    false,
		RunActionlintPerFile: false,
		Strict:               false,
		ValidateActionSHAs:   false,
	})
	if err != nil {
		t.Fatalf("second generation error: %v", err)
	}

	mdContent2, err := os.ReadFile(orchestratorPath2)
	if err != nil {
		t.Fatalf("failed to read second generated file: %v", err)
	}
	sourcePath2 := extractSourcePath(t, string(mdContent2))

	// Verify both paths are identical and stable
	if sourcePath1 != sourcePath2 {
		t.Errorf("source paths differ based on working directory:\n  from root: %q\n  from subdir: %q", sourcePath1, sourcePath2)
	}

	// Verify the path is normalized to .github/workflows/...
	expectedPath := ".github/workflows/test-campaign.campaign.md"
	if sourcePath1 != expectedPath {
		t.Errorf("expected stable path %q, got %q", expectedPath, sourcePath1)
	}
}

// extractSourcePath extracts the source path from a markdown file's source comment
func extractSourcePath(t *testing.T, content string) string {
	t.Helper()

	startMarker := "<!-- Source: "
	endMarker := " -->"

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		t.Fatalf("source comment not found in content")
	}

	startIdx += len(startMarker)
	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		t.Fatalf("source comment end marker not found")
	}

	return strings.TrimSpace(content[startIdx : startIdx+endIdx])
}

// TestCampaignOrchestratorGitHubToken verifies that when a campaign spec includes
// a project-github-token field, it is properly serialized into the generated
// .g.campaign.md file's safe-outputs configuration
func TestCampaignOrchestratorGitHubToken(t *testing.T) {
	tmpDir := t.TempDir()
	campaignSpecPath := filepath.Join(tmpDir, "test-campaign-with-token.campaign.md")

	// Test case 1: Campaign with custom GitHub token
	t.Run("with custom token", func(t *testing.T) {
		spec := &campaign.CampaignSpec{
			ID:                 "test-campaign-with-token",
			Name:               "Test Campaign With Token",
			Description:        "A test campaign with custom GitHub token",
			Workflows:          []string{"example-workflow"},
			MemoryPaths:        []string{"memory/campaigns/test-campaign-with-token/**"},
			ProjectGitHubToken: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
		}

		compiler := workflow.NewCompiler(false, "", GetVersion())
		compiler.SetSkipValidation(true)
		compiler.SetNoEmit(false)
		compiler.SetStrictMode(false)

		orchestratorPath, err := generateAndCompileCampaignOrchestrator(GenerateCampaignOrchestratorOptions{
			Compiler:             compiler,
			Spec:                 spec,
			CampaignSpecPath:     campaignSpecPath,
			Verbose:              false,
			NoEmit:               false,
			RunZizmorPerFile:     false,
			RunPoutinePerFile:    false,
			RunActionlintPerFile: false,
			Strict:               false,
			ValidateActionSHAs:   false,
		})
		if err != nil {
			t.Fatalf("generateAndCompileCampaignOrchestrator() error: %v", err)
		}

		// Read the generated markdown file
		mdContent, err := os.ReadFile(orchestratorPath)
		if err != nil {
			t.Fatalf("failed to read generated markdown: %v", err)
		}
		mdStr := string(mdContent)

		// Verify the github-token is present in the safe-outputs configuration
		if !strings.Contains(mdStr, "github-token:") {
			t.Errorf("expected generated markdown to contain 'github-token:' field")
		}

		if !strings.Contains(mdStr, "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}") {
			t.Errorf("expected generated markdown to contain the token expression")
		}

		// Verify the safe-outputs structure
		if !strings.Contains(mdStr, "safe-outputs:") {
			t.Errorf("expected generated markdown to contain 'safe-outputs:' section")
		}

		if !strings.Contains(mdStr, "update-project:") {
			t.Errorf("expected generated markdown to contain 'update-project:' section")
		}
	})

	// Test case 2: Campaign without custom GitHub token
	t.Run("without custom token", func(t *testing.T) {
		spec := &campaign.CampaignSpec{
			ID:          "test-campaign-no-token",
			Name:        "Test Campaign Without Token",
			Description: "A test campaign without custom GitHub token",
			Workflows:   []string{"example-workflow"},
			MemoryPaths: []string{"memory/campaigns/test-campaign-no-token/**"},
			// ProjectGitHubToken is intentionally omitted
		}

		compiler := workflow.NewCompiler(false, "", GetVersion())
		compiler.SetSkipValidation(true)
		compiler.SetNoEmit(false)
		compiler.SetStrictMode(false)

		orchestratorPath, err := generateAndCompileCampaignOrchestrator(GenerateCampaignOrchestratorOptions{
			Compiler:             compiler,
			Spec:                 spec,
			CampaignSpecPath:     filepath.Join(tmpDir, "test-campaign-no-token.campaign.md"),
			Verbose:              false,
			NoEmit:               false,
			RunZizmorPerFile:     false,
			RunPoutinePerFile:    false,
			RunActionlintPerFile: false,
			Strict:               false,
			ValidateActionSHAs:   false,
		})
		if err != nil {
			t.Fatalf("generateAndCompileCampaignOrchestrator() error: %v", err)
		}

		// Read the generated markdown file
		mdContent, err := os.ReadFile(orchestratorPath)
		if err != nil {
			t.Fatalf("failed to read generated markdown: %v", err)
		}
		mdStr := string(mdContent)

		// Verify the github-token is NOT present in safe-outputs when not configured
		// Note: The discovery step may have github-token in its `with:` section with a fallback,
		// but the safe-outputs section should not have a custom github-token field
		safeOutputsStart := strings.Index(mdStr, "safe-outputs:")
		// Find the end of safe-outputs section (before runs-on, tools, or steps)
		safeOutputsContent := mdStr[safeOutputsStart:]
		boundaries := []string{"\nruns-on:", "\ntools:", "\nsteps:"}
		safeOutputsEnd := len(safeOutputsContent)
		for _, boundary := range boundaries {
			if idx := strings.Index(safeOutputsContent, boundary); idx > 0 && idx < safeOutputsEnd {
				safeOutputsEnd = idx
			}
		}

		if safeOutputsStart >= 0 && safeOutputsEnd > 0 {
			safeOutputsSection := safeOutputsContent[:safeOutputsEnd]
			if strings.Contains(safeOutputsSection, "github-token:") {
				t.Errorf("expected safe-outputs section to NOT contain 'github-token:' field when not configured, got:\n%s", safeOutputsSection)
			}
		}

		// But safe-outputs and update-project should still be present
		if !strings.Contains(mdStr, "safe-outputs:") {
			t.Errorf("expected generated markdown to contain 'safe-outputs:' section")
		}

		if !strings.Contains(mdStr, "update-project:") {
			t.Errorf("expected generated markdown to contain 'update-project:' section")
		}
	})
}
