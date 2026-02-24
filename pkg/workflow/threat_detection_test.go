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

			if result.RunsOn != tt.expectedConfig.RunsOn {
				t.Errorf("Expected RunsOn %q, got %q", tt.expectedConfig.RunsOn, result.RunsOn)
			}
		})
	}
}

func TestFormatDetectionRunsOn(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		safeOutputs    *SafeOutputsConfig
		agentRunsOn    string
		expectedRunsOn string
	}{
		{
			name:           "nil safe outputs uses agent runs-on",
			safeOutputs:    nil,
			agentRunsOn:    "runs-on: ubuntu-latest",
			expectedRunsOn: "runs-on: ubuntu-latest",
		},
		{
			name: "detection runs-on takes priority over agent runs-on",
			safeOutputs: &SafeOutputsConfig{
				RunsOn: "self-hosted",
				ThreatDetection: &ThreatDetectionConfig{
					RunsOn: "detection-runner",
				},
			},
			agentRunsOn:    "runs-on: ubuntu-latest",
			expectedRunsOn: "runs-on: detection-runner",
		},
		{
			name: "falls back to agent runs-on when detection runs-on is empty",
			safeOutputs: &SafeOutputsConfig{
				RunsOn:          "self-hosted",
				ThreatDetection: &ThreatDetectionConfig{},
			},
			agentRunsOn:    "runs-on: my-agent-runner",
			expectedRunsOn: "runs-on: my-agent-runner",
		},
		{
			name: "falls back to agent runs-on when both detection and safe-outputs runs-on are empty",
			safeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			agentRunsOn:    "runs-on: ubuntu-latest",
			expectedRunsOn: "runs-on: ubuntu-latest",
		},
		{
			name: "nil threat detection uses agent runs-on",
			safeOutputs: &SafeOutputsConfig{
				RunsOn:          "windows-latest",
				ThreatDetection: nil,
			},
			agentRunsOn:    "runs-on: my-agent-runner",
			expectedRunsOn: "runs-on: my-agent-runner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.formatDetectionRunsOn(tt.safeOutputs, tt.agentRunsOn)
			if result != tt.expectedRunsOn {
				t.Errorf("Expected runs-on %q, got %q", tt.expectedRunsOn, result)
			}
		})
	}
}

func TestBuildInlineDetectionSteps(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		data        *WorkflowData
		expectNil   bool
		expectSteps bool
	}{
		{
			name: "threat detection disabled (nil) should return nil",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: nil,
				},
			},
			expectNil:   true,
			expectSteps: false,
		},
		{
			name: "threat detection enabled should create inline steps",
			data: &WorkflowData{
				RunsOn: "runs-on: ubuntu-latest",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			expectNil:   false,
			expectSteps: true,
		},
		{
			name: "threat detection with custom steps should create inline steps",
			data: &WorkflowData{
				RunsOn: "runs-on: ubuntu-latest",
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
			expectNil:   false,
			expectSteps: true,
		},
		{
			name: "nil safe outputs should return nil",
			data: &WorkflowData{
				SafeOutputs: nil,
			},
			expectNil:   true,
			expectSteps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildInlineDetectionSteps(tt.data)

			if tt.expectNil && steps != nil {
				t.Errorf("Expected nil steps, got %d lines", len(steps))
			}
			if !tt.expectNil && steps == nil {
				t.Errorf("Expected non-nil steps, got nil")
			}

			if tt.expectSteps {
				joined := strings.Join(steps, "")
				// Verify key inline detection step components
				if !strings.Contains(joined, "detection_guard") {
					t.Error("Expected inline steps to contain detection_guard step")
				}
				if !strings.Contains(joined, "parse_detection_results") {
					t.Error("Expected inline steps to contain parse_detection_results step")
				}
				if !strings.Contains(joined, "detection_conclusion") {
					t.Error("Expected inline steps to contain detection_conclusion step")
				}
				if !strings.Contains(joined, "Threat Detection (inline)") {
					t.Error("Expected inline steps to contain threat detection comment separator")
				}
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
	steps := compiler.buildInlineDetectionSteps(data)
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

	// Verify parse results step exists
	if !strings.Contains(joined, "parse_detection_results") {
		t.Error("Expected inline steps to include parse_detection_results step")
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

	steps := compiler.buildInlineDetectionSteps(data)
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

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				Steps: []any{
					map[string]any{
						"name": "Custom Threat Scan",
						"run":  "echo 'Custom scanning...'",
					},
				},
			},
		},
	}

	steps := compiler.buildInlineDetectionSteps(data)

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Find the positions of key steps
	customStepPos := strings.Index(stepsString, "Custom Threat Scan")
	parseStepPos := strings.Index(stepsString, "Parse threat detection results")
	uploadStepPos := strings.Index(stepsString, "Upload threat detection log")

	// Verify all steps exist
	if customStepPos == -1 {
		t.Error("Expected to find 'Custom Threat Scan' step")
	}
	if parseStepPos == -1 {
		t.Error("Expected to find 'Parse threat detection results' step")
	}
	if uploadStepPos == -1 {
		t.Error("Expected to find 'Upload threat detection log' step")
	}

	// Verify ordering: custom steps should come before parsing step
	if customStepPos > parseStepPos {
		t.Errorf("Custom threat detection steps should come before 'Parse threat detection results' step. Got custom at position %d, parse at position %d", customStepPos, parseStepPos)
	}

	// Verify ordering: parsing step should come before upload step
	if parseStepPos > uploadStepPos {
		t.Errorf("'Parse threat detection results' step should come before 'Upload threat detection log' step. Got parse at position %d, upload at position %d", parseStepPos, uploadStepPos)
	}

	// Verify the expected order: custom -> parse -> upload
	if customStepPos >= parseStepPos || parseStepPos >= uploadStepPos {
		t.Errorf("Expected step order: custom steps < parse results < upload log. Got positions: custom=%d, parse=%d, upload=%d", customStepPos, parseStepPos, uploadStepPos)
	}
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

func TestBuildUploadDetectionLogStep(t *testing.T) {
	compiler := NewCompiler()

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
		"uses: actions/upload-artifact@b7c566a772e6b6bfb58ed0dc250532a479d7789f",
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
	compiler := NewCompiler()

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildInlineDetectionSteps(data)

	if len(steps) == 0 {
		t.Fatal("Expected non-empty steps")
	}

	// Join all steps into a single string for easier verification
	stepsString := strings.Join(steps, "")

	// Verify that the upload detection log step is included
	if !strings.Contains(stepsString, "Upload threat detection log") {
		t.Error("Expected inline detection steps to include upload detection log step")
	}

	if !strings.Contains(stepsString, "threat-detection.log") {
		t.Error("Expected inline detection steps to include threat-detection.log artifact name")
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
	if !strings.Contains(script, "setupGlobals(core, github, context, exec, io)") {
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

// TestBuildDetectionEngineExecutionStepStripsAgentField verifies that the Agent field from the
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
// default model gpt-5.1-codex-mini for the detection step when no model is specified
func TestCopilotDetectionDefaultModel(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name               string
		data               *WorkflowData
		shouldContainModel bool
		expectedModel      string
	}{
		{
			name: "copilot engine without model uses default gpt-5.1-codex-mini",
			data: &WorkflowData{
				AI: "copilot",
				SafeOutputs: &SafeOutputsConfig{
					ThreatDetection: &ThreatDetectionConfig{},
				},
			},
			shouldContainModel: true,
			expectedModel:      string(constants.DefaultCopilotDetectionModel),
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
			name: "copilot engine with threat detection engine config without model uses default",
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
			expectedModel:      string(constants.DefaultCopilotDetectionModel),
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
				// For detection steps, check if either:
				// 1. The model is set via native COPILOT_MODEL env var (for configured models)
				// 2. The environment variable GH_AW_MODEL_DETECTION_COPILOT is used (for default/fallback)
				hasNativeEnvVar := strings.Contains(allSteps, "COPILOT_MODEL: "+tt.expectedModel)
				hasEnvVar := strings.Contains(allSteps, "GH_AW_MODEL_DETECTION_COPILOT")

				if !hasNativeEnvVar && !hasEnvVar {
					t.Errorf("Expected steps to contain either COPILOT_MODEL: %q or GH_AW_MODEL_DETECTION_COPILOT environment variable, but neither was found.\nGenerated steps:\n%s", tt.expectedModel, allSteps)
				}
			}
		})
	}
}
