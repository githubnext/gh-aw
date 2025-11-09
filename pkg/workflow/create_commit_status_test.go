package workflow

import (
	"testing"
)

func TestParseCommitStatusConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *CreateCommitStatusConfig
	}{
		{
			name:     "nil when not configured",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "empty config with defaults",
			input: map[string]any{
				"create-commit-status": map[string]any{},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1, // Default max is 1
				},
			},
		},
		{
			name: "config with custom context",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"context": "custom-status",
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Context: "custom-status",
			},
		},
		{
			name: "config with custom max",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"max": 3,
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1, // Max is always enforced to 1 for commit status
				},
			},
		},
		{
			name: "config with allowed-domains",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"allowed-domains": []any{"example.com", "*.trusted.org"},
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				AllowedDomains: []string{"example.com", "*.trusted.org"},
			},
		},
		{
			name: "config with github-token",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max:         1,
					GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
		},
		{
			name: "full config",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"context":      "ci/build",
					"max":          2,
					"github-token": "${{ secrets.TOKEN }}",
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max:         1, // Max is always enforced to 1 for commit status
					GitHubToken: "${{ secrets.TOKEN }}",
				},
				Context: "ci/build",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			result := c.parseCommitStatusConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected non-nil result")
			}

			if result.Max != tt.expected.Max {
				t.Errorf("Max: expected %d, got %d", tt.expected.Max, result.Max)
			}

			if result.Context != tt.expected.Context {
				t.Errorf("Context: expected %q, got %q", tt.expected.Context, result.Context)
			}

			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken: expected %q, got %q", tt.expected.GitHubToken, result.GitHubToken)
			}

			// Check AllowedDomains
			if len(result.AllowedDomains) != len(tt.expected.AllowedDomains) {
				t.Errorf("AllowedDomains length: expected %d, got %d", len(tt.expected.AllowedDomains), len(result.AllowedDomains))
			} else {
				for i, domain := range tt.expected.AllowedDomains {
					if result.AllowedDomains[i] != domain {
						t.Errorf("AllowedDomains[%d]: expected %q, got %q", i, domain, result.AllowedDomains[i])
					}
				}
			}
		})
	}
}

func TestBuildCreateCommitStatusJob(t *testing.T) {
	tests := []struct {
		name          string
		workflowData  *WorkflowData
		expectedError bool
	}{
		{
			name: "successful job creation with defaults",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateCommitStatus: &CreateCommitStatusConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 1,
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "successful job creation with custom context",
			workflowData: &WorkflowData{
				Name:            "Test Workflow",
				FrontmatterName: "Custom Name",
				SafeOutputs: &SafeOutputsConfig{
					CreateCommitStatus: &CreateCommitStatusConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 1,
						},
						Context: "custom-context",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "error when safe outputs not configured",
			workflowData: &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: nil,
			},
			expectedError: true,
		},
		{
			name: "error when create-commit-status not configured",
			workflowData: &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			job, err := c.buildCreateCommitStatusJob(tt.workflowData, "main_job")

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if job == nil {
				t.Fatal("Expected non-nil job")
			}

			// Verify job metadata
			if job.Name != "create_commit_status" {
				t.Errorf("Job name: expected 'create_commit_status', got %q", job.Name)
			}

			if job.TimeoutMinutes != 10 {
				t.Errorf("Timeout: expected 10, got %d", job.TimeoutMinutes)
			}

			// Verify outputs
			if job.Outputs == nil {
				t.Fatal("Expected outputs to be set")
			}

			if _, ok := job.Outputs["status_created"]; !ok {
				t.Error("Missing 'status_created' output")
			}

			if _, ok := job.Outputs["status_url"]; !ok {
				t.Error("Missing 'status_url' output")
			}

			// Verify needs
			if len(job.Needs) == 0 {
				t.Error("Expected job to have dependencies")
			}

			found := false
			for _, need := range job.Needs {
				if need == "main_job" {
					found = true
					break
				}
			}
			if !found {
				t.Error("Job should depend on main_job")
			}
		})
	}
}

func TestHasSafeOutputsEnabledWithCommitStatus(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeOutputsConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config",
			config:   &SafeOutputsConfig{},
			expected: false,
		},
		{
			name: "only create-commit-status enabled",
			config: &SafeOutputsConfig{
				CreateCommitStatus: &CreateCommitStatusConfig{},
			},
			expected: true,
		},
		{
			name: "multiple safe outputs enabled",
			config: &SafeOutputsConfig{
				CreateIssues:       &CreateIssuesConfig{},
				CreateCommitStatus: &CreateCommitStatusConfig{},
			},
			expected: true,
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
