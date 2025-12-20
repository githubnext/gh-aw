package workflow

import (
	"testing"
)

func TestGetSafeOutputEnvManifest(t *testing.T) {
	manifest := GetSafeOutputEnvManifest()

	// Verify manifest contains expected job types
	expectedJobTypes := []string{
		"create_pull_request",
		"add_comment",
		"create_issue",
		"create_discussion",
		"add_labels",
		"missing_tool",
		"noop",
		"create_agent_task",
		"create_code_scanning_alert",
		"create_pr_review_comment",
		"push_to_pull_request_branch",
	}

	for _, jobType := range expectedJobTypes {
		if _, exists := manifest[jobType]; !exists {
			t.Errorf("Expected job type %s to exist in manifest", jobType)
		}
	}

	// Verify manifest has correct number of job types
	if len(manifest) != len(expectedJobTypes) {
		t.Errorf("Expected %d job types in manifest, got %d", len(expectedJobTypes), len(manifest))
	}
}

func TestManifestCommonEnvVars(t *testing.T) {
	manifest := GetSafeOutputEnvManifest()

	// Common variables that should be present in all job types
	commonVars := []string{
		"GH_AW_WORKFLOW_NAME",
		"GITHUB_TOKEN",
	}

	for jobType, jobManifest := range manifest {
		envVarMap := make(map[string]bool)
		for _, envVar := range jobManifest.EnvVars {
			envVarMap[envVar.Name] = true
		}

		for _, commonVar := range commonVars {
			if !envVarMap[commonVar] {
				t.Errorf("Job type %s missing common variable %s", jobType, commonVar)
			}
		}
	}
}

func TestCreatePullRequestRequiredVars(t *testing.T) {
	requiredVars, err := GetRequiredEnvVarsForJobType("create_pull_request")
	if err != nil {
		t.Fatalf("Failed to get required vars for create_pull_request: %v", err)
	}

	expectedRequired := map[string]bool{
		"GH_AW_WORKFLOW_NAME": true,
		"GITHUB_TOKEN":        true,
		"GH_AW_WORKFLOW_ID":   true,
		"GH_AW_BASE_BRANCH":   true,
		"GH_AW_AGENT_OUTPUT":  true,
	}

	for _, varName := range requiredVars {
		if !expectedRequired[varName] {
			t.Errorf("Unexpected required variable for create_pull_request: %s", varName)
		}
		delete(expectedRequired, varName)
	}

	// Check that all expected required vars were found
	for varName := range expectedRequired {
		t.Errorf("Missing required variable for create_pull_request: %s", varName)
	}
}

func TestAddCommentRequiredVars(t *testing.T) {
	requiredVars, err := GetRequiredEnvVarsForJobType("add_comment")
	if err != nil {
		t.Fatalf("Failed to get required vars for add_comment: %v", err)
	}

	// add_comment only requires common variables
	expectedRequired := map[string]bool{
		"GH_AW_WORKFLOW_NAME": true,
		"GITHUB_TOKEN":        true,
	}

	for _, varName := range requiredVars {
		if !expectedRequired[varName] {
			t.Errorf("Unexpected required variable for add_comment: %s", varName)
		}
		delete(expectedRequired, varName)
	}

	// Check that all expected required vars were found
	for varName := range expectedRequired {
		t.Errorf("Missing required variable for add_comment: %s", varName)
	}
}

func TestNoopRequiredVars(t *testing.T) {
	requiredVars, err := GetRequiredEnvVarsForJobType("noop")
	if err != nil {
		t.Fatalf("Failed to get required vars for noop: %v", err)
	}

	// noop only requires common variables
	expectedRequired := map[string]bool{
		"GH_AW_WORKFLOW_NAME": true,
		"GITHUB_TOKEN":        true,
	}

	for _, varName := range requiredVars {
		if !expectedRequired[varName] {
			t.Errorf("Unexpected required variable for noop: %s", varName)
		}
		delete(expectedRequired, varName)
	}

	// Check that all expected required vars were found
	for varName := range expectedRequired {
		t.Errorf("Missing required variable for noop: %s", varName)
	}
}

func TestGetAllEnvVarsForJobType(t *testing.T) {
	testCases := []struct {
		jobType         string
		expectOptional  bool
		optionalVarName string
	}{
		{
			jobType:         "create_pull_request",
			expectOptional:  true,
			optionalVarName: "GH_AW_PR_TITLE_PREFIX",
		},
		{
			jobType:         "add_comment",
			expectOptional:  true,
			optionalVarName: "GH_AW_COMMENT_TARGET",
		},
		{
			jobType:         "create_issue",
			expectOptional:  true,
			optionalVarName: "GH_AW_ISSUE_TITLE_PREFIX",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.jobType, func(t *testing.T) {
			allVars, err := GetAllEnvVarsForJobType(tc.jobType)
			if err != nil {
				t.Fatalf("Failed to get all vars for %s: %v", tc.jobType, err)
			}

			if len(allVars) == 0 {
				t.Errorf("Expected at least some env vars for %s", tc.jobType)
			}

			// Check if optional variable exists
			if tc.expectOptional {
				found := false
				for _, envVar := range allVars {
					if envVar.Name == tc.optionalVarName {
						found = true
						if envVar.Required {
							t.Errorf("Expected %s to be optional for %s", tc.optionalVarName, tc.jobType)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected optional variable %s not found for %s", tc.optionalVarName, tc.jobType)
				}
			}
		})
	}
}

func TestValidateSafeOutputJobEnvVars(t *testing.T) {
	testCases := []struct {
		name         string
		jobType      string
		providedVars map[string]string
		expectMissing []string
	}{
		{
			name:    "create_pull_request with all required vars",
			jobType: "create_pull_request",
			providedVars: map[string]string{
				"GH_AW_WORKFLOW_NAME": "test-workflow",
				"GITHUB_TOKEN":        "ghp_test",
				"GH_AW_WORKFLOW_ID":   "main",
				"GH_AW_BASE_BRANCH":   "main",
				"GH_AW_AGENT_OUTPUT":  "/tmp/output.json",
			},
			expectMissing: nil,
		},
		{
			name:    "create_pull_request missing GH_AW_WORKFLOW_ID",
			jobType: "create_pull_request",
			providedVars: map[string]string{
				"GH_AW_WORKFLOW_NAME": "test-workflow",
				"GITHUB_TOKEN":        "ghp_test",
				"GH_AW_BASE_BRANCH":   "main",
				"GH_AW_AGENT_OUTPUT":  "/tmp/output.json",
			},
			expectMissing: []string{"GH_AW_WORKFLOW_ID"},
		},
		{
			name:         "create_pull_request missing all required vars",
			jobType:      "create_pull_request",
			providedVars: map[string]string{},
			expectMissing: []string{
				"GH_AW_WORKFLOW_NAME",
				"GITHUB_TOKEN",
				"GH_AW_WORKFLOW_ID",
				"GH_AW_BASE_BRANCH",
				"GH_AW_AGENT_OUTPUT",
			},
		},
		{
			name:    "noop with all required vars",
			jobType: "noop",
			providedVars: map[string]string{
				"GH_AW_WORKFLOW_NAME": "test-workflow",
				"GITHUB_TOKEN":        "ghp_test",
			},
			expectMissing: nil,
		},
		{
			name:    "noop missing GITHUB_TOKEN",
			jobType: "noop",
			providedVars: map[string]string{
				"GH_AW_WORKFLOW_NAME": "test-workflow",
			},
			expectMissing: []string{"GITHUB_TOKEN"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			missing := ValidateSafeOutputJobEnvVars(tc.jobType, tc.providedVars)

			if len(tc.expectMissing) == 0 && len(missing) > 0 {
				t.Errorf("Expected no missing vars, got: %v", missing)
			}

			if len(tc.expectMissing) > 0 {
				missingMap := make(map[string]bool)
				for _, m := range missing {
					missingMap[m] = true
				}

				for _, expected := range tc.expectMissing {
					if !missingMap[expected] {
						t.Errorf("Expected missing var %s not found in result", expected)
					}
				}
			}
		})
	}
}

func TestGetRequiredEnvVarsForUnknownJobType(t *testing.T) {
	_, err := GetRequiredEnvVarsForJobType("unknown_job_type")
	if err == nil {
		t.Error("Expected error for unknown job type, got nil")
	}
}

func TestGetAllEnvVarsForUnknownJobType(t *testing.T) {
	_, err := GetAllEnvVarsForJobType("unknown_job_type")
	if err == nil {
		t.Error("Expected error for unknown job type, got nil")
	}
}

func TestGetSupportedSafeOutputJobTypes(t *testing.T) {
	jobTypes := GetSupportedSafeOutputJobTypes()

	if len(jobTypes) == 0 {
		t.Error("Expected at least one job type")
	}

	// Verify known job types are included
	knownTypes := []string{
		"create_pull_request",
		"add_comment",
		"create_issue",
		"noop",
	}

	jobTypeMap := make(map[string]bool)
	for _, jt := range jobTypes {
		jobTypeMap[jt] = true
	}

	for _, known := range knownTypes {
		if !jobTypeMap[known] {
			t.Errorf("Expected known job type %s to be in supported list", known)
		}
	}
}

func TestManifestEnvVarDescriptions(t *testing.T) {
	manifest := GetSafeOutputEnvManifest()

	// Verify all env vars have descriptions
	for jobType, jobManifest := range manifest {
		for _, envVar := range jobManifest.EnvVars {
			if envVar.Description == "" {
				t.Errorf("Job type %s, env var %s missing description", jobType, envVar.Name)
			}
		}
	}
}

func TestManifestJobDescriptions(t *testing.T) {
	manifest := GetSafeOutputEnvManifest()

	// Verify all jobs have descriptions
	for jobType, jobManifest := range manifest {
		if jobManifest.Description == "" {
			t.Errorf("Job type %s missing description", jobType)
		}
	}
}

func TestCreateIssueRequiredVars(t *testing.T) {
	requiredVars, err := GetRequiredEnvVarsForJobType("create_issue")
	if err != nil {
		t.Fatalf("Failed to get required vars for create_issue: %v", err)
	}

	expectedRequired := map[string]bool{
		"GH_AW_WORKFLOW_NAME": true,
		"GITHUB_TOKEN":        true,
		"GH_AW_AGENT_OUTPUT":  true,
	}

	for _, varName := range requiredVars {
		if !expectedRequired[varName] {
			t.Errorf("Unexpected required variable for create_issue: %s", varName)
		}
		delete(expectedRequired, varName)
	}

	// Check that all expected required vars were found
	for varName := range expectedRequired {
		t.Errorf("Missing required variable for create_issue: %s", varName)
	}
}

func TestMissingToolRequiredVars(t *testing.T) {
	requiredVars, err := GetRequiredEnvVarsForJobType("missing_tool")
	if err != nil {
		t.Fatalf("Failed to get required vars for missing_tool: %v", err)
	}

	expectedRequired := map[string]bool{
		"GH_AW_WORKFLOW_NAME": true,
		"GITHUB_TOKEN":        true,
		"GH_AW_AGENT_OUTPUT":  true,
	}

	for _, varName := range requiredVars {
		if !expectedRequired[varName] {
			t.Errorf("Unexpected required variable for missing_tool: %s", varName)
		}
		delete(expectedRequired, varName)
	}

	// Check that all expected required vars were found
	for varName := range expectedRequired {
		t.Errorf("Missing required variable for missing_tool: %s", varName)
	}
}

func TestManifestDefaultValues(t *testing.T) {
	manifest := GetSafeOutputEnvManifest()

	// Test that default values are set appropriately
	createPR := manifest["create_pull_request"]
	
	var draftVar *SafeOutputEnvVar
	var maxPatchVar *SafeOutputEnvVar
	
	for i, envVar := range createPR.EnvVars {
		if envVar.Name == "GH_AW_PR_DRAFT" {
			draftVar = &createPR.EnvVars[i]
		}
		if envVar.Name == "GH_AW_MAX_PATCH_SIZE" {
			maxPatchVar = &createPR.EnvVars[i]
		}
	}

	if draftVar == nil {
		t.Error("Expected GH_AW_PR_DRAFT to be in create_pull_request manifest")
	} else {
		if draftVar.DefaultValue != "true" {
			t.Errorf("Expected GH_AW_PR_DRAFT default to be 'true', got %q", draftVar.DefaultValue)
		}
	}

	if maxPatchVar == nil {
		t.Error("Expected GH_AW_MAX_PATCH_SIZE to be in create_pull_request manifest")
	} else {
		if maxPatchVar.DefaultValue != "1024" {
			t.Errorf("Expected GH_AW_MAX_PATCH_SIZE default to be '1024', got %q", maxPatchVar.DefaultValue)
		}
	}
}
