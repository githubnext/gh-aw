package workflow

import (
	"strings"
	"testing"
)

func TestAddZizmorIgnoreForWorkflowRun(t *testing.T) {
	c := NewCompiler(false, "", "test")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "workflow_run trigger gets annotation",
			input: `"on":
  workflow_run:
    branches:
    - main`,
			expected: `"on":
  workflow_run:
    # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation
    branches:
    - main`,
		},
		{
			name: "no workflow_run trigger",
			input: `"on":
  push:
    branches:
    - main`,
			expected: `"on":
  push:
    branches:
    - main`,
		},
		{
			name: "workflow_run with different indentation",
			input: `"on":
    workflow_run:
      branches:
      - main`,
			expected: `"on":
    workflow_run:
      # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation
      branches:
      - main`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.addZizmorIgnoreForWorkflowRun(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestJobHasWorkflowRunSafetyChecks(t *testing.T) {
	tests := []struct {
		name        string
		job         *Job
		expectField bool
	}{
		{
			name: "job with workflow_run safety checks",
			job: &Job{
				Name:                       "activation",
				HasWorkflowRunSafetyChecks: true,
				If:                         "github.event.workflow_run.repository.id == github.repository_id",
				RunsOn:                     "runs-on: ubuntu-latest",
			},
			expectField: true,
		},
		{
			name: "job without workflow_run safety checks",
			job: &Job{
				Name:                       "build",
				HasWorkflowRunSafetyChecks: false,
				If:                         "github.event_name == 'push'",
				RunsOn:                     "runs-on: ubuntu-latest",
			},
			expectField: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.job.HasWorkflowRunSafetyChecks != tt.expectField {
				t.Errorf("Expected HasWorkflowRunSafetyChecks=%v, got %v", tt.expectField, tt.job.HasWorkflowRunSafetyChecks)
			}

			// Test that the field is present in rendered YAML when true
			jm := NewJobManager()
			jm.AddJob(tt.job)
			yaml := jm.RenderToYAML()

			if tt.expectField {
				if !strings.Contains(yaml, "# zizmor: ignore[dangerous-triggers]") {
					t.Error("Expected zizmor annotation in rendered YAML")
				}
			}
		})
	}
}
