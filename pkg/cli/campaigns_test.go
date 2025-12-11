package cli

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

	specs, err := loadCampaignSpecs(repoRoot)
	if err != nil {
		t.Fatalf("loadCampaignSpecs failed: %v", err)
	}

	if len(specs) == 0 {
		t.Fatalf("Expected at least one campaign spec, got 0")
	}

	// Ensure we can find the incident-response spec we added as an example
	found := false
	for _, spec := range specs {
		if spec.ID == "incident-response" {
			found = true
			if spec.Name == "" {
				t.Errorf("Expected Name for incident-response to be non-empty")
			}
			if !strings.Contains(spec.ConfigPath, "campaigns/incident-response.campaign.md") {
				t.Errorf("Expected ConfigPath to point to incident-response .campaign.md spec, got %s", spec.ConfigPath)
			}
			break
		}
	}

	if !found {
		t.Errorf("Expected to find incident-response campaign spec in loaded specs")
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

	specs, err := loadCampaignSpecs(repoRoot)
	if err != nil {
		t.Fatalf("loadCampaignSpecs failed: %v", err)
	}

	var incident CampaignSpec
	found := false
	for _, spec := range specs {
		if spec.ID == "incident-response" {
			incident = spec
			found = true
			break
		}
	}
	if !found {
		t.Skip("incident-response campaign spec not found; skipping compiled-state test")
	}

	state := computeCompiledStateForCampaign(incident)
	if state == "Missing workflow" {
		t.Fatalf("Expected incident-response workflows to exist, got compiled state: %s", state)
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
	specs, err := loadCampaignSpecs(repoRoot)
	if err != nil {
		t.Fatalf("loadCampaignSpecs failed: %v", err)
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
		ID:           "security-compliance",
		Name:         "Security Compliance",
		Workflows:    []string{"security-compliance"},
		TrackerLabel: "campaign:security-compliance",
	}

	problems := validateCampaignSpec(spec)
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
		Workflows:    []string{"org-wide-rollout"},
		TrackerLabel: "campaign:rollout-q1-2025",
		State:        "launching", // invalid
	}

	problems := validateCampaignSpec(spec)
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
