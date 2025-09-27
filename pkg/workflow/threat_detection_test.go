package workflow

import (
	"testing"
)

func TestParseThreatDetectionConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		outputMap      map[string]any
		expectedConfig *ThreatDetectionConfig
	}{
		{
			name:           "missing threat-detection should return default enabled",
			outputMap:      map[string]any{},
			expectedConfig: &ThreatDetectionConfig{Enabled: true},
		},
		{
			name: "boolean true should enable with defaults",
			outputMap: map[string]any{
				"threat-detection": true,
			},
			expectedConfig: &ThreatDetectionConfig{Enabled: true},
		},
		{
			name: "boolean false should disable",
			outputMap: map[string]any{
				"threat-detection": false,
			},
			expectedConfig: &ThreatDetectionConfig{Enabled: false},
		},
		{
			name: "object with enabled true",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": true,
				},
			},
			expectedConfig: &ThreatDetectionConfig{Enabled: true},
		},
		{
			name: "object with enabled false",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": false,
				},
			},
			expectedConfig: &ThreatDetectionConfig{Enabled: false},
		},
		{
			name: "object with custom engine",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"engine": "claude-3.5",
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Enabled: true,
				Engine:  "claude-3.5",
			},
		},
		{
			name: "object with custom prompt",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"prompt": "/path/to/custom-prompt.md",
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Enabled: true,
				Prompt:  "/path/to/custom-prompt.md",
			},
		},
		{
			name: "object with custom steps",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"steps": []any{
						map[string]any{
							"name": "Custom validation",
							"run":  "echo 'Validating...'",
						},
					},
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Enabled: true,
				Steps: []any{
					map[string]any{
						"name": "Custom validation",
						"run":  "echo 'Validating...'",
					},
				},
			},
		},
		{
			name: "object with all overrides",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": true,
					"engine":  "custom-engine",
					"prompt":  "https://example.com/prompt.md",
					"steps": []any{
						map[string]any{
							"name": "Extra step",
							"uses": "actions/custom@v1",
						},
					},
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Enabled: true,
				Engine:  "custom-engine",
				Prompt:  "https://example.com/prompt.md",
				Steps: []any{
					map[string]any{
						"name": "Extra step",
						"uses": "actions/custom@v1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.parseThreatDetectionConfig(tt.outputMap)

			if result == nil && tt.expectedConfig != nil {
				t.Fatalf("Expected non-nil result, got nil")
			}
			if result != nil && tt.expectedConfig == nil {
				t.Fatalf("Expected nil result, got %+v", result)
			}
			if result == nil && tt.expectedConfig == nil {
				return
			}

			if result.Enabled != tt.expectedConfig.Enabled {
				t.Errorf("Expected Enabled %v, got %v", tt.expectedConfig.Enabled, result.Enabled)
			}
			if result.Engine != tt.expectedConfig.Engine {
				t.Errorf("Expected Engine %q, got %q", tt.expectedConfig.Engine, result.Engine)
			}
			if result.Prompt != tt.expectedConfig.Prompt {
				t.Errorf("Expected Prompt %q, got %q", tt.expectedConfig.Prompt, result.Prompt)
			}
			if len(result.Steps) != len(tt.expectedConfig.Steps) {
				t.Errorf("Expected %d steps, got %d", len(tt.expectedConfig.Steps), len(result.Steps))
			}
		})
	}
}

func TestBuildThreatDetectionJob(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		data        *WorkflowData
		mainJobName string
		expectError bool
		expectJob   bool
	}{
		{
			name: "threat detection disabled should return error",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						Enabled: false,
					},
				},
			},
			mainJobName: "agent",
			expectError: true,
			expectJob:   false,
		},
		{
			name: "threat detection enabled should create job",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						Enabled: true,
					},
				},
			},
			mainJobName: "agent",
			expectError: false,
			expectJob:   true,
		},
		{
			name: "threat detection with custom steps should create job",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						Enabled: true,
						Steps: []any{
							map[string]any{
								"name": "Custom step",
								"run":  "echo 'custom validation'",
							},
						},
					},
				},
			},
			mainJobName: "agent",
			expectError: false,
			expectJob:   true,
		},
		{
			name: "nil safe outputs should return error",
			data: &WorkflowData{
				SafeOutputs: nil,
			},
			mainJobName: "agent",
			expectError: true,
			expectJob:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := compiler.buildThreatDetectionJob(tt.data, tt.mainJobName)

			if tt.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.expectJob && job == nil {
				t.Errorf("Expected job to be created, got nil")
			}
			if !tt.expectJob && job != nil {
				t.Errorf("Expected no job, got %+v", job)
			}

			if job != nil {
				// Verify job properties
				if job.Name != "detection" {
					t.Errorf("Expected job name 'detection', got %q", job.Name)
				}
				if job.RunsOn != "runs-on: ubuntu-latest" {
					t.Errorf("Expected ubuntu-latest runner, got %q", job.RunsOn)
				}
				if job.Permissions != "permissions: read-all" {
					t.Errorf("Expected read-all permissions, got %q", job.Permissions)
				}
				if len(job.Needs) != 1 || job.Needs[0] != tt.mainJobName {
					t.Errorf("Expected job to depend on %q, got %v", tt.mainJobName, job.Needs)
				}
				if job.TimeoutMinutes != 10 {
					t.Errorf("Expected 10 minute timeout, got %d", job.TimeoutMinutes)
				}
			}
		})
	}
}

func TestThreatDetectionDefaultBehavior(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that threat detection is enabled by default when safe-outputs exist
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{},
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)
	if config == nil {
		t.Fatal("Expected safe outputs config to be created")
	}

	if config.ThreatDetection == nil {
		t.Fatal("Expected threat detection to be automatically enabled")
	}

	if !config.ThreatDetection.Enabled {
		t.Error("Expected threat detection to be enabled by default")
	}
}

func TestThreatDetectionExplicitDisable(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that threat detection can be explicitly disabled
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue":       map[string]any{},
			"threat-detection": false,
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)
	if config == nil {
		t.Fatal("Expected safe outputs config to be created")
	}

	if config.ThreatDetection == nil {
		t.Fatal("Expected threat detection config to exist")
	}

	if config.ThreatDetection.Enabled {
		t.Error("Expected threat detection to be disabled when explicitly set to false")
	}
}

func TestThreatDetectionJobDependencies(t *testing.T) {
	// Test that safe-output jobs depend on detection job when threat detection is enabled
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				Enabled: true,
			},
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	// Build safe output jobs (this will automatically build the detection job too)
	frontmatter := map[string]any{}
	if err := compiler.buildSafeOutputsJobs(data, "agent", false, frontmatter, "test.md"); err != nil {
		t.Fatalf("Failed to build safe output jobs: %v", err)
	}

	// Check that both detection and create_issue jobs were created
	jobs := compiler.jobManager.GetAllJobs()
	var detectionJob, createIssueJob *Job
	
	for _, job := range jobs {
		switch job.Name {
		case "detection":
			detectionJob = job
		case "create_issue":
			createIssueJob = job
		}
	}

	if detectionJob == nil {
		t.Fatal("Expected detection job to be created")
	}

	if createIssueJob == nil {
		t.Fatal("Expected create_issue job to be created")
	}

	// Check that detection job depends on agent job
	if len(detectionJob.Needs) != 1 || detectionJob.Needs[0] != "agent" {
		t.Errorf("Expected detection job to depend on agent job, got dependencies: %v", detectionJob.Needs)
	}

	// Check that create_issue job depends on detection job
	if len(createIssueJob.Needs) != 1 || createIssueJob.Needs[0] != "detection" {
		t.Errorf("Expected create_issue job to depend on detection job, got dependencies: %v", createIssueJob.Needs)
	}
}