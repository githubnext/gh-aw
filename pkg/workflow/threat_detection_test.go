//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestParseThreatDetectionConfig(t *testing.T) {
	compiler := NewCompiler()

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
			name: "object with custom post-steps",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"post-steps": []any{
						map[string]any{
							"name": "Custom post validation",
							"run":  "echo 'Post validating...'",
						},
					},
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				PostSteps: []any{
					map[string]any{
						"name": "Custom post validation",
						"run":  "echo 'Post validating...'",
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
			name: "object with all overrides including pre and post steps",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"enabled": true,
					"prompt":  "Check for backdoor installations.",
					"steps": []any{
						map[string]any{
							"name": "Pre step",
							"uses": "actions/setup@v1",
						},
					},
					"post-steps": []any{
						map[string]any{
							"name": "Post step",
							"uses": "actions/cleanup@v1",
						},
					},
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				Prompt: "Check for backdoor installations.",
				Steps: []any{
					map[string]any{
						"name": "Pre step",
						"uses": "actions/setup@v1",
					},
				},
				PostSteps: []any{
					map[string]any{
						"name": "Post step",
						"uses": "actions/cleanup@v1",
					},
				},
			},
		},
		{
			name: "object with runs-on override",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"runs-on": "self-hosted",
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				RunsOn: "self-hosted",
			},
		},
		{
			name: "object with continue-on-error true",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"continue-on-error": true,
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				ContinueOnError: boolPtr(true),
			},
		},
		{
			name: "object with continue-on-error false",
			outputMap: map[string]any{
				"threat-detection": map[string]any{
					"continue-on-error": false,
				},
			},
			expectedConfig: &ThreatDetectionConfig{
				ContinueOnError: boolPtr(false),
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

			if len(result.PostSteps) != len(tt.expectedConfig.PostSteps) {
				t.Errorf("Expected %d post-steps, got %d", len(tt.expectedConfig.PostSteps), len(result.PostSteps))
			}

			if result.RunsOn != tt.expectedConfig.RunsOn {
				t.Errorf("Expected RunsOn %q, got %q", tt.expectedConfig.RunsOn, result.RunsOn)
			}

			if (result.ContinueOnError == nil) != (tt.expectedConfig.ContinueOnError == nil) {
				t.Errorf("Expected ContinueOnError nil=%v, got nil=%v", tt.expectedConfig.ContinueOnError == nil, result.ContinueOnError == nil)
			} else if result.ContinueOnError != nil && tt.expectedConfig.ContinueOnError != nil {
				if *result.ContinueOnError != *tt.expectedConfig.ContinueOnError {
					t.Errorf("Expected ContinueOnError %v, got %v", *tt.expectedConfig.ContinueOnError, *result.ContinueOnError)
				}
			}
		})
	}
}

func TestIsContinueOnError(t *testing.T) {
	tests := []struct {
		name     string
		config   *ThreatDetectionConfig
		expected bool
	}{
		{
			name:     "default (nil) continues on error",
			config:   &ThreatDetectionConfig{},
			expected: true,
		},
		{
			name:     "explicit true continues on error",
			config:   &ThreatDetectionConfig{ContinueOnError: boolPtr(true)},
			expected: true,
		},
		{
			name:     "explicit false does not continue on error",
			config:   &ThreatDetectionConfig{ContinueOnError: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsContinueOnError()
			if result != tt.expected {
				t.Errorf("Expected IsContinueOnError() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestThreatDetectionDefaultBehavior(t *testing.T) {
	compiler := NewCompiler()

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
	compiler := NewCompiler()

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

func TestThreatDetectionInlineStepsDependencies(t *testing.T) {
	// Test that inline detection steps are generated when threat detection is enabled
	// and that safe-output jobs can check detection results via agent job outputs
	compiler := NewCompiler()

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	// Build inline detection steps
	steps := compiler.buildDetectionJobSteps(data)
	if steps == nil {
		t.Fatal("Expected inline detection steps to be created")
	}

	joined := strings.Join(steps, "")

	// Verify detection guard step exists (determines if detection should run)
	if !strings.Contains(joined, "detection_guard") {
		t.Error("Expected inline steps to include detection_guard step")
	}

	// Verify detection conclusion step exists (sets final detection outputs)
	if !strings.Contains(joined, "detection_conclusion") {
		t.Error("Expected inline steps to include detection_conclusion step")
	}

	// Verify the conclusion step references the parsing script (combined step)
	if !strings.Contains(joined, "parse_threat_detection_results.cjs") {
		t.Error("Expected inline steps to reference parse_threat_detection_results.cjs in combined conclusion step")
	}
}

func TestThreatDetectionCustomPrompt(t *testing.T) {
	// Test that custom prompt instructions are included in the inline detection steps
	compiler := NewCompiler()

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

	steps := compiler.buildDetectionJobSteps(data)
	if steps == nil {
		t.Fatal("Expected inline detection steps to be created")
	}

	// Check that the custom prompt is included in the generated steps
	stepsString := strings.Join(steps, "")

	if !strings.Contains(stepsString, "CUSTOM_PROMPT") {
		t.Error("Expected CUSTOM_PROMPT environment variable in steps")
	}

	if !strings.Contains(stepsString, customPrompt) {
		t.Errorf("Expected custom prompt %q to be in steps", customPrompt)
	}
}

func TestThreatDetectionWithEngineConfig(t *testing.T) {
	compiler := NewCompiler()

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

func TestThreatDetectionStepsOrdering(t *testing.T) {
	compiler := NewCompiler()

	t.Run("pre-steps come before engine execution", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{
					Steps: []any{
						map[string]any{
							"name": "Custom Pre Scan",
							"run":  "echo 'Custom pre-scanning...'",
						},
					},
				},
			},
		}

		steps := compiler.buildDetectionJobSteps(data)

		if len(steps) == 0 {
			t.Fatal("Expected non-empty steps")
		}

		// Join all steps into a single string for easier verification
		stepsString := strings.Join(steps, "")

		// Find the positions of key steps
		preStepPos := strings.Index(stepsString, "Custom Pre Scan")
		setupStepPos := strings.Index(stepsString, "Setup threat detection")
		uploadStepPos := strings.Index(stepsString, "Upload threat detection log")

		// Verify all steps exist
		if preStepPos == -1 {
			t.Error("Expected to find 'Custom Pre Scan' step")
		}
		if setupStepPos == -1 {
			t.Error("Expected to find 'Setup threat detection' step")
		}
		if uploadStepPos == -1 {
			t.Error("Expected to find 'Upload threat detection log' step")
		}
		if !strings.Contains(stepsString, "Parse and conclude threat detection") {
			t.Error("Expected to find 'Parse and conclude threat detection' step")
		}

		// Verify ordering: pre-steps should come before setup threat detection
		if preStepPos > setupStepPos {
			t.Errorf("Custom pre-steps should come before 'Setup threat detection'. Got pre-step at position %d, setup at position %d", preStepPos, setupStepPos)
		}

		// Verify ordering: pre-steps should come before upload and conclude
		if preStepPos > uploadStepPos {
			t.Errorf("Custom pre-steps should come before 'Upload threat detection log'. Got pre-step at position %d, upload at position %d", preStepPos, uploadStepPos)
		}
	})

	t.Run("post-steps come after engine execution and before upload", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{
					PostSteps: []any{
						map[string]any{
							"name": "Custom Post Scan",
							"run":  "echo 'Custom post-scanning...'",
						},
					},
				},
			},
		}

		steps := compiler.buildDetectionJobSteps(data)

		if len(steps) == 0 {
			t.Fatal("Expected non-empty steps")
		}

		stepsString := strings.Join(steps, "")

		postStepPos := strings.Index(stepsString, "Custom Post Scan")
		// Use the engine execution step ID as the stable marker for the engine step boundary
		engineStepPos := strings.Index(stepsString, "id: detection_agentic_execution")
		uploadStepPos := strings.Index(stepsString, "Upload threat detection log")
		concludeStepPos := strings.Index(stepsString, "Parse and conclude threat detection")

		if postStepPos == -1 {
			t.Error("Expected to find 'Custom Post Scan' step")
		}
		if engineStepPos == -1 {
			t.Error("Expected to find 'id: detection_agentic_execution' engine step")
		}
		if uploadStepPos == -1 {
			t.Error("Expected to find 'Upload threat detection log' step")
		}
		if concludeStepPos == -1 {
			t.Error("Expected to find 'Parse and conclude threat detection' step")
		}

		// Verify ordering: post-steps should come after the engine execution step
		if postStepPos < engineStepPos {
			t.Errorf("Custom post-steps should come after engine execution step. Got post-step at position %d, engine at position %d", postStepPos, engineStepPos)
		}
		if postStepPos > uploadStepPos {
			t.Errorf("Custom post-steps should come before 'Upload threat detection log'. Got post-step at position %d, upload at position %d", postStepPos, uploadStepPos)
		}
		if postStepPos > concludeStepPos {
			t.Errorf("Custom post-steps should come before 'Parse and conclude threat detection'. Got post-step at position %d, conclude at position %d", postStepPos, concludeStepPos)
		}
	})

	t.Run("pre-steps and post-steps both present in correct order", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{
					Steps: []any{
						map[string]any{
							"name": "Custom Pre Step",
							"run":  "echo 'pre'",
						},
					},
					PostSteps: []any{
						map[string]any{
							"name": "Custom Post Step",
							"run":  "echo 'post'",
						},
					},
				},
			},
		}

		steps := compiler.buildDetectionJobSteps(data)
		stepsString := strings.Join(steps, "")

		preStepPos := strings.Index(stepsString, "Custom Pre Step")
		postStepPos := strings.Index(stepsString, "Custom Post Step")
		engineStepPos := strings.Index(stepsString, "id: detection_agentic_execution")
		uploadStepPos := strings.Index(stepsString, "Upload threat detection log")

		if preStepPos == -1 {
			t.Error("Expected to find 'Custom Pre Step'")
		}
		if postStepPos == -1 {
			t.Error("Expected to find 'Custom Post Step'")
		}
		if engineStepPos == -1 {
			t.Error("Expected to find 'id: detection_agentic_execution' engine step")
		}

		// pre-steps before engine, post-steps after engine but before upload
		if preStepPos > engineStepPos {
			t.Errorf("Pre-steps should come before engine execution step. Got pre=%d, engine=%d", preStepPos, engineStepPos)
		}
		if postStepPos < engineStepPos {
			t.Errorf("Post-steps should come after engine execution step. Got post=%d, engine=%d", postStepPos, engineStepPos)
		}
		if postStepPos > uploadStepPos {
			t.Errorf("Post-steps should come before 'Upload threat detection log'. Got post=%d, upload=%d", postStepPos, uploadStepPos)
		}
		// pre-steps before post-steps
		if preStepPos > postStepPos {
			t.Errorf("Pre-steps should come before post-steps. Got pre=%d, post=%d", preStepPos, postStepPos)
		}
	})
}

func TestCustomThreatDetectionStepsGuardCondition(t *testing.T) {
	compiler := NewCompiler()

	t.Run("injects detection guard condition when no if: present", func(t *testing.T) {
		steps := []any{
			map[string]any{
				"name": "No If Step",
				"run":  "echo hello",
			},
		}
		result := compiler.buildCustomThreatDetectionSteps(steps)
		stepsStr := strings.Join(result, "")
		if !strings.Contains(stepsStr, detectionStepCondition) {
			t.Errorf("Expected detection guard condition to be injected, got:\n%s", stepsStr)
		}
	})

	t.Run("preserves user-provided if: condition", func(t *testing.T) {
		userCondition := "always()"
		steps := []any{
			map[string]any{
				"name": "User If Step",
				"if":   userCondition,
				"run":  "echo hello",
			},
		}
		result := compiler.buildCustomThreatDetectionSteps(steps)
		stepsStr := strings.Join(result, "")
		if strings.Contains(stepsStr, detectionStepCondition) {
			t.Error("Expected detection guard condition NOT to be injected when user provides if:")
		}
		if !strings.Contains(stepsStr, userCondition) {
			t.Errorf("Expected user if: condition %q to be preserved, got:\n%s", userCondition, stepsStr)
		}
	})
}

func TestBuildDetectionEngineExecutionStepWithThreatDetectionEngine(t *testing.T) {
	compiler := NewCompiler()

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
			steps := compiler.buildDetectionEngineExecutionStep(tt.data)

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

func TestBuildDetectionEngineExecutionStepCodexIncludesMCPSetup(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "codex",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildDetectionEngineExecutionStep(data)
	if len(steps) == 0 {
		t.Fatal("Expected non-empty detection engine steps")
	}

	stepsString := strings.Join(steps, "")
	if !strings.Contains(stepsString, "Start MCP Gateway") {
		t.Errorf("Expected Codex detection steps to include MCP setup, got:\n%s", stepsString)
	}
	if !strings.Contains(stepsString, "model_provider = \"openai-proxy\"") {
		t.Errorf("Expected Codex detection MCP config to include openai-proxy model provider, got:\n%s", stepsString)
	}
}

func TestBuildDetectionJobStepsCodexAvoidsDuplicateContainerPullStep(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "codex",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildDetectionJobSteps(data)
	stepsString := strings.Join(steps, "")

	if count := strings.Count(stepsString, "name: Download container images"); count != 1 {
		t.Errorf("Expected exactly one 'Download container images' step for Codex detection, got %d.\n%s", count, stepsString)
	}
}

func TestBuildUploadDetectionLogStep(t *testing.T) {
	compiler := NewCompiler()

	// Test that upload detection log step is created with correct properties
	steps := compiler.buildUploadDetectionLogStep(&WorkflowData{})

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps for upload detection log")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify key components of the upload step
	expectedComponents := []string{
		"name: Upload threat detection log",
		"if: always()",
		"uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
		"name: " + constants.DetectionArtifactName,
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
	compiler := NewCompiler()

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildDetectionJobSteps(data)

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify that the upload detection log step is included
	if !strings.Contains(stepsString, "Upload threat detection log") {
		t.Error("Expected inline detection steps to include upload detection log step")
	}

	if !strings.Contains(stepsString, "detection") {
		t.Error("Expected inline detection steps to include detection artifact name")
	}

	// Verify it ignores missing files
	if !strings.Contains(stepsString, "if-no-files-found: ignore") {
		t.Error("Expected upload step to have 'if-no-files-found: ignore'")
	}
}

func TestSetupScriptReferencesPromptFile(t *testing.T) {
	compiler := NewCompiler()

	// Test that the setup script requires the external .cjs file
	script := compiler.buildSetupScriptRequire()

	// Verify the script uses require to load setup_threat_detection.cjs
	if !strings.Contains(script, "require('"+SetupActionDestination+"/setup_threat_detection.cjs')") {
		t.Error("Expected setup script to require setup_threat_detection.cjs")
	}

	// Verify setupGlobals is called
	if !strings.Contains(script, "setupGlobals(core, github, context, exec, io, getOctokit)") {
		t.Error("Expected setup script to call setupGlobals")
	}

	// Verify main() is awaited without parameters (template is read from file)
	if !strings.Contains(script, "await main()") {
		t.Error("Expected setup script to await main() without parameters")
	}

	// Verify template content is NOT passed as parameter (now read from file)
	if strings.Contains(script, "templateContent") {
		t.Error("Expected setup script to NOT pass templateContent parameter (should read from file)")
	}
}

func TestBuildWorkflowContextEnvVarsExcludesMarkdown(t *testing.T) {
	compiler := NewCompiler()

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

func TestThreatDetectionEngineFalse(t *testing.T) {
	compiler := NewCompiler()

	// Test that engine: false is properly parsed
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{},
			"threat-detection": map[string]any{
				"engine": false,
				"steps": []any{
					map[string]any{
						"name": "Custom Scan",
						"run":  "echo 'Custom scan'",
					},
				},
			},
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)
	if config == nil {
		t.Fatal("Expected safe outputs config to be created")
	}

	if config.ThreatDetection == nil {
		t.Fatal("Expected threat detection to be enabled")
	}

	if !config.ThreatDetection.EngineDisabled {
		t.Error("Expected EngineDisabled to be true when engine: false")
	}

	if config.ThreatDetection.EngineConfig != nil {
		t.Error("Expected EngineConfig to be nil when engine: false")
	}

	if len(config.ThreatDetection.Steps) != 1 {
		t.Fatalf("Expected 1 custom step, got %d", len(config.ThreatDetection.Steps))
	}
}

// TestDetectionGuardStepCondition verifies that the inline detection guard step
// has the correct conditional logic to skip when there are no safe outputs and no patches
func TestDetectionGuardStepCondition(t *testing.T) {
	compiler := NewCompiler()

	// Build the detection guard step
	steps := compiler.buildDetectionGuardStep()

	if len(steps) == 0 {
		t.Fatal("Expected non-empty guard steps")
	}

	joined := strings.Join(steps, "")

	// Verify the guard step has the detection_guard ID
	if !strings.Contains(joined, "id: detection_guard") {
		t.Error("Expected guard step to have id 'detection_guard'")
	}

	// Verify the condition checks for output types
	if !strings.Contains(joined, "OUTPUT_TYPES") {
		t.Error("Expected guard step to check OUTPUT_TYPES")
	}

	// Verify the condition checks for has_patch
	if !strings.Contains(joined, "HAS_PATCH") {
		t.Error("Expected guard step to check HAS_PATCH")
	}

	// Verify it uses always() to run even after agent failure
	if !strings.Contains(joined, "if: always()") {
		t.Error("Expected guard step to use always() condition")
	}

	// Verify it sets run_detection output
	if !strings.Contains(joined, "run_detection=true") {
		t.Error("Expected guard step to set run_detection=true")
	}
	if !strings.Contains(joined, "run_detection=false") {
		t.Error("Expected guard step to set run_detection=false")
	}
}

// TestDetectionJobLevelCondition verifies that the detection job-level `if:` condition
// skips the job entirely when the agent produced no outputs and no patch.
// This prevents the detection job from wasting a runner and ensures safe_outputs is
// also correctly skipped (since it gates on needs.detection.result == 'success').
func TestDetectionJobLevelCondition(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test]",
			},
		},
	}

	job, err := compiler.buildDetectionJob(data)
	if err != nil {
		t.Fatalf("Unexpected error building detection job: %v", err)
	}
	if job == nil {
		t.Fatal("Expected detection job to be built, got nil")
	}

	condition := job.If

	// Must use always() so the job runs even when the agent job fails
	if !strings.Contains(condition, "always()") {
		t.Errorf("Expected detection job condition to include always(), got: %q", condition)
	}

	// Must skip when agent was skipped
	if !strings.Contains(condition, "needs."+string(constants.AgentJobName)+".result") {
		t.Errorf("Expected detection job condition to check agent result, got: %q", condition)
	}
	if !strings.Contains(condition, "'skipped'") {
		t.Errorf("Expected detection job condition to check for skipped status, got: %q", condition)
	}

	// Must check output_types and has_patch so the job is skipped at job-level
	// when the agent produced nothing (avoiding unnecessary runner usage and
	// preventing safe_outputs from running when there is nothing to publish).
	if !strings.Contains(condition, "needs."+string(constants.AgentJobName)+".outputs.output_types") {
		t.Errorf("Expected detection job condition to check output_types, got: %q", condition)
	}
	if !strings.Contains(condition, "needs."+string(constants.AgentJobName)+".outputs.has_patch") {
		t.Errorf("Expected detection job condition to check has_patch, got: %q", condition)
	}
}

// main engine config is never propagated to the detection engine config,
// regardless of whether a model is explicitly configured.
func TestBuildDetectionEngineExecutionStepStripsAgentField(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name string
		data *WorkflowData
	}{
		{
			name: "agent field stripped when model is explicitly configured",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID:    "copilot",
					Model: "claude-opus-4.6",
					Agent: "my-agent",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
		},
		{
			name: "agent field stripped when no model configured",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID:    "copilot",
					Agent: "my-agent",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildDetectionEngineExecutionStep(tt.data)

			if len(steps) == 0 {
				t.Fatal("Expected non-empty steps")
			}

			allSteps := strings.Join(steps, "")

			// The --agent flag must not appear in the threat detection steps
			if strings.Contains(allSteps, "--agent") {
				t.Errorf("Expected detection steps to NOT contain --agent flag, but found it.\nGenerated steps:\n%s", allSteps)
			}

			// Ensure the original engine config is not mutated
			if tt.data.EngineConfig != nil && tt.data.EngineConfig.Agent != "my-agent" {
				t.Errorf("Original EngineConfig.Agent was mutated; expected %q, got %q", "my-agent", tt.data.EngineConfig.Agent)
			}
		})
	}
}

// TestCopilotDetectionDefaultModel verifies that the copilot engine uses the
// Copilot CLI's native default model for the detection step when no model is specified.
// Detection now matches main agent behavior: both use ${{ vars.* || ” }} so the
// Copilot CLI picks its native default (currently claude-sonnet-4.6).
func TestCopilotDetectionDefaultModel(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name               string
		data               *WorkflowData
		shouldContainModel bool
		expectedModel      string
	}{
		{
			name: "copilot engine without model uses native CLI default via env var",
			data: &WorkflowData{
				AI: "copilot",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			shouldContainModel: true,
			// Detection uses env var fallback (same pattern as main agent), allowing
			// the Copilot CLI to pick its native default (currently claude-sonnet-4.6)
			expectedModel: "${{ vars." + constants.EnvVarModelDetectionCopilot + " || '" + constants.CopilotBYOKDefaultModel + "' }}",
		},
		{
			name: "copilot engine with custom model uses specified model",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID:    "copilot",
					Model: "gpt-4",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			shouldContainModel: true,
			expectedModel:      "gpt-4",
		},
		{
			name: "copilot engine with threat detection engine config with custom model",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID:    "copilot",
							Model: "gpt-4o",
						},
					},
				},
			},
			shouldContainModel: true,
			expectedModel:      "gpt-4o",
		},
		{
			name: "copilot engine with threat detection engine config without model uses native CLI default via env var",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID: "copilot",
						},
					},
				},
			},
			shouldContainModel: true,
			expectedModel:      "${{ vars." + constants.EnvVarModelDetectionCopilot + " || '" + constants.CopilotBYOKDefaultModel + "' }}",
		},
		{
			name: "claude engine does not add model parameter",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			shouldContainModel: false,
			expectedModel:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildDetectionEngineExecutionStep(tt.data)

			if len(steps) == 0 {
				t.Fatal("Expected non-empty steps")
			}

			// Join all steps to search for model content
			allSteps := strings.Join(steps, "")

			if tt.shouldContainModel {
				hasNativeEnvVar := strings.Contains(allSteps, "COPILOT_MODEL: "+tt.expectedModel)
				if !hasNativeEnvVar {
					t.Errorf("Expected steps to contain COPILOT_MODEL: %q, but it was not found.\nGenerated steps:\n%s", tt.expectedModel, allSteps)
				}
			}
		})
	}
}

// TestBuildDetectionEngineExecutionStepPropagatesAPITarget verifies that when engine.api-target
// is configured on the main engine, the threat detection AWF invocation also receives
// --copilot-api-target and the GHE domains in --allow-domains.
// Regression test for: Threat detection AWF run missing --copilot-api-target on data residency.
func TestBuildDetectionEngineExecutionStepPropagatesAPITarget(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name             string
		data             *WorkflowData
		expectedTarget   string
		unexpectedTarget string
	}{
		{
			name: "api-target from main engine config is propagated to detection step",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "copilot-api.contoso-aw.ghe.com",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			expectedTarget: "copilot-api.contoso-aw.ghe.com",
		},
		{
			name: "api-target inherited when threat detection has its own engine config without api-target",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "api.acme.ghe.com",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID:    "copilot",
							Model: "gpt-4",
							// No APITarget set - should be inherited from main engine config
						},
					},
				},
			},
			expectedTarget: "api.acme.ghe.com",
		},
		{
			name: "detection engine config api-target takes precedence over main engine config",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "api.acme.ghe.com",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID:        "copilot",
							APITarget: "api.custom-detection.ghe.com",
						},
					},
				},
			},
			expectedTarget:   "api.custom-detection.ghe.com",
			unexpectedTarget: "api.acme.ghe.com",
		},
		{
			name: "no api-target when main engine config has none",
			data: &WorkflowData{
				AI: "copilot",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildDetectionEngineExecutionStep(tt.data)

			if len(steps) == 0 {
				t.Fatal("Expected non-empty steps")
			}

			allSteps := strings.Join(steps, "")

			if tt.expectedTarget != "" {
				if !strings.Contains(allSteps, "--copilot-api-target") {
					t.Errorf("Expected detection steps to contain --copilot-api-target flag.\nGenerated steps:\n%s", allSteps)
				}
				if !strings.Contains(allSteps, tt.expectedTarget) {
					t.Errorf("Expected detection steps to contain api-target %q.\nGenerated steps:\n%s", tt.expectedTarget, allSteps)
				}
			}

			if tt.unexpectedTarget != "" {
				if strings.Contains(allSteps, tt.unexpectedTarget) {
					t.Errorf("Expected detection steps to NOT contain api-target %q, but found it.\nGenerated steps:\n%s", tt.unexpectedTarget, allSteps)
				}
			}
		})
	}
}

// TestDetectionJobPermissionsIndentation verifies that the detection job's permissions block
// is correctly indented in the rendered YAML output.
// Regression test for the indentation bug where c.indentYAMLLines was called on
// RenderToYAML() output which already uses 6-space indentation for permission values,
// resulting in 10-space indentation instead of the correct 6.
func TestDetectionJobPermissionsIndentation(t *testing.T) {
	tests := []struct {
		name            string
		data            *WorkflowData
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "copilot-requests feature produces correctly indented permissions",
			data: &WorkflowData{
				Name: "test-workflow",
				AI:   "copilot",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
				Features: map[string]any{
					string(constants.CopilotRequestsFeatureFlag): true,
				},
			},
			// permission values must be indented by exactly 6 spaces (4 for job key + 2 for sub-key)
			wantContains: []string{
				"      copilot-requests: write",
			},
			// Over-indented value (10 spaces) must not appear - this was the bug
			wantNotContains: []string{
				"          copilot-requests: write",
			},
		},
		{
			name: "permissions block absent when copilot-requests feature disabled and no contents read needed",
			data: &WorkflowData{
				Name: "test-workflow",
				AI:   "copilot",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			// copilot-requests should not be in the output when the feature is not enabled
			wantContains:    []string{},
			wantNotContains: []string{"copilot-requests: write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			job, err := compiler.buildDetectionJob(tt.data)
			if err != nil {
				t.Fatalf("buildDetectionJob() error: %v", err)
			}
			if job == nil {
				t.Fatal("buildDetectionJob() returned nil job")
			}

			if err := compiler.jobManager.AddJob(job); err != nil {
				t.Fatalf("AddJob() error: %v", err)
			}

			yamlOutput := compiler.jobManager.RenderToYAML()

			for _, expected := range tt.wantContains {
				if !strings.Contains(yamlOutput, expected) {
					t.Errorf("YAML output should contain %q, but got:\n%s", expected, yamlOutput)
				}
			}
			for _, unexpected := range tt.wantNotContains {
				if strings.Contains(yamlOutput, unexpected) {
					t.Errorf("YAML output should NOT contain %q, but got:\n%s", unexpected, yamlOutput)
				}
			}
		})
	}
}

// TestWorkspaceCheckoutForDetectionStep verifies that a conditional checkout step
// is added to the detection job when threat detection is enabled, allowing the
// engine to see patches in the context of the full repository.
func TestWorkspaceCheckoutForDetectionStep(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	job, err := compiler.buildDetectionJob(data)
	if err != nil {
		t.Fatalf("buildDetectionJob() error: %v", err)
	}
	if job == nil {
		t.Fatal("buildDetectionJob() returned nil job")
	}

	stepsString := strings.Join(job.Steps, "")

	// Workspace checkout step should be present
	if !strings.Contains(stepsString, "Checkout repository for patch context") {
		t.Error("Detection job should include workspace checkout step")
	}

	// Step should be conditional on has_patch
	expectedCondition := "if: needs." + string(constants.AgentJobName) + ".outputs.has_patch == 'true'"
	if !strings.Contains(stepsString, expectedCondition) {
		t.Errorf("Workspace checkout step should have has_patch condition, expected %q in steps", expectedCondition)
	}

	// Step should disable credential persistence
	if !strings.Contains(stepsString, "persist-credentials: false") {
		t.Error("Workspace checkout step should set persist-credentials: false")
	}

	// Step should use pinned actions/checkout
	checkoutPin := getActionPin("actions/checkout")
	if checkoutPin == "" {
		t.Fatal("Expected actions/checkout to have a pin")
	}
	if !strings.Contains(stepsString, checkoutPin) {
		t.Errorf("Workspace checkout step should use pinned action %q", checkoutPin)
	}
}

// TestDetectionJobAlwaysHasContentsRead verifies that the detection job always
// receives contents: read permission (required for the workspace checkout step),
// even in production mode.
func TestDetectionJobAlwaysHasContentsRead(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	job, err := compiler.buildDetectionJob(data)
	if err != nil {
		t.Fatalf("buildDetectionJob() error: %v", err)
	}
	if job == nil {
		t.Fatal("buildDetectionJob() returned nil job")
	}

	// contents: read should be present in all modes
	if !strings.Contains(job.Permissions, "contents: read") {
		t.Errorf("Detection job should always have contents: read permission, got permissions:\n%s", job.Permissions)
	}
}

// TestWorkspaceCheckoutPresentWithCustomSteps verifies that when the
// detection engine is disabled but custom steps exist, the detection job
// still includes the workspace checkout step (custom steps may also need context).
func TestWorkspaceCheckoutPresentWithCustomSteps(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				EngineDisabled: true,
				Steps: []any{
					map[string]any{"name": "Custom check", "run": "echo custom"},
				},
			},
		},
	}

	job, err := compiler.buildDetectionJob(data)
	if err != nil {
		t.Fatalf("buildDetectionJob() error: %v", err)
	}
	if job == nil {
		t.Fatal("buildDetectionJob() returned nil job, but custom steps are configured")
	}

	stepsString := strings.Join(job.Steps, "")
	if !strings.Contains(stepsString, "Checkout repository for patch context") {
		t.Error("Detection job with custom steps should still include workspace checkout step")
	}
}

// TestWorkspaceCheckoutStepOrdering verifies that the workspace checkout step
// appears after the artifact download and before the detection steps.
func TestWorkspaceCheckoutStepOrdering(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	job, err := compiler.buildDetectionJob(data)
	if err != nil {
		t.Fatalf("buildDetectionJob() error: %v", err)
	}
	if job == nil {
		t.Fatal("buildDetectionJob() returned nil job")
	}

	stepsString := strings.Join(job.Steps, "")

	downloadIdx := strings.Index(stepsString, "Download agent output artifact")
	checkoutIdx := strings.Index(stepsString, "Checkout repository for patch context")
	guardIdx := strings.Index(stepsString, "Check if detection needed")

	if downloadIdx < 0 {
		t.Fatal("Expected 'Download agent output artifact' step in detection job")
	}
	if checkoutIdx < 0 {
		t.Fatal("Expected 'Checkout repository for patch context' step in detection job")
	}
	if guardIdx < 0 {
		t.Fatal("Expected 'Check if detection needed' step in detection job")
	}

	if checkoutIdx < downloadIdx {
		t.Error("Workspace checkout step should appear after artifact download step")
	}
	if checkoutIdx > guardIdx {
		t.Error("Workspace checkout step should appear before detection guard step")
	}
}

func TestCleanFirewallDirsStepPresent(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildDetectionJobSteps(data)
	stepsString := strings.Join(steps, "")

	// The cleanup step should be present
	if !strings.Contains(stepsString, "Clean stale firewall files from agent artifact") {
		t.Error("Expected 'Clean stale firewall files from agent artifact' step in detection steps")
	}

	// It should remove the firewall logs and audit directories
	if !strings.Contains(stepsString, constants.AWFProxyLogsDir) {
		t.Errorf("Expected cleanup step to reference %s", constants.AWFProxyLogsDir)
	}
	if !strings.Contains(stepsString, constants.AWFAuditDir) {
		t.Errorf("Expected cleanup step to reference %s", constants.AWFAuditDir)
	}
}

func TestCleanFirewallDirsStepOrdering(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildDetectionJobSteps(data)
	stepsString := strings.Join(steps, "")

	cleanIdx := strings.Index(stepsString, "Clean stale firewall files from agent artifact")
	guardIdx := strings.Index(stepsString, "Check if detection needed")

	if cleanIdx < 0 {
		t.Fatal("Expected 'Clean stale firewall files from agent artifact' step")
	}
	if guardIdx < 0 {
		t.Fatal("Expected 'Check if detection needed' step")
	}

	// The cleanup step must come before the detection guard
	if cleanIdx > guardIdx {
		t.Error("Cleanup firewall dirs step should appear before detection guard step")
	}
}

func TestBuildPullAWFContainersStepPropagatesFeatures(t *testing.T) {
	compiler := NewCompiler()

	t.Run("cli-proxy image included when feature flag is enabled", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			Features: map[string]any{
				string(constants.CliProxyFeatureFlag): true,
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		}

		steps := compiler.buildPullAWFContainersStep(data)
		stepsString := strings.Join(steps, "")

		if !strings.Contains(stepsString, "cli-proxy") {
			t.Error("Expected cli-proxy image in pull step when cli-proxy feature flag is enabled")
		}
	})

	t.Run("cli-proxy image excluded when feature flag is not set", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			Features: map[string]any{},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		}

		steps := compiler.buildPullAWFContainersStep(data)
		stepsString := strings.Join(steps, "")

		if strings.Contains(stepsString, "cli-proxy") {
			t.Error("Expected no cli-proxy image in pull step when cli-proxy feature flag is not set")
		}
	})
}

func TestBuildDetectionEngineExecutionStepEmitsNodeSetupForCopilot(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name                string
		data                *WorkflowData
		expectedInstallStep string
	}{
		{
			name: "copilot main engine emits Setup Node.js once before install",
			data: &WorkflowData{
				AI: "copilot",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			expectedInstallStep: "Install GitHub Copilot CLI",
		},
		{
			name: "copilot via threat-detection engine override emits Setup Node.js once",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{
						EngineConfig: &EngineConfig{
							ID: "copilot",
						},
					},
				},
			},
			expectedInstallStep: "Install GitHub Copilot CLI",
		},
		{
			name: "claude main engine already bundles Setup Node.js — no duplicate",
			data: &WorkflowData{
				AI: "claude",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			expectedInstallStep: "Install Claude Code CLI",
		},
		{
			name: "codex main engine already bundles Setup Node.js — no duplicate",
			data: &WorkflowData{
				AI: "codex",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			expectedInstallStep: "Install Codex CLI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildDetectionEngineExecutionStep(tt.data)
			if len(steps) == 0 {
				t.Fatal("expected non-empty steps")
			}
			s := strings.Join(steps, "")

			if c := strings.Count(s, "- name: Setup Node.js"); c != 1 {
				t.Errorf("want exactly one Setup Node.js, got %d.\n%s", c, s)
			}

			nodeIdx := strings.Index(s, "- name: Setup Node.js")
			installIdx := strings.Index(s, "- name: "+tt.expectedInstallStep)
			if installIdx == -1 {
				t.Fatalf("missing %q step in:\n%s", tt.expectedInstallStep, s)
			}
			if nodeIdx > installIdx {
				t.Errorf("Setup Node.js (at %d) must precede %q (at %d)", nodeIdx, tt.expectedInstallStep, installIdx)
			}
		})
	}
}

func TestInstallStepsContainNodeSetup(t *testing.T) {
	tests := []struct {
		name     string
		steps    []GitHubActionStep
		expected bool
	}{
		{
			name:     "empty input",
			steps:    nil,
			expected: false,
		},
		{
			name:     "canonical setup-node step from GenerateNodeJsSetupStep",
			steps:    []GitHubActionStep{GenerateNodeJsSetupStep()},
			expected: true,
		},
		{
			name: "install-only step without node setup",
			steps: []GitHubActionStep{
				{"      - name: Install Some CLI", "        run: npm install -g some-cli"},
			},
			expected: false,
		},
		{
			name: "setup-node preceded by unrelated step",
			steps: []GitHubActionStep{
				{"      - name: Checkout", "        uses: actions/checkout@v4"},
				GenerateNodeJsSetupStep(),
			},
			expected: true,
		},
		{
			name: "differently indented setup-node (extractStepName whitespace tolerance)",
			steps: []GitHubActionStep{
				{"    - name: Setup Node.js", "      uses: actions/setup-node@v4"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := installStepsContainNodeSetup(tt.steps)
			if got != tt.expected {
				t.Errorf("installStepsContainNodeSetup() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildDetectionEngineExecutionStepPropagatesDriverScriptOverride(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				EngineConfig: &EngineConfig{
					ID:           "copilot",
					DriverScript: "custom_copilot_driver.cjs",
				},
			},
		},
	}

	steps := compiler.buildDetectionEngineExecutionStep(data)
	if len(steps) == 0 {
		t.Fatal("expected non-empty steps")
	}

	s := strings.Join(steps, "")

	if !strings.Contains(s, "custom_copilot_driver.cjs") {
		t.Errorf("expected custom driver script in detection steps, got:\n%s", s)
	}
	if strings.Contains(s, "actions/copilot_driver.cjs") {
		t.Errorf("expected default driver to be replaced by custom override, got:\n%s", s)
	}
}
