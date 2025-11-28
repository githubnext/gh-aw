package workflow

import (
	"strings"
	"testing"
)

func TestLinkSubIssueJobDependencies(t *testing.T) {
	tests := []struct {
		name               string
		createIssueJobName string
		expectedNeeds      []string
		expectTempIDEnvVar bool
	}{
		{
			name:               "No create_issue dependency",
			createIssueJobName: "",
			expectedNeeds:      []string{"main"},
			expectTempIDEnvVar: false,
		},
		{
			name:               "With create_issue dependency",
			createIssueJobName: "create_issue",
			expectedNeeds:      []string{"main", "create_issue"},
			expectTempIDEnvVar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					LinkSubIssue: &LinkSubIssueConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 5,
						},
					},
				},
			}

			job, err := compiler.buildLinkSubIssueJob(
				workflowData,
				"main",
				tt.createIssueJobName,
			)
			if err != nil {
				t.Fatalf("Failed to build link_sub_issue job: %v", err)
			}

			// Check job dependencies (needs)
			if len(job.Needs) != len(tt.expectedNeeds) {
				t.Errorf("Expected %d needs, got %d: %v", len(tt.expectedNeeds), len(job.Needs), job.Needs)
			}
			for _, expectedNeed := range tt.expectedNeeds {
				found := false
				for _, need := range job.Needs {
					if need == expectedNeed {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected need '%s' not found in job.Needs: %v", expectedNeed, job.Needs)
				}
			}

			// Convert steps to string to check for environment variables
			stepsStr := strings.Join(job.Steps, "")

			// Check for temporary ID map environment variable declaration
			// Use the exact syntax pattern to avoid matching the bundled script content
			envVarDeclaration := "GH_AW_TEMPORARY_ID_MAP: ${{ needs.create_issue.outputs.temporary_id_map }}"
			if tt.expectTempIDEnvVar {
				if !strings.Contains(stepsStr, envVarDeclaration) {
					t.Error("Expected GH_AW_TEMPORARY_ID_MAP environment variable declaration not found in job steps")
				}
			} else {
				if strings.Contains(stepsStr, envVarDeclaration) {
					t.Error("Unexpected GH_AW_TEMPORARY_ID_MAP environment variable declaration found in job steps")
				}
			}
		})
	}
}
