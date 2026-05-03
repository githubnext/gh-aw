//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPiEngine(t *testing.T) {
	engine := NewPiEngine()
	require.NotNil(t, engine, "NewPiEngine should return a non-nil engine")
	assert.Equal(t, "pi", engine.GetID(), "Engine ID should be 'pi'")
	assert.Equal(t, "Pi", engine.GetDisplayName(), "Display name should be 'Pi'")
	assert.True(t, engine.IsExperimental(), "Pi engine should be experimental")
	assert.True(t, engine.SupportsToolsAllowlist(), "Pi should support tools allowlist (needed for gh-proxy/cli-proxy settings)")
	assert.False(t, engine.SupportsMaxTurns(), "Pi should not support max turns")
}

func TestPiEngine_GetModelEnvVarName(t *testing.T) {
	engine := NewPiEngine()
	assert.Equal(t, "PI_MODEL", engine.GetModelEnvVarName(), "Model env var should be PI_MODEL")
}

func TestPiEngine_GetRequiredSecretNames(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{Name: "test-workflow"}
	secrets := engine.GetRequiredSecretNames(workflowData)
	assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Required secrets should include COPILOT_GITHUB_TOKEN")
	assert.NotContains(t, secrets, "PI_API_KEY", "Required secrets should not include PI_API_KEY")
}

func TestPiEngine_GetLogParserScriptId(t *testing.T) {
	engine := NewPiEngine()
	assert.Equal(t, "parse_pi_log", engine.GetLogParserScriptId(), "Log parser script ID should be parse_pi_log")
}

func TestPiEngine_GetLogFileForParsing(t *testing.T) {
	engine := NewPiEngine()
	assert.Equal(t, PiStreamingLogFile, engine.GetLogFileForParsing(), "Log file for parsing should be PiStreamingLogFile")
}

func TestPiEngine_GetAgentManifestFiles(t *testing.T) {
	engine := NewPiEngine()
	files := engine.GetAgentManifestFiles()
	assert.Contains(t, files, "PI.md", "Manifest files should include PI.md")
	assert.Contains(t, files, "AGENTS.md", "Manifest files should include AGENTS.md")
}

func TestPiEngine_GetAgentManifestPathPrefixes(t *testing.T) {
	engine := NewPiEngine()
	prefixes := engine.GetAgentManifestPathPrefixes()
	assert.Contains(t, prefixes, ".pi/", "Path prefixes should include .pi/")
}

func TestPiEngine_GetDeclaredOutputFiles(t *testing.T) {
	engine := NewPiEngine()
	files := engine.GetDeclaredOutputFiles()
	assert.Contains(t, files, PiStreamingLogFile, "Declared output files should include the streaming log")
}

func TestPiEngine_GetInstallationSteps_NoCustomCommand(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi"},
	}
	steps := engine.GetInstallationSteps(workflowData)
	assert.NotEmpty(t, steps, "Installation steps should not be empty")

	// The steps should reference @pi/cli
	found := false
	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "@pi/cli") {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Installation steps should install @pi/cli")
}

func TestPiEngine_GetInstallationSteps_WithCustomCommand(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Command: "/custom/pi"},
	}
	steps := engine.GetInstallationSteps(workflowData)
	assert.Empty(t, steps, "Installation steps should be skipped when custom command is set")
}

func TestPiEngine_GetInstallationSteps_WithExtensions(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:         "pi",
			Extensions: []string{"@pi/web-search", "@pi/file-browser"},
		},
	}
	steps := engine.GetInstallationSteps(workflowData)
	require.NotEmpty(t, steps, "Steps should not be empty with extensions")

	// Find extension install steps
	var extensionSteps []GitHubActionStep
	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "Install Pi extension") {
				extensionSteps = append(extensionSteps, step)
				break
			}
		}
	}
	assert.Len(t, extensionSteps, 2, "Should have 2 extension install steps")
}

func TestPiEngine_GetExecutionSteps_Basic(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi"},
		ParsedTools:  NewTools(map[string]any{}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "Execute Pi CLI", "Step should be named 'Execute Pi CLI'")
	assert.Contains(t, stepText, "pi run", "Step should run `pi run`")
	assert.Contains(t, stepText, "json-log", "Step should include JSON log flag")
	assert.Contains(t, stepText, "agentic_execution", "Step should have agentic_execution id")
	assert.Contains(t, stepText, "pi_steering_extension.cjs", "Step should automatically load the steering extension")
}

func TestPiEngine_GetExecutionSteps_WithModel(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "pi-3"},
		ParsedTools:  NewTools(map[string]any{}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.NotEmpty(t, steps, "Steps should not be empty")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "PI_MODEL", "Step env should include PI_MODEL when model is configured")
	assert.Contains(t, stepText, "pi-3", "Step env should include the model value")
}

func TestPiEngine_ImplementsCodingAgentEngine(t *testing.T) {
	var _ CodingAgentEngine = NewPiEngine()
}
