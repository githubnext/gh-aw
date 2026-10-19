//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDefaultAgentModel verifies that each known engine returns the correct
// built-in default model string, and that unknown engines return an empty string.
func TestGetDefaultAgentModel(t *testing.T) {
	tests := []struct {
		engineID      string
		expectedModel string
	}{
		{
			engineID:      string(constants.CopilotEngine),
			expectedModel: constants.CopilotBYOKDefaultModel,
		},
		{
			engineID:      string(constants.ClaudeEngine),
			expectedModel: constants.AgentDefaultModel,
		},
		{
			engineID:      string(constants.GeminiEngine),
			expectedModel: constants.AgentDefaultModel,
		},
		{
			engineID:      string(constants.PiEngine),
			expectedModel: constants.AgentDefaultModel,
		},
		{
			engineID:      string(constants.CodexEngine),
			expectedModel: constants.CodexDefaultModel,
		},
		{
			engineID:      "unknown-engine",
			expectedModel: "",
		},
		{
			engineID:      "",
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.engineID, func(t *testing.T) {
			got := getDefaultAgentModel(tt.engineID)
			assert.Equal(t, tt.expectedModel, got, "getDefaultAgentModel(%q)", tt.engineID)
		})
	}
}

// TestGetDefaultModelOverrideVar verifies that each known engine returns the correct
// enterprise override variable name, and that engines without an override return "".
func TestGetDefaultModelOverrideVar(t *testing.T) {
	tests := []struct {
		engineID    string
		expectedVar string
	}{
		{
			engineID:    string(constants.CopilotEngine),
			expectedVar: compilerenv.DefaultModelCopilot,
		},
		{
			engineID:    string(constants.ClaudeEngine),
			expectedVar: compilerenv.DefaultModelClaude,
		},
		{
			engineID:    string(constants.CodexEngine),
			expectedVar: compilerenv.DefaultModelCodex,
		},
		{
			// GeminiEngine has no enterprise default override variable.
			engineID:    string(constants.GeminiEngine),
			expectedVar: "",
		},
		{
			engineID:    string(constants.PiEngine),
			expectedVar: "",
		},
		{
			engineID:    "unknown-engine",
			expectedVar: "",
		},
		{
			engineID:    "",
			expectedVar: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.engineID, func(t *testing.T) {
			got := getDefaultModelOverrideVar(tt.engineID)
			assert.Equal(t, tt.expectedVar, got, "getDefaultModelOverrideVar(%q)", tt.engineID)
		})
	}
}

func TestModelSelectionSource(t *testing.T) {
	tests := []struct {
		name        string
		data        *WorkflowData
		engineID    string
		modelEnvVar string
		expected    string
	}{
		{
			name:        "workflow model",
			data:        &WorkflowData{Model: "gpt-5"},
			engineID:    string(constants.CopilotEngine),
			modelEnvVar: constants.EnvVarModelAgentCopilot,
			expected:    "workflow engine.model",
		},
		{
			name:        "team default",
			data:        &WorkflowData{},
			engineID:    string(constants.CopilotEngine),
			modelEnvVar: constants.EnvVarModelAgentCopilot,
			expected:    "${{ vars.GH_AW_MODEL_AGENT_COPILOT && 'GH_AW_MODEL_AGENT_COPILOT' || vars.GH_AW_DEFAULT_MODEL_COPILOT && 'GH_AW_DEFAULT_MODEL_COPILOT' || 'copilot built-in default' }}",
		},
		{
			name:        "model routing",
			data:        &WorkflowData{EngineConfig: &EngineConfig{ModelRouting: &CopilotModelRoutingConfig{}}},
			engineID:    string(constants.CopilotEngine),
			modelEnvVar: constants.EnvVarModelAgentCopilot,
			expected:    "model routing",
		},
		{
			name:        "engine default without team override",
			data:        &WorkflowData{},
			engineID:    string(constants.GeminiEngine),
			modelEnvVar: constants.EnvVarModelAgentCustom,
			expected:    "${{ vars.GH_AW_MODEL_AGENT_CUSTOM && 'GH_AW_MODEL_AGENT_CUSTOM' || 'engine default' }}",
		},
		{
			name:        "codex native default",
			data:        &WorkflowData{},
			engineID:    string(constants.CodexEngine),
			modelEnvVar: constants.EnvVarModelAgentCodex,
			expected:    "${{ vars.GH_AW_MODEL_AGENT_CODEX && 'GH_AW_MODEL_AGENT_CODEX' || vars.GH_AW_DEFAULT_MODEL_CODEX && 'GH_AW_DEFAULT_MODEL_CODEX' || 'engine default' }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, modelSelectionSource(tt.data, tt.engineID, tt.modelEnvVar))
		})
	}
}

func TestCompiledModelSelectionSource(t *testing.T) {
	tests := []struct {
		name           string
		engineConfig   string
		expectedSource string
	}{
		{
			name:           "team default",
			engineConfig:   "engine: copilot",
			expectedSource: "${{ vars.GH_AW_MODEL_AGENT_COPILOT && 'GH_AW_MODEL_AGENT_COPILOT' || vars.GH_AW_DEFAULT_MODEL_COPILOT && 'GH_AW_DEFAULT_MODEL_COPILOT' || 'copilot built-in default' }}",
		},
		{
			name:           "workflow model",
			engineConfig:   "engine:\n  id: copilot\n  model: gpt-5.4",
			expectedSource: "workflow engine.model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := testutil.TempDir(t, "test-model-selection-source-*")
			workflowPath := filepath.Join(testDir, "workflow.md")
			workflow := "---\non: workflow_dispatch\n" + tt.engineConfig + "\n---\n\nTest workflow\n"
			require.NoError(t, os.WriteFile(workflowPath, []byte(workflow), 0600))

			require.NoError(t, NewCompiler().CompileWorkflow(workflowPath))
			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lock, err := os.ReadFile(lockPath)
			require.NoError(t, err)
			require.Contains(t, string(lock), "GH_AW_INFO_MODEL_SOURCE: "+tt.expectedSource)
		})
	}
}
