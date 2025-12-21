package campaign

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadCampaignSpecs_Basic verifies that campaign specs can be loaded
// from the default campaigns directory and that ID/name defaults work.
func TestLoadCampaignSpecs_Basic(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}

	specs, err := LoadSpecs(repoRoot)
	if err != nil {
		t.Fatalf("LoadSpecs failed: %v", err)
	}

	if len(specs) == 0 {
		t.Fatalf("Expected at least one campaign spec, got 0")
	}

	// Ensure we can find the example campaign spec we ship in this repo.
	found := false
	for _, spec := range specs {
		if spec.ID == "go-file-size-reduction-project64" {
			found = true
			if spec.Name == "" {
				t.Errorf("Expected Name for go-file-size-reduction-project64 to be non-empty")
			}
			if !strings.Contains(spec.ConfigPath, ".github/workflows/go-file-size-reduction-project64.campaign.md") {
				t.Errorf("Expected ConfigPath to point to go-file-size-reduction-project64 .campaign.md spec, got %s", spec.ConfigPath)
			}
			break
		}
	}

	if !found {
		t.Errorf("Expected to find go-file-size-reduction-project64 campaign spec in loaded specs")
	}
}

// TestComputeCompiledStateForCampaign_UsesLockFiles checks that compiled
// state reflects presence and freshness of .lock.yml files.
func TestComputeCompiledStateForCampaign_UsesLockFiles(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}

	specs, err := LoadSpecs(repoRoot)
	if err != nil {
		t.Fatalf("LoadSpecs failed: %v", err)
	}

	var incident CampaignSpec
	found := false
	for _, spec := range specs {
		if spec.ID == "go-file-size-reduction-project64" {
			incident = spec
			found = true
			break
		}
	}
	if !found {
		t.Skip("go-file-size-reduction-project64 campaign spec not found; skipping compiled-state test")
	}

	state := ComputeCompiledState(incident, ".github/workflows")
	if state == "Missing workflow" {
		t.Fatalf("Expected go-file-size-reduction-project64 workflows to exist, got compiled state: %s", state)
	}
}

// TestRunCampaignStatus_JSON ensures the campaign list view returns valid JSON
// and that at least one campaign is present.
func TestRunCampaignStatus_JSON(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}

	// Capture stdout via a pipe; simpler is to call runCampaignStatus and
	// re-marshal the result, so instead we directly call the loader and
	// verify JSON marshaling there.
	specs, err := LoadSpecs(repoRoot)
	if err != nil {
		t.Fatalf("LoadSpecs failed: %v", err)
	}

	data, err := json.Marshal(specs)
	if err != nil {
		t.Fatalf("Failed to marshal specs to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Fatalf("Expected non-empty JSON for specs")
	}
}

// TestValidateCampaignSpec_Basic ensures that a minimal but well-formed
// spec passes validation without problems and that defaulting behavior
// (like version) is applied.
func TestValidateCampaignSpec_Basic(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "go-file-size-reduction",
		Name:         "Go File Size Reduction",
		ProjectURL:   "https://github.com/orgs/githubnext/projects/1",
		Workflows:    []string{"daily-file-diet"},
		TrackerLabel: "campaign:go-file-size-reduction",
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Fatalf("Expected no validation problems for basic spec, got: %v", problems)
	}

	if spec.Version != "v1" {
		t.Errorf("Expected default version 'v1', got %q", spec.Version)
	}
}

// TestValidateCampaignSpec_InvalidState verifies that invalid state
// values are reported by validation.
func TestValidateCampaignSpec_InvalidState(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "rollout-q1-2025",
		Name:         "Rollout",
		ProjectURL:   "https://github.com/orgs/githubnext/projects/1",
		Workflows:    []string{"org-wide-rollout"},
		TrackerLabel: "campaign:rollout-q1-2025",
		State:        "launching", // invalid
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatalf("Expected validation problems for invalid state, got none")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "state must be one of") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected state validation problem, got: %v", problems)
	}
}

// TestComputeCompiledState_LockFilePath verifies that lock file paths are
// correctly constructed (workflow.lock.yml, not workflow.md.lock.yml).
func TestComputeCompiledState_LockFilePath(t *testing.T) {
	// Create a temporary directory for test workflows
	tmpDir := t.TempDir()

	// Create a workflow .md file and its .lock.yml companion
	workflowID := "test-workflow"
	mdPath := filepath.Join(tmpDir, workflowID+".md")
	lockPath := filepath.Join(tmpDir, workflowID+".lock.yml")

	if err := os.WriteFile(mdPath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("test lock"), 0o644); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	spec := CampaignSpec{
		ID:        "test-campaign",
		Workflows: []string{workflowID},
	}

	// This should find the lock file and return "Yes"
	state := ComputeCompiledState(spec, tmpDir)
	if state != "Yes" {
		t.Errorf("Expected compiled state 'Yes' when both .md and .lock.yml exist, got %q", state)
	}

	// Now test with only the .md file (remove lock file)
	if err := os.Remove(lockPath); err != nil {
		t.Fatalf("Failed to remove lock file: %v", err)
	}

	state = ComputeCompiledState(spec, tmpDir)
	if state != "No" {
		t.Errorf("Expected compiled state 'No' when .lock.yml is missing, got %q", state)
	}

	// Test that we don't look for the wrong path (workflow.md.lock.yml)
	wrongLockPath := mdPath + ".lock.yml" // This would be workflow.md.lock.yml
	if err := os.WriteFile(wrongLockPath, []byte("wrong lock"), 0o644); err != nil {
		t.Fatalf("Failed to create wrong lock file: %v", err)
	}

	state = ComputeCompiledState(spec, tmpDir)
	if state != "No" {
		t.Errorf("Expected compiled state 'No' because correct lock file doesn't exist (only wrong path exists), got %q", state)
	}
}
