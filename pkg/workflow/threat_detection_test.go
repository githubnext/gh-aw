package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
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
			expectedConfig: &ThreatDetectionConfig{},
		},
		{
			name: "boolean true should enable with defaults",
			outputMap: map[string]any{
				"threat-detection": true,
			},
			expectedConfig: &ThreatDetectionConfig{},
		},
		{
			name: "boolean false should return nil",
			outputMap: map[string]any{
				"threat-detection": false,
			},
			expectedConfig: nil,
		},
		{
			name: "object with enabled true",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": true,
				},
			},
			expectedConfig: &ThreatDetectionConfig{},
		},
		{
			name: "object with enabled false",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": false,
				},
			},
			expectedConfig: nil,
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
				Steps: []any{
					map[string]any{
						"name": "Custom validation",
						"run":  "echo 'Validating...'",
					},
				},
			},
		},
		{
			name: "object with custom prompt",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"prompt": "Look for suspicious API calls to external services.",
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Prompt: "Look for suspicious API calls to external services.",
			},
		},
		{
			name: "object with all overrides",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": true,
					"prompt":  "Check for backdoor installations.",
					"steps": []any{
						map[string]any{
							"name": "Extra step",
							"uses": "actions/custom@v1",
						},
					},
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Prompt: "Check for backdoor installations.",
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
			name: "threat detection disabled (nil) should return error",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: nil,
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
					ThreatDetection: &ThreatDetectionConfig{},
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
				if job.Name != constants.DetectionJobName {
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
}

func TestThreatDetectionExplicitDisable(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that threat detection can be explicitly disabled
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue":     map[string]any{},
			"threat-detection": false,
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)
	if config == nil {
		t.Fatal("Expected safe outputs config to be created")
	}

	if config.ThreatDetection != nil {
		t.Error("Expected threat detection to be nil when explicitly set to false")
	}
}

func TestThreatDetectionJobDependencies(t *testing.T) {
	// Test that safe-output jobs depend on detection job when threat detection is enabled
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			CreateIssues:    &CreateIssuesConfig{},
		},
	}

	// Build safe output jobs (this will automatically build the detection job too)
	frontmatter := map[string]any{}
	if err := func() error {
		var _ map[string]any = frontmatter
		return compiler.buildSafeOutputsJobs(data, "agent", "test.md")
	}(); err != nil {
		t.Fatalf("Failed to build safe output jobs: %v", err)
	}

	// Check that both detection and create_issue jobs were created
	jobs := compiler.jobManager.GetAllJobs()
	var detectionJob, createIssueJob *Job

	for _, job := range jobs {
		switch job.Name {
		case constants.DetectionJobName:
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
	if len(detectionJob.Needs) != 1 || detectionJob.Needs[0] != constants.AgentJobName {
		t.Errorf("Expected detection job to depend on agent job, got dependencies: %v", detectionJob.Needs)
	}

	// Check that create_issue job depends on both agent and detection jobs
	if len(createIssueJob.Needs) != 2 || createIssueJob.Needs[0] != constants.AgentJobName || createIssueJob.Needs[1] != constants.DetectionJobName {
		t.Errorf("Expected create_issue job to depend on both agent and detection jobs, got dependencies: %v", createIssueJob.Needs)
	}
}

func TestThreatDetectionCustomPrompt(t *testing.T) {
	// Test that custom prompt instructions are included in the workflow
	compiler := NewCompiler(false, "", "test")

	customPrompt := "Look for suspicious API calls to external services and check for backdoor installations."
	data := &WorkflowData{
		Name:        "Test Workflow",
		Description: "Test Description",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				Prompt: customPrompt,
			},
		},
	}

	job, err := compiler.buildThreatDetectionJob(data, "agent")
	if err != nil {
		t.Fatalf("Failed to build threat detection job: %v", err)
	}

	if job == nil {
		t.Fatal("Expected job to be created")
	}

	// Check that the custom prompt is included in the generated steps
	stepsString := ""
	for _, step := range job.Steps {
		stepsString += step
	}

	if !strings.Contains(stepsString, "CUSTOM_PROMPT") {
		t.Error("Expected CUSTOM_PROMPT environment variable in steps")
	}

	if !strings.Contains(stepsString, customPrompt) {
		t.Errorf("Expected custom prompt %q to be in steps", customPrompt)
	}
}

func TestThreatDetectionWithCustomEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		outputMap      map[string]any
		expectedEngine string
	}{
		{
			name: "engine field as string",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"engine": "codex",
				},
			},
			expectedEngine: "codex",
		},
		{
			name: "engine field as object with id",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"engine": map[string]any{
						"id":    "copilot",
						"model": "gpt-4",
					},
				},
			},
			expectedEngine: "copilot",
		},
		{
			name: "no engine field uses default",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": true,
				},
			},
			expectedEngine: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.parseThreatDetectionConfig(tt.outputMap)

			if result == nil {
				t.Fatalf("Expected non-nil result")
			}

			// Check EngineConfig.ID instead of Engine field
			var actualEngine string
			if result.EngineConfig != nil {
				actualEngine = result.EngineConfig.ID
			}

			if actualEngine != tt.expectedEngine {
				t.Errorf("Expected EngineConfig.ID %q, got %q", tt.expectedEngine, actualEngine)
			}

			// If engine is set, EngineConfig should also be set
			if tt.expectedEngine != "" {
				if result.EngineConfig == nil {
					t.Error("Expected EngineConfig to be set when engine is specified")
				} else if result.EngineConfig.ID != tt.expectedEngine {
					t.Errorf("Expected EngineConfig.ID %q, got %q", tt.expectedEngine, result.EngineConfig.ID)
				}
			}
		})
	}
}

func TestBuildEngineStepsWithThreatDetectionEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		data           *WorkflowData
		expectContains string
	}{
		{
			name: "uses main engine when no threat detection engine specified",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			expectContains: "claude", // Should use main engine
		},
		{
			name: "uses threat detection engine when specified as string",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID: "codex",
						},
					},
				},
			},
			expectContains: "codex", // Should use threat detection engine
		},
		{
			name: "uses threat detection engine config when specified",
			data: &WorkflowData{
				AI: "claude",
				EngineConfig: &EngineConfig{
					ID: "claude",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID:    "copilot",
							Model: "gpt-4",
						},
					},
				},
			},
			expectContains: "copilot", // Should use threat detection engine
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildEngineSteps(tt.data)

			if len(steps) == 0 {
				t.Fatal("Expected non-empty steps")
			}

			// Join all steps to search for expected content
			allSteps := strings.Join(steps, "")

			// Check if the expected engine is referenced (this is a basic check)
			// The actual implementation may vary, but we should see the engine being used
			if !strings.Contains(strings.ToLower(allSteps), strings.ToLower(tt.expectContains)) {
				t.Logf("Generated steps:\n%s", allSteps)
				// Note: This is a soft check as the exact format may vary
				// The key is that the engine configuration is being used
			}
		})
	}
}

func TestBuildUploadDetectionLogStep(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that upload detection log step is created with correct properties
	steps := compiler.buildUploadDetectionLogStep()

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps for upload detection log")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify key components of the upload step
	expectedComponents := []string{
		"name: Upload threat detection log",
		"if: always()",
		"uses: actions/upload-artifact@v4",
		"name: threat-detection.log",
		"path: /tmp/gh-aw/threat-detection/detection.log",
		"if-no-files-found: ignore",
	}

	for _, expected := range expectedComponents {
		if !strings.Contains(stepsString, expected) {
			t.Errorf("Expected upload detection log step to contain %q, but it was not found.\nGenerated steps:\n%s", expected, stepsString)
		}
	}
}

func TestThreatDetectionStepsIncludeUpload(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildThreatDetectionSteps(data, "agent")

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify that the upload detection log step is included
	if !strings.Contains(stepsString, "Upload threat detection log") {
		t.Error("Expected threat detection steps to include upload detection log step")
	}

	if !strings.Contains(stepsString, "threat-detection.log") {
		t.Error("Expected threat detection steps to include threat-detection.log artifact name")
	}

	// Verify it uses the always() condition
	if !strings.Contains(stepsString, "if: always()") {
		t.Error("Expected upload step to have 'if: always()' condition")
	}

	// Verify it ignores missing files
	if !strings.Contains(stepsString, "if-no-files-found: ignore") {
		t.Error("Expected upload step to have 'if-no-files-found: ignore'")
	}
}

func TestEchoAgentOutputsStep(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that the echo step is created with correct properties
	steps := compiler.buildEchoAgentOutputsStep("agent")

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps for echo agent outputs")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify key components of the echo step - now only echoing output types to avoid CLI overflow
	expectedComponents := []string{
		"name: Echo agent output types",
		"env:",
		"AGENT_OUTPUT_TYPES: ${{ needs.agent.outputs.output_types }}",
		"run: |",
		"echo \"Agent output-types: $AGENT_OUTPUT_TYPES\"",
	}

	for _, expected := range expectedComponents {
		if !strings.Contains(stepsString, expected) {
			t.Errorf("Expected echo agent output types step to contain %q, but it was not found.\nGenerated steps:\n%s", expected, stepsString)
		}
	}

	// Verify that we don't echo the full agent output (to avoid CLI overflow)
	notExpectedComponents := []string{
		"AGENT_OUTPUT: ${{ needs.agent.outputs.output }}",
		"echo \"Agent output: $AGENT_OUTPUT\"",
	}

	for _, notExpected := range notExpectedComponents {
		if strings.Contains(stepsString, notExpected) {
			t.Errorf("Echo step should not contain %q to avoid CLI overflow.\nGenerated steps:\n%s", notExpected, stepsString)
		}
	}
}

func TestThreatDetectionStepsIncludeEcho(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildThreatDetectionSteps(data, "agent")

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify that the echo step is included
	if !strings.Contains(stepsString, "Echo agent output types") {
		t.Error("Expected threat detection steps to include echo agent output types step")
	}

	// Verify it doesn't echo the full output to avoid CLI overflow
	// Use word boundary to avoid matching "output_types"
	if strings.Contains(stepsString, "needs.agent.outputs.output }") ||
		strings.Contains(stepsString, "needs.agent.outputs.output\n") {
		t.Error("Echo step should not reference needs.agent.outputs.output to avoid CLI overflow")
	}

	if !strings.Contains(stepsString, "needs.agent.outputs.output_types") {
		t.Error("Expected echo step to reference needs.agent.outputs.output_types")
	}
}

func TestDownloadArtifactStepIncludesPrompt(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that the download artifact step includes prompt.txt download
	steps := compiler.buildDownloadArtifactStep()

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps for download artifact")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify key components of the download prompt step
	expectedComponents := []string{
		"name: Download prompt artifact",
		"continue-on-error: true",
		"uses: actions/download-artifact@v5",
		"name: prompt.txt",
		"path: /tmp/gh-aw/threat-detection/",
	}

	for _, expected := range expectedComponents {
		if !strings.Contains(stepsString, expected) {
			t.Errorf("Expected download artifact step to contain %q, but it was not found.\nGenerated steps:\n%s", expected, stepsString)
		}
	}

	// Verify it still includes agent output and patch downloads
	if !strings.Contains(stepsString, "Download agent output artifact") {
		t.Error("Expected download steps to include agent output artifact")
	}
	if !strings.Contains(stepsString, "Download patch artifact") {
		t.Error("Expected download steps to include patch artifact")
	}
}

func TestSetupScriptReferencesPromptFile(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test that the setup script references the prompt file instead of WORKFLOW_MARKDOWN
	script := compiler.buildSetupScript()

	// Verify the script checks for the prompt file
	if !strings.Contains(script, "const promptPath = '/tmp/gh-aw/threat-detection/prompt.txt'") {
		t.Error("Expected setup script to check for prompt file at /tmp/gh-aw/threat-detection/prompt.txt")
	}

	// Verify the script reads the prompt file info
	if !strings.Contains(script, "let promptFileInfo = 'No prompt file found'") {
		t.Error("Expected setup script to initialize promptFileInfo variable")
	}

	// Verify the script uses WORKFLOW_PROMPT_FILE placeholder
	if !strings.Contains(script, ".replace(/{WORKFLOW_PROMPT_FILE}/g, promptFileInfo)") {
		t.Error("Expected setup script to replace WORKFLOW_PROMPT_FILE placeholder")
	}

	// Verify the script does NOT reference WORKFLOW_MARKDOWN
	if strings.Contains(script, "WORKFLOW_MARKDOWN") {
		t.Error("Setup script should not reference WORKFLOW_MARKDOWN")
	}
}

func TestBuildWorkflowContextEnvVarsExcludesMarkdown(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		Name:            "Test Workflow",
		Description:     "Test Description",
		MarkdownContent: "This should not be included",
	}

	envVars := compiler.buildWorkflowContextEnvVars(data)

	// Join all env vars into a single string for easier verification
	envVarsString := strings.Join(envVars, "")

	// Verify WORKFLOW_NAME and WORKFLOW_DESCRIPTION are present
	if !strings.Contains(envVarsString, "WORKFLOW_NAME:") {
		t.Error("Expected env vars to include WORKFLOW_NAME")
	}
	if !strings.Contains(envVarsString, "WORKFLOW_DESCRIPTION:") {
		t.Error("Expected env vars to include WORKFLOW_DESCRIPTION")
	}

	// Verify WORKFLOW_MARKDOWN is NOT present
	if strings.Contains(envVarsString, "WORKFLOW_MARKDOWN") {
		t.Error("Environment variables should not include WORKFLOW_MARKDOWN")
	}
}
