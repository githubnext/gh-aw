package workflow

import (
	"testing"
)

func TestParseTriggerWorkflowConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *TriggerWorkflowConfig
	}{
		{
			name: "basic configuration with allowed workflows",
			input: map[string]any{
				"trigger-workflow": map[string]any{
					"allowed": []any{"build.yml", "deploy.yml", "test.yml"},
				},
			},
			expected: &TriggerWorkflowConfig{
				Allowed: []string{"build.yml", "deploy.yml", "test.yml"},
			},
		},
		{
			name: "configuration with max and min",
			input: map[string]any{
				"trigger-workflow": map[string]any{
					"allowed": []any{"build.yml"},
					"max":     5,
					"min":     1,
				},
			},
			expected: &TriggerWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 5,
					Min: 1,
				},
				Allowed: []string{"build.yml"},
			},
		},
		{
			name: "configuration with github-token",
			input: map[string]any{
				"trigger-workflow": map[string]any{
					"allowed":      []any{"build.yml"},
					"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &TriggerWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
				},
				Allowed: []string{"build.yml"},
			},
		},
		{
			name: "null configuration",
			input: map[string]any{
				"trigger-workflow": nil,
			},
			expected: &TriggerWorkflowConfig{
				Allowed: []string{},
			},
		},
		{
			name:     "missing configuration",
			input:    map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			result := c.parseTriggerWorkflowConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %+v, got nil", tt.expected)
				return
			}

			// Compare allowed workflows
			if len(result.Allowed) != len(tt.expected.Allowed) {
				t.Errorf("Expected %d allowed workflows, got %d", len(tt.expected.Allowed), len(result.Allowed))
			} else {
				for i, workflow := range tt.expected.Allowed {
					if result.Allowed[i] != workflow {
						t.Errorf("Expected allowed[%d] = %s, got %s", i, workflow, result.Allowed[i])
					}
				}
			}

			// Compare max
			if result.Max != tt.expected.Max {
				t.Errorf("Expected max = %d, got %d", tt.expected.Max, result.Max)
			}

			// Compare min
			if result.Min != tt.expected.Min {
				t.Errorf("Expected min = %d, got %d", tt.expected.Min, result.Min)
			}

			// Compare github-token
			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("Expected github-token = %s, got %s", tt.expected.GitHubToken, result.GitHubToken)
			}
		})
	}
}

func TestExtractTriggerWorkflowConfig(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expectNil   bool
		expected    *TriggerWorkflowConfig
	}{
		{
			name: "trigger-workflow in safe-outputs",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"trigger-workflow": map[string]any{
						"allowed": []any{"build.yml", "deploy.yml"},
						"max":     10,
					},
				},
			},
			expectNil: false,
			expected: &TriggerWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
				Allowed: []string{"build.yml", "deploy.yml"},
			},
		},
		{
			name: "no safe-outputs configuration",
			frontmatter: map[string]any{
				"on": "push",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			config := c.extractSafeOutputsConfig(tt.frontmatter)

			if tt.expectNil {
				if config != nil && config.TriggerWorkflow != nil {
					t.Errorf("Expected nil TriggerWorkflow config, got %+v", config.TriggerWorkflow)
				}
				return
			}

			if config == nil || config.TriggerWorkflow == nil {
				t.Errorf("Expected TriggerWorkflow config, got nil")
				return
			}

			result := config.TriggerWorkflow

			// Compare allowed workflows
			if len(result.Allowed) != len(tt.expected.Allowed) {
				t.Errorf("Expected %d allowed workflows, got %d", len(tt.expected.Allowed), len(result.Allowed))
			} else {
				for i, workflow := range tt.expected.Allowed {
					if result.Allowed[i] != workflow {
						t.Errorf("Expected allowed[%d] = %s, got %s", i, workflow, result.Allowed[i])
					}
				}
			}

			// Compare max
			if result.Max != tt.expected.Max {
				t.Errorf("Expected max = %d, got %d", tt.expected.Max, result.Max)
			}
		})
	}
}

func TestGenerateSafeOutputsConfigWithTriggerWorkflow(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		contains []string
	}{
		{
			name: "trigger-workflow configuration",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					TriggerWorkflow: &TriggerWorkflowConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 5,
							Min: 1,
						},
						Allowed: []string{"build.yml", "deploy.yml"},
					},
				},
			},
			contains: []string{
				`"trigger_workflow"`,
				`"allowed"`,
				`"build.yml"`,
				`"deploy.yml"`,
				`"max":5`,
				`"min":1`,
			},
		},
		{
			name: "trigger-workflow with empty allowed list",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					TriggerWorkflow: &TriggerWorkflowConfig{
						Allowed: []string{},
					},
				},
			},
			contains: []string{
				`"trigger_workflow"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSafeOutputsConfig(tt.data)

			if result == "" {
				t.Error("Expected non-empty config JSON, got empty string")
				return
			}

			for _, expected := range tt.contains {
				if !containsSubstring(result, expected) {
					t.Errorf("Expected config to contain %q, but it didn't. Config: %s", expected, result)
				}
			}
		})
	}
}

func TestHasSafeOutputsEnabledWithTriggerWorkflow(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeOutputsConfig
		expected bool
	}{
		{
			name: "trigger-workflow enabled",
			config: &SafeOutputsConfig{
				TriggerWorkflow: &TriggerWorkflowConfig{
					Allowed: []string{"build.yml"},
				},
			},
			expected: true,
		},
		{
			name: "trigger-workflow with other outputs",
			config: &SafeOutputsConfig{
				TriggerWorkflow: &TriggerWorkflowConfig{
					Allowed: []string{"build.yml"},
				},
				CreateIssues: &CreateIssuesConfig{},
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
