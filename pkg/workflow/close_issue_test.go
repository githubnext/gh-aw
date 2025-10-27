package workflow

import (
	"testing"
)

func TestParseCloseIssuesConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *CloseIssuesConfig
	}{
		{
			name: "basic configuration",
			input: map[string]any{
				"close-issue": map[string]any{
					"max": 1,
				},
			},
			expected: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
			},
		},
		{
			name: "configuration with required labels",
			input: map[string]any{
				"close-issue": map[string]any{
					"required-labels": []any{"stale", "wontfix"},
					"max":             3,
				},
			},
			expected: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 3,
				},
				RequiredLabels: []string{"stale", "wontfix"},
			},
		},
		{
			name: "configuration with allowed outcomes",
			input: map[string]any{
				"close-issue": map[string]any{
					"outcome": []any{"completed", "not_planned"},
					"max":     2,
				},
			},
			expected: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 2,
				},
				Outcome: []string{"completed", "not_planned"},
			},
		},
		{
			name: "configuration with target",
			input: map[string]any{
				"close-issue": map[string]any{
					"target": "123",
					"max":    1,
				},
			},
			expected: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Target: "123",
			},
		},
		{
			name: "configuration with target-repo",
			input: map[string]any{
				"close-issue": map[string]any{
					"target-repo": "owner/repo",
					"max":         1,
				},
			},
			expected: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				TargetRepoSlug: "owner/repo",
			},
		},
		{
			name: "full configuration",
			input: map[string]any{
				"close-issue": map[string]any{
					"required-labels": []any{"stale"},
					"outcome":         []any{"completed"},
					"target":          "*",
					"target-repo":     "owner/repo",
					"min":             1,
					"max":             5,
					"github-token":    "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Min:         1,
					Max:         5,
					GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
				},
				RequiredLabels: []string{"stale"},
				Outcome:        []string{"completed"},
				Target:         "*",
				TargetRepoSlug: "owner/repo",
			},
		},
		{
			name:     "no close-issue configuration",
			input:    map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			result := c.parseCloseIssuesConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected result, got nil")
			}

			if result.Max != tt.expected.Max {
				t.Errorf("Max: expected %d, got %d", tt.expected.Max, result.Max)
			}

			if result.Min != tt.expected.Min {
				t.Errorf("Min: expected %d, got %d", tt.expected.Min, result.Min)
			}

			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken: expected %q, got %q", tt.expected.GitHubToken, result.GitHubToken)
			}

			if len(result.RequiredLabels) != len(tt.expected.RequiredLabels) {
				t.Errorf("RequiredLabels length: expected %d, got %d", len(tt.expected.RequiredLabels), len(result.RequiredLabels))
			}

			for i, label := range tt.expected.RequiredLabels {
				if i >= len(result.RequiredLabels) || result.RequiredLabels[i] != label {
					t.Errorf("RequiredLabels[%d]: expected %q, got %q", i, label, result.RequiredLabels[i])
				}
			}

			if len(result.Outcome) != len(tt.expected.Outcome) {
				t.Errorf("Outcome length: expected %d, got %d", len(tt.expected.Outcome), len(result.Outcome))
			}

			for i, outcome := range tt.expected.Outcome {
				if i >= len(result.Outcome) || result.Outcome[i] != outcome {
					t.Errorf("Outcome[%d]: expected %q, got %q", i, outcome, result.Outcome[i])
				}
			}

			if result.Target != tt.expected.Target {
				t.Errorf("Target: expected %q, got %q", tt.expected.Target, result.Target)
			}

			if result.TargetRepoSlug != tt.expected.TargetRepoSlug {
				t.Errorf("TargetRepoSlug: expected %q, got %q", tt.expected.TargetRepoSlug, result.TargetRepoSlug)
			}
		})
	}
}

func TestCloseIssueJobGeneration(t *testing.T) {
	c := &Compiler{}
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CloseIssues: &CloseIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				RequiredLabels: []string{"stale"},
				Outcome:        []string{"completed", "not_planned"},
			},
		},
	}

	job, err := c.buildCloseIssueJob(data, "agent")
	if err != nil {
		t.Fatalf("buildCloseIssueJob failed: %v", err)
	}

	if job == nil {
		t.Fatal("Expected job, got nil")
	}

	if job.Name != "close_issue" {
		t.Errorf("Expected job name 'close_issue', got %q", job.Name)
	}

	if job.TimeoutMinutes != 10 {
		t.Errorf("Expected timeout 10, got %d", job.TimeoutMinutes)
	}

	// Check that the job has the necessary outputs
	if job.Outputs == nil {
		t.Error("Expected outputs to be set")
	}

	if _, ok := job.Outputs["issue_number"]; !ok {
		t.Error("Expected issue_number output")
	}

	if _, ok := job.Outputs["issue_url"]; !ok {
		t.Error("Expected issue_url output")
	}

	// Check that the job depends on the main job
	if len(job.Needs) == 0 {
		t.Error("Expected job to have dependencies")
	}

	if job.Needs[0] != "agent" {
		t.Errorf("Expected job to depend on 'agent', got %q", job.Needs[0])
	}
}

func TestHasSafeOutputsEnabledWithCloseIssue(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeOutputsConfig
		expected bool
	}{
		{
			name: "close-issue enabled",
			config: &SafeOutputsConfig{
				CloseIssues: &CloseIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 1,
					},
				},
			},
			expected: true,
		},
		{
			name:     "no outputs enabled",
			config:   &SafeOutputsConfig{},
			expected: false,
		},
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSafeOutputsEnabled(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
