package campaign

import (
	"strings"
	"testing"
)

func TestValidateSpec_ValidSpec(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		Version:        "v1",
		State:          "active",
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1", "workflow2"},
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems, got: %v", problems)
	}
}

func TestValidateSpec_MissingID(t *testing.T) {
	spec := &CampaignSpec{
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
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
		ID:             "Test_Campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
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
		ID:             "test-campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
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
		AllowedRepos: []string{"org/repo1"},
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

func TestValidateSpec_InvalidState(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		State:          "invalid-state",
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
			ID:             "test-campaign",
			Name:           "Test Campaign",
			ProjectURL:     "https://github.com/orgs/org/projects/1",
			AllowedRepos:   []string{"org/repo1"},
			DiscoveryRepos: []string{"test/repo"},
			Workflows:      []string{"workflow1"},
			State:          state,
		}

		problems := ValidateSpec(spec)
		if len(problems) != 0 {
			t.Errorf("Expected no validation problems for state '%s', got: %v", state, problems)
		}
	}
}

func TestValidateSpec_VersionDefault(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
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
			ID:             "test-campaign",
			Name:           "Test Campaign",
			ProjectURL:     "https://github.com/orgs/org/projects/1",
			AllowedRepos:   []string{"org/repo1"},
			DiscoveryRepos: []string{"test/repo"},
			Workflows:      []string{"workflow1"},
			RiskLevel:      riskLevel,
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
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
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
		AllowedRepos:       []string{"org/repo1"},
		Version:            "v1",
		DiscoveryRepos:     []string{"org/repo1"},
		Workflows:          []string{"workflow1", "workflow2"},
		MemoryPaths:        []string{"memory/campaigns/complete/**"},
		MetricsGlob:        "memory/campaigns/complete-*.json",
		Owners:             []string{"owner1", "owner2"},
		ExecutiveSponsors:  []string{"sponsor1"},
		RiskLevel:          "medium",
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

func TestValidateSpec_ObjectiveWithoutKPIs(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		Objective:      "Improve CI stability",
		// KPIs intentionally omitted
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for objective without kpis")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "kpis should include at least one KPI") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected objective/kpis coupling validation problem, got: %v", problems)
	}
}

func TestValidateSpec_KPIsWithoutObjective(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		KPIs: []CampaignKPI{
			{
				Name:           "Build success rate",
				Priority:       "primary",
				Baseline:       0.8,
				Target:         0.95,
				TimeWindowDays: 7,
			},
		},
		// Objective intentionally omitted
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for kpis without objective")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "objective should be set when kpis") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected objective/kpis coupling validation problem, got: %v", problems)
	}
}

func TestValidateSpec_KPIsMultipleWithoutPrimary(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		Objective:      "Improve delivery",
		KPIs: []CampaignKPI{
			{Name: "PR cycle time", Priority: "supporting", Baseline: 10, Target: 7, TimeWindowDays: 30},
			{Name: "Open PRs", Priority: "supporting", Baseline: 20, Target: 10, TimeWindowDays: 30},
		},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for multiple KPIs without a primary")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "exactly one primary KPI") && strings.Contains(p, "priority: primary") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected primary KPI validation problem, got: %v", problems)
	}
}

func TestValidateSpec_KPIsMultipleWithMultiplePrimary(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		Objective:      "Improve delivery",
		KPIs: []CampaignKPI{
			{Name: "Build success rate", Priority: "primary", Baseline: 0.8, Target: 0.95, TimeWindowDays: 7},
			{Name: "PR cycle time", Priority: "primary", Baseline: 10, Target: 7, TimeWindowDays: 30},
		},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for multiple primary KPIs")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "exactly one primary KPI") && strings.Contains(p, "found 2") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected multiple primary KPI validation problem with count, got: %v", problems)
	}
}

func TestValidateSpec_SingleKPIOmitsPriorityIsAllowed(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		Objective:      "Improve CI stability",
		KPIs: []CampaignKPI{
			{
				Name: "Build success rate",
				// Priority intentionally omitted; should be implicitly primary.
				Baseline:       0.8,
				Target:         0.95,
				TimeWindowDays: 7,
			},
		},
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems for single KPI with omitted priority, got: %v", problems)
	}
}

func TestValidateSpec_KPIFieldConstraints(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		Objective:      "Improve CI stability",
		KPIs: []CampaignKPI{
			{
				Name:           "Build success rate",
				Priority:       "primary",
				Baseline:       0.8,
				Target:         0.95,
				TimeWindowDays: 0,
				Direction:      "up",
				Source:         "unknown",
			},
		},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid KPI fields")
	}

	expectSubstrings := []string{
		"time-window-days must be >= 1",
		"'increase' or 'decrease'",
		"'ci', 'pull_requests', 'code_security', or 'custom'",
	}
	for _, needle := range expectSubstrings {
		found := false
		for _, p := range problems {
			if strings.Contains(p, needle) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected validation problem containing %q, got: %v", needle, problems)
		}
	}
}

func TestValidateSpec_MissingAllowedReposIsValid(t *testing.T) {
	spec := &CampaignSpec{
		ID:         "test-campaign",
		Name:       "Test Campaign",
		ProjectURL: "https://github.com/orgs/org/projects/1",
		// No Workflows, no DiscoveryRepos - should be valid since no discovery needed
	}

	problems := ValidateSpec(spec)
	// Should have one problem: missing workflows
	if len(problems) != 1 {
		t.Errorf("Expected 1 validation problem (workflows), got: %v", problems)
	}
}

func TestValidateSpec_MissingAllowedReposWithWorkflowsIsInvalid(t *testing.T) {
	spec := &CampaignSpec{
		ID:         "test-campaign",
		Name:       "Test Campaign",
		ProjectURL: "https://github.com/orgs/org/projects/1",
		Workflows:  []string{"workflow1"},
		// DiscoveryRepos intentionally omitted - should fail because workflows need scoping
	}

	problems := ValidateSpec(spec)
	// Should have validation problems for missing scope
	hasDiscoveryScopeError := false
	for _, p := range problems {
		if strings.Contains(p, "campaigns with workflows or tracker-label must specify discovery-repos or discovery-orgs") {
			hasDiscoveryScopeError = true
			break
		}
	}
	if !hasDiscoveryScopeError {
		t.Errorf("Expected validation error for missing scope with workflows, got: %v", problems)
	}
}

func TestValidateSpec_InvalidAllowedReposFormat(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		AllowedRepos:   []string{"invalid-repo-format", "org/repo1"},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid repo format")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "must be in 'owner/repo' format") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected repo format validation problem, got: %v", problems)
	}
}

func TestValidateSpec_EmptyAllowedReposIsValid(t *testing.T) {
	spec := &CampaignSpec{
		ID:           "test-campaign",
		Name:         "Test Campaign",
		ProjectURL:   "https://github.com/orgs/org/projects/1",
		AllowedRepos: []string{}, // Empty list, no workflows
	}

	problems := ValidateSpec(spec)
	// Should have one problem: missing workflows
	if len(problems) != 1 {
		t.Errorf("Expected 1 validation problem (workflows), got: %v", problems)
	}
}

func TestValidateSpec_EmptyAllowedReposWithWorkflowsIsInvalid(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		Workflows:      []string{"workflow1"},
		DiscoveryRepos: []string{}, // Empty list - should fail with workflows
	}

	problems := ValidateSpec(spec)
	// Should have validation problems for missing scope
	hasDiscoveryScopeError := false
	for _, p := range problems {
		if strings.Contains(p, "campaigns with workflows or tracker-label must specify discovery-repos or discovery-orgs") {
			hasDiscoveryScopeError = true
			break
		}
	}
	if !hasDiscoveryScopeError {
		t.Errorf("Expected validation error for empty scope with workflows, got: %v", problems)
	}
}

func TestValidateSpec_ValidAllowedOrgs(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		AllowedRepos:   []string{"org/repo1"},
		AllowedOrgs:    []string{"github", "microsoft"},
		DiscoveryOrgs:  []string{"github", "microsoft"},
	}

	problems := ValidateSpec(spec)
	if len(problems) != 0 {
		t.Errorf("Expected no validation problems with valid discovery-orgs, got: %v", problems)
	}
}

func TestValidateSpec_InvalidAllowedOrgsFormat(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
		AllowedRepos:   []string{"org/repo1"},
		AllowedOrgs:    []string{"github/repo"}, // Invalid - contains slash
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid org format")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "must be an organization name") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected org format validation problem, got: %v", problems)
	}
}

func TestSuggestValidID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase to lowercase",
			input:    "Security-Q1-2025",
			expected: "security-q1-2025",
		},
		{
			name:     "underscores to hyphens",
			input:    "test_campaign_id",
			expected: "test-campaign-id",
		},
		{
			name:     "spaces to hyphens",
			input:    "my campaign id",
			expected: "my-campaign-id",
		},
		{
			name:     "mixed invalid characters",
			input:    "Test Campaign! @2025",
			expected: "test-campaign-2025",
		},
		{
			name:     "multiple hyphens collapsed",
			input:    "test--campaign---id",
			expected: "test-campaign-id",
		},
		{
			name:     "leading and trailing hyphens removed",
			input:    "-test-campaign-",
			expected: "test-campaign",
		},
		{
			name:     "already valid id",
			input:    "valid-campaign-123",
			expected: "valid-campaign-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggestValidID(tt.input)
			if result != tt.expected {
				t.Errorf("suggestValidID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateSpec_InvalidIDWithSuggestion(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "Test Campaign 2025",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for invalid ID")
	}

	// Check that the error message includes a suggestion
	found := false
	for _, p := range problems {
		if strings.Contains(p, "test-campaign-2025") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ID validation problem with suggestion, got: %v", problems)
	}
}

func TestValidateSpec_AllowedReposWildcard(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/*"}, // Invalid - wildcard not allowed
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for wildcard in allowed-repos")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "cannot contain wildcards") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected wildcard validation problem, got: %v", problems)
	}
}

func TestValidateSpec_AllowedOrgsWildcard(t *testing.T) {
	spec := &CampaignSpec{
		ID:             "test-campaign",
		Name:           "Test Campaign",
		ProjectURL:     "https://github.com/orgs/org/projects/1",
		AllowedRepos:   []string{"org/repo1"},
		AllowedOrgs:    []string{"github*"}, // Invalid - wildcard not allowed
		DiscoveryRepos: []string{"test/repo"},
		Workflows:      []string{"workflow1"},
	}

	problems := ValidateSpec(spec)
	if len(problems) == 0 {
		t.Fatal("Expected validation problems for wildcard in allowed-orgs")
	}

	found := false
	for _, p := range problems {
		if strings.Contains(p, "cannot contain wildcards") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected wildcard validation problem for orgs, got: %v", problems)
	}
}
