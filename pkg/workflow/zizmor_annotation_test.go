//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestAddZizmorIgnoreForWorkflowRun(t *testing.T) {
	c := NewCompiler()

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
		{
			name: "workflow_run in comment should not get annotation",
			input: `"on":
  push:
    # This is not a workflow_run: trigger
    branches:
    - main`,
			expected: `"on":
  push:
    # This is not a workflow_run: trigger
    branches:
    - main`,
		},
		{
			name: "workflow_run with inline comment gets annotation",
			input: `"on":
  workflow_run: # This is a workflow_run trigger
    branches:
    - main`,
			expected: `"on":
  workflow_run: # This is a workflow_run trigger
    # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation
    branches:
    - main`,
		},
		{
			name: "multiple workflow_run keys only annotates first",
			input: `"on":
  workflow_run:
    branches:
    - main
  workflow_run:
    branches:
    - develop`,
			expected: `"on":
  workflow_run:
    # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation
    branches:
    - main
  workflow_run:
    branches:
    - develop`,
		},
		{
			name: "workflow_run with value should not get annotation",
			input: `"on":
  push:
    branches:
    - workflow_run: something`,
			expected: `"on":
  push:
    branches:
    - workflow_run: something`,
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

func TestAddZizmorIgnoreForSecretsOutsideEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "env var with secret gets annotation",
			input: `    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`,
			expected: `    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
		},
		{
			name: "multiple secrets on same line",
			input: `    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`,
			expected: `    env:
      GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
		},
		{
			name: "no secrets present",
			input: `    env:
      NODE_ENV: production`,
			expected: `    env:
      NODE_ENV: production`,
		},
		{
			name: "comment lines are not modified",
			input: `# secrets.GITHUB_TOKEN is used here
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`,
			expected: `# secrets.GITHUB_TOKEN is used here
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
		},
		{
			name: "already annotated lines are not modified",
			input: `    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
			expected: `    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
		},
		{
			name: "github-token input gets annotation",
			input: `      - uses: actions/checkout@v4
        with:
          github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`,
			expected: `      - uses: actions/checkout@v4
        with:
          github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
		},
		{
			name:     "empty string returns unchanged",
			input:    "",
			expected: "",
		},
		{
			name: "secrets without expression syntax not modified",
			input: `    # This mentions secrets.GITHUB_TOKEN but not in an expression
    env:
      FOO: bar`,
			expected: `    # This mentions secrets.GITHUB_TOKEN but not in an expression
    env:
      FOO: bar`,
		},
		{
			name: "multiple lines with secrets all get annotations",
			input: `    env:
      COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`,
			expected: `    env:
      COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]
      GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }} # zizmor: ignore[secrets-outside-env]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addZizmorIgnoreForSecretsOutsideEnv(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}
