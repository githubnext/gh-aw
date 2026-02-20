//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestModelEnvVarInjectionForAgentJob tests that agent jobs get the correct model environment variable
func TestModelEnvVarInjectionForAgentJob(t *testing.T) {
	tests := []struct {
		name            string
		engine          string
		expectedEnvVar  string
		expectedCommand string
	}{
		{
			name:            "Copilot coding agent uses GH_AW_MODEL_AGENT_COPILOT",
			engine:          "copilot",
			expectedEnvVar:  constants.EnvVarModelAgentCopilot,
			expectedCommand: "${" + constants.EnvVarModelAgentCopilot + ":+ --model",
		},
		{
			name:            "Claude agent uses GH_AW_MODEL_AGENT_CLAUDE",
			engine:          "claude",
			expectedEnvVar:  constants.EnvVarModelAgentClaude,
			expectedCommand: "${" + constants.EnvVarModelAgentClaude + ":+ --model",
		},
		{
			name:            "Codex agent uses GH_AW_MODEL_AGENT_CODEX",
			engine:          "codex",
			expectedEnvVar:  constants.EnvVarModelAgentCodex,
			expectedCommand: "${" + constants.EnvVarModelAgentCodex + ":+-c model=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple workflow with the specified engine
			// Add SafeOutputs to distinguish from detection jobs
			workflowData := &WorkflowData{
				Name: "test-workflow",
				AI:   tt.engine,
				Tools: map[string]any{
					"bash": []any{"echo"},
				},
				SafeOutputs: &SafeOutputsConfig{
					// Just enough to make it an agent job
				},
			}

			// Get the engine
			engine, err := GetGlobalEngineRegistry().GetEngine(tt.engine)
			if err != nil {
				t.Fatalf("Failed to get engine: %v", err)
			}

			// Get execution steps
			steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

			// Convert steps to string for analysis
			var stepsStr strings.Builder
			for _, step := range steps {
				for _, line := range step {
					stepsStr.WriteString(line)
					stepsStr.WriteString("\n")
				}
			}
			stepsContent := stepsStr.String()

			// Check that the environment variable is present
			if !strings.Contains(stepsContent, tt.expectedEnvVar+":") {
				t.Errorf("Expected environment variable %s not found in steps:\n%s", tt.expectedEnvVar, stepsContent)
			}

			// Check that the command uses the env var conditionally
			if !strings.Contains(stepsContent, tt.expectedCommand) {
				t.Errorf("Expected command pattern '%s' not found in steps:\n%s", tt.expectedCommand, stepsContent)
			}

			// Verify env var has fallback to empty string for agent jobs
			expectedEnvLine := tt.expectedEnvVar + ": ${{ vars." + tt.expectedEnvVar + " || '' }}"
			if !strings.Contains(stepsContent, expectedEnvLine) {
				t.Errorf("Expected env var line '%s' not found in steps:\n%s", expectedEnvLine, stepsContent)
			}
		})
	}
}

// TestModelEnvVarInjectionForDetectionJob tests that detection jobs get the correct model environment variable
func TestModelEnvVarInjectionForDetectionJob(t *testing.T) {
	tests := []struct {
		name            string
		engine          string
		expectedEnvVar  string
		expectedDefault string
	}{
		{
			name:            "Copilot detection uses GH_AW_MODEL_DETECTION_COPILOT",
			engine:          "copilot",
			expectedEnvVar:  constants.EnvVarModelDetectionCopilot,
			expectedDefault: "", // No builtin default, CLI will use its own
		},
		{
			name:            "Claude detection uses GH_AW_MODEL_DETECTION_CLAUDE",
			engine:          "claude",
			expectedEnvVar:  constants.EnvVarModelDetectionClaude,
			expectedDefault: "", // Claude has no default detection model
		},
		{
			name:            "Codex detection uses GH_AW_MODEL_DETECTION_CODEX",
			engine:          "codex",
			expectedEnvVar:  constants.EnvVarModelDetectionCodex,
			expectedDefault: "", // Codex has no default detection model
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal detection workflow (no SafeOutputs)
			workflowData := &WorkflowData{
				Name:        "test-detection",
				AI:          tt.engine,
				SafeOutputs: nil, // This makes it a detection job
				Tools: map[string]any{
					"bash": []any{"cat", "grep"},
				},
			}

			// Get the engine
			engine, err := GetGlobalEngineRegistry().GetEngine(tt.engine)
			if err != nil {
				t.Fatalf("Failed to get engine: %v", err)
			}

			// Get execution steps
			steps := engine.GetExecutionSteps(workflowData, "/tmp/detection.log")

			// Convert steps to string for analysis
			var stepsStr strings.Builder
			for _, step := range steps {
				for _, line := range step {
					stepsStr.WriteString(line)
					stepsStr.WriteString("\n")
				}
			}
			stepsContent := stepsStr.String()

			// Check that the environment variable is present
			if !strings.Contains(stepsContent, tt.expectedEnvVar+":") {
				t.Errorf("Expected environment variable %s not found in detection steps:\n%s", tt.expectedEnvVar, stepsContent)
			}

			// For Copilot, verify it has the default detection model as fallback
			if tt.expectedDefault != "" {
				expectedEnvLine := tt.expectedEnvVar + ": ${{ vars." + tt.expectedEnvVar + " || '" + tt.expectedDefault + "' }}"
				if !strings.Contains(stepsContent, expectedEnvLine) {
					t.Errorf("Expected env var line with default '%s' not found in steps:\n%s", expectedEnvLine, stepsContent)
				}
			} else {
				// For other engines, verify empty string fallback
				expectedEnvLine := tt.expectedEnvVar + ": ${{ vars." + tt.expectedEnvVar + " || '' }}"
				if !strings.Contains(stepsContent, expectedEnvLine) {
					t.Errorf("Expected env var line '%s' not found in steps:\n%s", expectedEnvLine, stepsContent)
				}
			}
		})
	}
}

// TestExplicitModelConfigOverridesEnvVar tests that explicit model configuration takes precedence
func TestExplicitModelConfigOverridesEnvVar(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-explicit-model",
		AI:   "copilot",
		EngineConfig: &EngineConfig{
			ID:    "copilot",
			Model: "gpt-4",
		},
		Tools: map[string]any{
			"bash": []any{"echo"},
		},
		SafeOutputs: &SafeOutputsConfig{
			// Just enough to make it an agent job
		},
	}

	engine, err := GetGlobalEngineRegistry().GetEngine("copilot")
	if err != nil {
		t.Fatalf("Failed to get engine: %v", err)
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Convert steps to string
	var stepsStr strings.Builder
	for _, step := range steps {
		for _, line := range step {
			stepsStr.WriteString(line)
			stepsStr.WriteString("\n")
		}
	}
	stepsContent := stepsStr.String()

	// When model is explicitly configured, the GH_AW_ fallback env var should NOT be present
	if strings.Contains(stepsContent, constants.EnvVarModelAgentCopilot+":") {
		t.Errorf("Fallback env var %s should not be present when model is explicitly configured", constants.EnvVarModelAgentCopilot)
	}

	// The model should be passed via the native COPILOT_MODEL env var (not via --model flag)
	expectedEnvLine := constants.CopilotCLIModelEnvVar + ": gpt-4"
	if !strings.Contains(stepsContent, expectedEnvLine) {
		t.Errorf("Expected native env var line '%s' not found in steps:\n%s", expectedEnvLine, stepsContent)
	}

	// The --model flag should NOT appear in the shell command (model is via env var)
	if strings.Contains(stepsContent, "--model gpt-4") {
		t.Errorf("--model flag should not be in command when model is set via native env var:\n%s", stepsContent)
	}
}

// TestExpressionModelUsesEnvVar tests that when model is a GitHub Actions expression,
// it is set as an environment variable rather than embedded directly in the shell command.
// This prevents template injection validation failures.
func TestExpressionModelUsesEnvVar(t *testing.T) {
	tests := []struct {
		name                 string
		engine               string
		model                string
		expectedEnvVar       string
		expectedEnvVal       string
		expectShellExpansion bool // whether command should use ${VAR:+ --model "$VAR"}
	}{
		{
			name:                 "Copilot agent with inputs.model expression uses native COPILOT_MODEL",
			engine:               "copilot",
			model:                "${{ inputs.model }}",
			expectedEnvVar:       constants.CopilotCLIModelEnvVar,
			expectedEnvVal:       "${{ inputs.model }}",
			expectShellExpansion: false, // Copilot reads COPILOT_MODEL natively, no shell expansion needed
		},
		{
			name:                 "Copilot agent with vars.model expression uses native COPILOT_MODEL",
			engine:               "copilot",
			model:                "${{ vars.MY_MODEL }}",
			expectedEnvVar:       constants.CopilotCLIModelEnvVar,
			expectedEnvVal:       "${{ vars.MY_MODEL }}",
			expectShellExpansion: false,
		},
		{
			name:                 "Claude agent with inputs.model expression uses native ANTHROPIC_MODEL",
			engine:               "claude",
			model:                "${{ inputs.model }}",
			expectedEnvVar:       constants.ClaudeCLIModelEnvVar,
			expectedEnvVal:       "${{ inputs.model }}",
			expectShellExpansion: false, // Claude reads ANTHROPIC_MODEL natively, no shell expansion needed
		},
		{
			name:                 "Codex agent with inputs.model expression",
			engine:               "codex",
			model:                "${{ inputs.model }}",
			expectedEnvVar:       constants.EnvVarModelAgentCodex,
			expectedEnvVal:       "${{ inputs.model }}",
			expectShellExpansion: true, // Codex has no native model env var, uses shell expansion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name: "test-expression-model",
				AI:   tt.engine,
				EngineConfig: &EngineConfig{
					ID:    tt.engine,
					Model: tt.model,
				},
				Tools: map[string]any{
					"bash": []any{"echo"},
				},
				SafeOutputs: &SafeOutputsConfig{},
			}

			engine, err := GetGlobalEngineRegistry().GetEngine(tt.engine)
			if err != nil {
				t.Fatalf("Failed to get engine: %v", err)
			}

			steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

			var stepsStr strings.Builder
			for _, step := range steps {
				for _, line := range step {
					stepsStr.WriteString(line)
					stepsStr.WriteString("\n")
				}
			}
			stepsContent := stepsStr.String()

			// The expression must NOT appear directly in the shell command run: block
			// (it should be in the env: block only)
			if strings.Contains(stepsContent, "--model ${{") || strings.Contains(stepsContent, "--model \"${{") {
				t.Errorf("Model expression should not be embedded directly in shell command in steps:\n%s", stepsContent)
			}

			// The env var must be set to the expression value
			expectedEnvLine := tt.expectedEnvVar + ": " + tt.expectedEnvVal
			if !strings.Contains(stepsContent, expectedEnvLine) {
				t.Errorf("Expected env line '%s' not found in steps:\n%s", expectedEnvLine, stepsContent)
			}

			// Check shell expansion expectation
			shellExpansionPattern := "${" + tt.expectedEnvVar + ":+"
			hasShellExpansion := strings.Contains(stepsContent, shellExpansionPattern)
			if tt.expectShellExpansion && !hasShellExpansion {
				t.Errorf("Expected conditional env var usage '${%s:+' not found in steps:\n%s", tt.expectedEnvVar, stepsContent)
			} else if !tt.expectShellExpansion && hasShellExpansion {
				t.Errorf("Unexpected conditional env var usage '${%s:+' found in steps (should use native env var):\n%s", tt.expectedEnvVar, stepsContent)
			}
		})
	}
}

// TestExpressionModelDetectionJobUsesEnvVar tests that detection jobs with expression model
// for Copilot use the native COPILOT_MODEL environment variable.
func TestExpressionModelDetectionJobUsesEnvVar(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-detection-expression-model",
		AI:   "copilot",
		EngineConfig: &EngineConfig{
			ID:    "copilot",
			Model: "${{ inputs.model }}",
		},
		Tools: map[string]any{
			"bash": []any{"cat", "grep"},
		},
		SafeOutputs: nil, // detection job
	}

	engine, err := GetGlobalEngineRegistry().GetEngine("copilot")
	if err != nil {
		t.Fatalf("Failed to get engine: %v", err)
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/detection.log")

	var stepsStr strings.Builder
	for _, step := range steps {
		for _, line := range step {
			stepsStr.WriteString(line)
			stepsStr.WriteString("\n")
		}
	}
	stepsContent := stepsStr.String()

	// Detection job for Copilot should use COPILOT_MODEL (native CLI env var)
	expectedEnvLine := constants.CopilotCLIModelEnvVar + ": ${{ inputs.model }}"
	if !strings.Contains(stepsContent, expectedEnvLine) {
		t.Errorf("Expected env line '%s' not found in steps:\n%s", expectedEnvLine, stepsContent)
	}

	// Must not embed expression directly in shell command
	if strings.Contains(stepsContent, "--model ${{") {
		t.Errorf("Model expression should not be embedded directly in shell command:\n%s", stepsContent)
	}
}

// TestGetModelEnvVarName tests that engines return the correct native model env var name.
func TestGetModelEnvVarName(t *testing.T) {
	tests := []struct {
		engine   string
		expected string
	}{
		{"copilot", constants.CopilotCLIModelEnvVar}, // "COPILOT_MODEL"
		{"claude", constants.ClaudeCLIModelEnvVar},   // "ANTHROPIC_MODEL"
		{"codex", ""}, // no native model env var
		{"gemini", constants.GeminiCLIModelEnvVar}, // "GEMINI_MODEL"
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			eng, err := GetGlobalEngineRegistry().GetEngine(tt.engine)
			if err != nil {
				t.Fatalf("Failed to get engine %s: %v", tt.engine, err)
			}
			provider, ok := eng.(ModelEnvVarProvider)
			if !ok {
				t.Fatalf("Engine %s does not implement ModelEnvVarProvider", tt.engine)
			}
			if got := provider.GetModelEnvVarName(); got != tt.expected {
				t.Errorf("Engine %s: GetModelEnvVarName() = %q, want %q", tt.engine, got, tt.expected)
			}
		})
	}
}
