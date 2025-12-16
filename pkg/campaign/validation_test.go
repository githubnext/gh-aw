package campaign

import (
	"os"
	"path/filepath"
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

func TestValidateWorkflowsExist_AllWorkflowsFound(t *testing.T) {
	// Create a temporary directory for test workflows
	tmpDir := t.TempDir()

	// Create test workflow files
	if err := os.WriteFile(filepath.Join(tmpDir, "workflow1.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "workflow2.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	spec := &CampaignSpec{
		ID:        "test-campaign",
		Workflows: []string{"workflow1", "workflow2"},
	}

	problems := ValidateWorkflowsExist(spec, tmpDir)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems when workflows exist, got: %v", problems)
	}
}

func TestValidateWorkflowsExist_WorkflowNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	spec := &CampaignSpec{
		ID:        "test-campaign",
		Workflows: []string{"nonexistent-workflow"},
	}

	problems := ValidateWorkflowsExist(spec, tmpDir)
	if len(problems) != 1 {
		t.Fatalf("Expected 1 validation problem, got %d: %v", len(problems), problems)
	}

	problem := problems[0]
	
	// Check that the error message contains key troubleshooting information
	if !strings.Contains(problem, "workflow 'nonexistent-workflow' not found") {
		t.Errorf("Error message should mention the workflow name, got: %s", problem)
	}
	if !strings.Contains(problem, "Troubleshooting:") {
		t.Errorf("Error message should contain 'Troubleshooting:' section, got: %s", problem)
	}
	if !strings.Contains(problem, "Check config paths") {
		t.Errorf("Error message should mention config paths, got: %s", problem)
	}
	if !strings.Contains(problem, "Check file existence") {
		t.Errorf("Error message should mention file existence, got: %s", problem)
	}
	if !strings.Contains(problem, "Relative vs. absolute paths") {
		t.Errorf("Error message should mention relative vs absolute paths, got: %s", problem)
	}
	if !strings.Contains(problem, tmpDir) {
		t.Errorf("Error message should include the workflow directory path, got: %s", problem)
	}
}

func TestValidateWorkflowsExist_LockFileWithoutSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only lock file, not source
	if err := os.WriteFile(filepath.Join(tmpDir, "workflow1.lock.yml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	spec := &CampaignSpec{
		ID:        "test-campaign",
		Workflows: []string{"workflow1"},
	}

	problems := ValidateWorkflowsExist(spec, tmpDir)
	if len(problems) != 1 {
		t.Fatalf("Expected 1 validation problem, got %d: %v", len(problems), problems)
	}

	problem := problems[0]
	
	// Check that the error message contains appropriate troubleshooting for missing source
	if !strings.Contains(problem, "has lock file but missing source .md file") {
		t.Errorf("Error message should mention missing source file, got: %s", problem)
	}
	if !strings.Contains(problem, "Troubleshooting:") {
		t.Errorf("Error message should contain 'Troubleshooting:' section, got: %s", problem)
	}
	if !strings.Contains(problem, "Check file existence") {
		t.Errorf("Error message should mention file existence, got: %s", problem)
	}
	if !strings.Contains(problem, ".lock.yml file is generated from the .md source") {
		t.Errorf("Error message should explain lock file generation, got: %s", problem)
	}
}

func TestValidateWorkflowsExist_MultipleProblems(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid workflow
	if err := os.WriteFile(filepath.Join(tmpDir, "valid.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	// Create one workflow with only lock file
	if err := os.WriteFile(filepath.Join(tmpDir, "missing-source.lock.yml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	spec := &CampaignSpec{
		ID:        "test-campaign",
		Workflows: []string{"valid", "missing-source", "completely-missing"},
	}

	problems := ValidateWorkflowsExist(spec, tmpDir)
	
	// Should have 2 problems: one for missing source, one for completely missing
	if len(problems) != 2 {
		t.Fatalf("Expected 2 validation problems, got %d: %v", len(problems), problems)
	}

	// Check that we got the right types of errors
	hasLockWithoutSource := false
	hasCompletelyMissing := false

	for _, problem := range problems {
		if strings.Contains(problem, "missing-source") && strings.Contains(problem, "has lock file but missing source") {
			hasLockWithoutSource = true
		}
		if strings.Contains(problem, "completely-missing") && strings.Contains(problem, "not found") {
			hasCompletelyMissing = true
		}
	}

	if !hasLockWithoutSource {
		t.Errorf("Expected problem for workflow with lock but no source")
	}
	if !hasCompletelyMissing {
		t.Errorf("Expected problem for completely missing workflow")
	}
}
