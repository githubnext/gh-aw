package workflow

import (
	"testing"
)

func TestParseDispatchWorkflowConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *DispatchWorkflowConfig
		wantNil  bool
	}{
		{
			name: "valid configuration with allowed workflows",
			input: map[string]any{
				"dispatch-workflow": map[string]any{
					"allowed-workflows": []any{"ci.yml", "build.yml"},
					"max":               3,
				},
			},
			expected: &DispatchWorkflowConfig{
				AllowedWorkflows: []string{"ci.yml", "build.yml"},
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 3,
				},
			},
			wantNil: false,
		},
		{
			name: "valid configuration with single workflow",
			input: map[string]any{
				"dispatch-workflow": map[string]any{
					"allowed-workflows": []any{"deploy.yml"},
				},
			},
			expected: &DispatchWorkflowConfig{
				AllowedWorkflows: []string{"deploy.yml"},
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1, // Default
				},
			},
			wantNil: false,
		},
		{
			name: "invalid - empty allowed workflows",
			input: map[string]any{
				"dispatch-workflow": map[string]any{
					"allowed-workflows": []any{},
				},
			},
			wantNil: true,
		},
		{
			name: "invalid - missing allowed workflows",
			input: map[string]any{
				"dispatch-workflow": map[string]any{
					"max": 5,
				},
			},
			wantNil: true,
		},
		{
			name: "no dispatch-workflow config",
			input: map[string]any{
				"create-issue": map[string]any{},
			},
			wantNil: true,
		},
		{
			name: "configuration with github-token",
			input: map[string]any{
				"dispatch-workflow": map[string]any{
					"allowed-workflows": []any{"test.yml"},
					"github-token":      "${{ secrets.PAT }}",
				},
			},
			expected: &DispatchWorkflowConfig{
				AllowedWorkflows: []string{"test.yml"},
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max:         1,
					GitHubToken: "${{ secrets.PAT }}",
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			result := compiler.parseDispatchWorkflowConfig(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected non-nil result, got nil")
			}

			if len(result.AllowedWorkflows) != len(tt.expected.AllowedWorkflows) {
				t.Errorf("AllowedWorkflows length mismatch: got %d, want %d",
					len(result.AllowedWorkflows), len(tt.expected.AllowedWorkflows))
			}

			for i, workflow := range result.AllowedWorkflows {
				if workflow != tt.expected.AllowedWorkflows[i] {
					t.Errorf("AllowedWorkflows[%d] = %s, want %s",
						i, workflow, tt.expected.AllowedWorkflows[i])
				}
			}

			if result.Max != tt.expected.Max {
				t.Errorf("Max = %d, want %d", result.Max, tt.expected.Max)
			}

			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken = %s, want %s", result.GitHubToken, tt.expected.GitHubToken)
			}
		})
	}
}

func TestBuildDispatchWorkflowJob(t *testing.T) {
	tests := []struct {
		name        string
		data        *WorkflowData
		mainJobName string
		wantErr     bool
		checkJob    func(*testing.T, *Job)
	}{
		{
			name: "valid dispatch workflow job",
			data: &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					DispatchWorkflow: &DispatchWorkflowConfig{
						AllowedWorkflows: []string{"ci.yml", "build.yml"},
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 1,
						},
					},
				},
			},
			mainJobName: "agent",
			wantErr:     false,
			checkJob: func(t *testing.T, job *Job) {
				if job.Name != "dispatch_workflow" {
					t.Errorf("job name = %s, want dispatch_workflow", job.Name)
				}
				if job.TimeoutMinutes != 10 {
					t.Errorf("timeout = %d, want 10", job.TimeoutMinutes)
				}
				// Check that agent is in needs
				found := false
				for _, need := range job.Needs {
					if need == "agent" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("job needs should include 'agent', got %v", job.Needs)
				}
				if job.Permissions == "" {
					t.Error("permissions should not be empty")
				}
				if len(job.Outputs) != 2 {
					t.Errorf("outputs count = %d, want 2", len(job.Outputs))
				}
			},
		},
		{
			name: "missing safe outputs config",
			data: &WorkflowData{
				Name: "Test Workflow",
			},
			mainJobName: "agent",
			wantErr:     true,
		},
		{
			name: "missing dispatch workflow config",
			data: &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{},
			},
			mainJobName: "agent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			job, err := compiler.buildDispatchWorkflowJob(tt.data, tt.mainJobName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkJob != nil {
				tt.checkJob(t, job)
			}
		})
	}
}

func TestDispatchWorkflowSafeOutputsEnabled(t *testing.T) {
	config := &SafeOutputsConfig{
		DispatchWorkflow: &DispatchWorkflowConfig{
			AllowedWorkflows: []string{"test.yml"},
		},
	}

	if !HasSafeOutputsEnabled(config) {
		t.Error("HasSafeOutputsEnabled should return true when dispatch-workflow is configured")
	}
}
