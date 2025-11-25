package workflow

import (
	"strings"
	"testing"
)

// TestBuildListSafeOutputJob verifies the shared list-based safe-output job builder
func TestBuildListSafeOutputJob(t *testing.T) {
	tests := []struct {
		name               string
		params             ListSafeOutputJobParams
		staged             bool
		targetRepoSlug     string
		expectedEnvVars    []string
		expectedJobName    string
		expectedConditions []string
	}{
		{
			name: "add_labels with allowed list and max",
			params: ListSafeOutputJobParams{
				JobName:        "add_labels",
				StepName:       "Add Labels",
				StepID:         "add_labels",
				MainJobName:    "agent",
				EnvPrefix:      "LABELS",
				AllowedItems:   []string{"bug", "enhancement"},
				MaxCount:       5,
				Target:         "",
				TargetRepoSlug: "",
				Script:         "const test = 1;",
				Permissions:    NewPermissionsContentsReadIssuesWritePRWrite(),
				Token:          "",
				OutputKey:      "labels_added",
				TriggeringContextConditions: []ConditionNode{
					BuildPropertyAccess("github.event.issue.number"),
					BuildPropertyAccess("github.event.pull_request.number"),
				},
			},
			expectedEnvVars: []string{
				"GH_AW_LABELS_ALLOWED: \"bug,enhancement\"",
				"GH_AW_LABELS_MAX_COUNT: 5",
			},
			expectedJobName: "add_labels",
			expectedConditions: []string{
				"github.event.issue.number",
				"github.event.pull_request.number",
			},
		},
		{
			name: "add_reviewer with reviewers list",
			params: ListSafeOutputJobParams{
				JobName:        "add_reviewer",
				StepName:       "Add Reviewers",
				StepID:         "add_reviewer",
				MainJobName:    "agent",
				EnvPrefix:      "REVIEWERS",
				AllowedItems:   []string{"octocat", "someone"},
				MaxCount:       2,
				Target:         "",
				TargetRepoSlug: "",
				Script:         "const test = 1;",
				Permissions:    NewPermissionsContentsReadPRWrite(),
				Token:          "",
				OutputKey:      "reviewers_added",
				TriggeringContextConditions: []ConditionNode{
					BuildPropertyAccess("github.event.pull_request.number"),
				},
			},
			expectedEnvVars: []string{
				"GH_AW_REVIEWERS_ALLOWED: \"octocat,someone\"",
				"GH_AW_REVIEWERS_MAX_COUNT: 2",
			},
			expectedJobName: "add_reviewer",
			expectedConditions: []string{
				"github.event.pull_request.number",
			},
		},
		{
			name: "add_labels with explicit target",
			params: ListSafeOutputJobParams{
				JobName:      "add_labels",
				StepName:     "Add Labels",
				StepID:       "add_labels",
				MainJobName:  "agent",
				EnvPrefix:    "LABELS",
				AllowedItems: []string{},
				MaxCount:     0, // Should use default of 3
				Target:       "*",
				Script:       "const test = 1;",
				Permissions:  NewPermissionsContentsReadIssuesWritePRWrite(),
				OutputKey:    "labels_added",
				TriggeringContextConditions: []ConditionNode{
					BuildPropertyAccess("github.event.issue.number"),
					BuildPropertyAccess("github.event.pull_request.number"),
				},
			},
			expectedEnvVars: []string{
				"GH_AW_LABELS_ALLOWED: \"\"",
				"GH_AW_LABELS_MAX_COUNT: 3",
				"GH_AW_LABELS_TARGET: \"*\"",
			},
			expectedJobName: "add_labels",
			// Should NOT include context conditions when target is explicit
			expectedConditions: []string{},
		},
		{
			name: "add_reviewer with target repo",
			params: ListSafeOutputJobParams{
				JobName:        "add_reviewer",
				StepName:       "Add Reviewers",
				StepID:         "add_reviewer",
				MainJobName:    "agent",
				EnvPrefix:      "REVIEWERS",
				AllowedItems:   []string{"reviewer1"},
				MaxCount:       1,
				Target:         "",
				TargetRepoSlug: "owner/other-repo",
				Script:         "const test = 1;",
				Permissions:    NewPermissionsContentsReadPRWrite(),
				Token:          "${{ secrets.CUSTOM_TOKEN }}",
				OutputKey:      "reviewers_added",
				TriggeringContextConditions: []ConditionNode{
					BuildPropertyAccess("github.event.pull_request.number"),
				},
			},
			staged:         true,
			targetRepoSlug: "owner/other-repo",
			expectedEnvVars: []string{
				"GH_AW_REVIEWERS_ALLOWED: \"reviewer1\"",
				"GH_AW_REVIEWERS_MAX_COUNT: 1",
				"GH_AW_TARGET_REPO_SLUG: \"owner/other-repo\"",
			},
			expectedJobName:    "add_reviewer",
			expectedConditions: []string{"github.event.pull_request.number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler(false, "", "test")

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					Staged: tt.staged,
				},
			}

			job, err := c.buildListSafeOutputJob(workflowData, tt.params)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify job name
			if job.Name != tt.expectedJobName {
				t.Errorf("Expected job name %q, got %q", tt.expectedJobName, job.Name)
			}

			// Convert steps to string for testing
			stepsContent := strings.Join(job.Steps, "")

			// Verify expected environment variables are present
			for _, envVar := range tt.expectedEnvVars {
				if !strings.Contains(stepsContent, envVar) {
					t.Errorf("Expected env var %q not found in job steps", envVar)
				}
			}

			// Verify condition includes expected context checks
			for _, condition := range tt.expectedConditions {
				if !strings.Contains(job.If, condition) {
					t.Errorf("Expected condition to include %q, got: %s", condition, job.If)
				}
			}

			// Verify condition does NOT include context checks when target is explicit
			if tt.params.Target != "" {
				for _, condition := range tt.params.TriggeringContextConditions {
					rendered := condition.Render()
					if strings.Contains(job.If, rendered) {
						t.Errorf("Expected condition to NOT include %q when target is explicit, got: %s", rendered, job.If)
					}
				}
			}

			// Verify outputs
			outputKey := tt.params.OutputKey
			if _, exists := job.Outputs[outputKey]; !exists {
				t.Errorf("Expected output key %q not found", outputKey)
			}
		})
	}
}

// TestBuildListSafeOutputJobDefaultMaxCount verifies that the default max count is used when not specified
func TestBuildListSafeOutputJobDefaultMaxCount(t *testing.T) {
	c := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	params := ListSafeOutputJobParams{
		JobName:     "add_labels",
		StepName:    "Add Labels",
		StepID:      "add_labels",
		MainJobName: "agent",
		EnvPrefix:   "LABELS",
		MaxCount:    0, // Should use default of 3
		Script:      "const test = 1;",
		Permissions: NewPermissionsContentsReadIssuesWritePRWrite(),
		OutputKey:   "labels_added",
	}

	job, err := c.buildListSafeOutputJob(workflowData, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "GH_AW_LABELS_MAX_COUNT: 3") {
		t.Error("Expected default max count of 3 when MaxCount is 0")
	}
}

// TestBuildListSafeOutputJobEmptyAllowedItems verifies that empty allowed items results in empty string
func TestBuildListSafeOutputJobEmptyAllowedItems(t *testing.T) {
	c := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	params := ListSafeOutputJobParams{
		JobName:      "add_labels",
		StepName:     "Add Labels",
		StepID:       "add_labels",
		MainJobName:  "agent",
		EnvPrefix:    "LABELS",
		AllowedItems: []string{}, // Empty list
		MaxCount:     3,
		Script:       "const test = 1;",
		Permissions:  NewPermissionsContentsReadIssuesWritePRWrite(),
		OutputKey:    "labels_added",
	}

	job, err := c.buildListSafeOutputJob(workflowData, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "GH_AW_LABELS_ALLOWED: \"\"") {
		t.Error("Expected empty string for allowed items when list is empty")
	}
}
