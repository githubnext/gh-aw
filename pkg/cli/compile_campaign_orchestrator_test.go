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
		ID:           "test-campaign",
		Name:         "Test Campaign",
		Description:  "A test campaign",
		Workflows:    []string{"example-workflow"},
		TrackerLabel: "campaign:test-campaign",
		MemoryPaths:  []string{"memory/campaigns/test-campaign-*/**"},
	}

	compiler := workflow.NewCompiler(false, "", GetVersion())
	compiler.SetSkipValidation(true)
	compiler.SetNoEmit(false)
	compiler.SetStrictMode(false)

	orchestratorPath, err := generateAndCompileCampaignOrchestrator(
		compiler,
		spec,
		campaignSpecPath,
		false, // verbose
		false, // noEmit
		false, // zizmor
		false, // poutine
		false, // actionlint
		false, // strict
		false, // validateActionSHAs
	)
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

	lockPath := strings.TrimSuffix(orchestratorPath, ".md") + ".lock.yml"
	if _, statErr := os.Stat(lockPath); statErr != nil {
		t.Fatalf("expected orchestrator lock file to exist, stat error: %v", statErr)
	}

	// Verify that the generated orchestrator has the required permissions
	lockContent, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		t.Fatalf("failed to read lock file: %v", readErr)
	}
	lockStr := string(lockContent)

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
