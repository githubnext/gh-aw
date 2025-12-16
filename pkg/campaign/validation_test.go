package campaign

import (
	"strings"
	"testing"
)

func TestValidateSpec_ValidSpec(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Version:      "v1",
		State:        "active",
		Workflows:    []string{"workflow1", "workflow2"},
		TrackerLabel: "campaign:test",
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems, got: %v", problems)
	}
}

func TestValidateSpec_MissingID(t *testing.T) {
	spec := &CampaignSpec{
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "campaign:test",
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for missing ID")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "id is required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ID validation problem, got: %v", problems)
	}
}

func TestValidateSpec_InvalidIDCharacters(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "Test_Campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "campaign:test",
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid ID characters")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "lowercase letters, digits, and hyphens") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ID character validation problem, got: %v", problems)
	}
}

func TestValidateSpec_MissingName(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "campaign:test",
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for missing name")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "name should be provided") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected name validation problem, got: %v", problems)
	}
}

func TestValidateSpec_MissingWorkflows(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		TrackerLabel: "campaign:test",
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for missing workflows")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "workflows should list at least one workflow") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected workflows validation problem, got: %v", problems)
	}
}

func TestValidateSpec_MissingTrackerLabel(t *testing.T) {
	spec := &CampaignSpec{
		ID:         "test-campaign",
		Name:       "Test Campaign",
		ProjectURL: "https://github.com/orgs/org/projects/1",
		Workflows:  []string{"workflow1"},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for missing tracker label")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "tracker-label should be set") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected tracker label validation problem, got: %v", problems)
	}
}

func TestValidateSpec_InvalidTrackerLabelFormat(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "no-colon-here",
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid tracker label format")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "tracker-label should follow a namespaced pattern") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected tracker label format validation problem, got: %v", problems)
	}
}

func TestValidateSpec_InvalidState(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "campaign:test",
		State:        "invalid-state",
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid state")
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

func TestValidateSpec_ValidStates(t *testing.T) {
	validStates := []string{"planned", "active", "paused", "completed", "archived"}

	for _, state := range validStates {
		spec := &CampaignSpec{
			ID:           "test-campaign",
			Name:         "Test Campaign",
			ProjectURL:   "https://github.com/orgs/org/projects/1",
			Workflows:    []string{"workflow1"},
			TrackerLabel: "campaign:test",
			State:        state,
		}

		problems := ValidateSpec(spec)
		if len(problems) != 0 {
			t.Errorf("Expected no validation problems for state '%s', got: %v", state, problems)
		}
	}
}

func TestValidateSpec_VersionDefault(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "campaign:test",
	}

	_ = ValidateSpec(spec)

	if spec.Version != "v1" {
		t.Errorf("Expected version to default to 'v1', got '%s'", spec.Version)
	}
}

func TestValidateSpec_RiskLevel(t *testing.T) {
	validRiskLevels := []string{"low", "medium", "high"}

	for _, riskLevel := range validRiskLevels {
		spec := &CampaignSpec{
			ID:           "test-campaign",
			Name:         "Test Campaign",
			ProjectURL:   "https://github.com/orgs/org/projects/1",
			Workflows:    []string{"workflow1"},
			TrackerLabel: "campaign:test",
			RiskLevel:    riskLevel,
		}

		problems := ValidateSpec(spec)
		// Risk level validation is currently not enforced beyond schema
		// This test ensures the field is accepted
		if len(problems) != 0 {
			t.Errorf("Expected no validation problems for risk level '%s', got: %v", riskLevel, problems)
		}
	}
}

func TestValidateSpec_WithApprovalPolicy(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		Workflows:    []string{"workflow1"},
		TrackerLabel: "campaign:test",
		ApprovalPolicy: &CampaignApprovalPolicy{
			RequiredApprovals: 2,
			RequiredRoles:     []string{"admin", "security"},
			ChangeControl:     true,
		},
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems with approval policy, got: %v", problems)
	}
}

func TestValidateSpec_CompleteSpec(t *testing.T) {
	spec := &CampaignSpec{
		ID:                 "complete-campaign",
		Name:               "Complete Campaign",
		Description:        "A complete campaign spec for testing",
		ProjectURL:         "https://github.com/orgs/org/projects/1",
		Version:            "v1",
		Workflows:          []string{"workflow1", "workflow2"},
		MemoryPaths:        []string{"memory/campaigns/complete/**"},
		MetricsGlob:        "memory/campaigns/complete-*.json",
		Owners:             []string{"owner1", "owner2"},
		ExecutiveSponsors:  []string{"sponsor1"},
		RiskLevel:          "medium",
		TrackerLabel:       "campaign:complete",
		State:              "active",
		Tags:               []string{"security", "compliance"},
		AllowedSafeOutputs: []string{"create-issue", "create-pull-request"},
		ApprovalPolicy: &CampaignApprovalPolicy{
			RequiredApprovals: 3,
			RequiredRoles:     []string{"admin", "security", "compliance"},
			ChangeControl:     true,
		},
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems for complete spec, got: %v", problems)
	}
}
