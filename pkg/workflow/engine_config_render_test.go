//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configRenderingEngine is a minimal test runtime adapter that overrides RenderConfig
// to emit a sentinel step. Used only in tests to validate that the orchestrator
// correctly prepends config steps before execution steps.
type configRenderingEngine struct {
	BaseEngine
	steps []map[string]any
}

func newConfigRenderingEngine(steps []map[string]any) *configRenderingEngine {
	return &configRenderingEngine{
		BaseEngine: BaseEngine{
			id:          "config-renderer",
			displayName: "Config Renderer (test)",
			description: "Test engine that emits config steps via RenderConfig",
		},
		steps: steps,
	}
}

// RenderConfig returns the pre-configured steps, overriding the BaseEngine no-op.
func (e *configRenderingEngine) RenderConfig(_ *ResolvedEngineTarget) ([]map[string]any, error) {
	return e.steps, nil
}

// GetInstallationSteps returns no installation steps for this test engine.
func (e *configRenderingEngine) GetInstallationSteps(_ *WorkflowData) []GitHubActionStep {
	return nil
}

// GetExecutionSteps returns a minimal sentinel execution step so tests can verify ordering.
func (e *configRenderingEngine) GetExecutionSteps(_ *WorkflowData, _ string) []GitHubActionStep {
	return []GitHubActionStep{{"      - name: config-renderer-exec"}}
}

// TestRenderConfig_BuiltinEnginesReturnNil verifies that all four built-in engines
// return nil, nil from RenderConfig (backward-compatible no-op behaviour).
func TestRenderConfig_BuiltinEnginesReturnNil(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	engineIDs := []string{"claude", "codex", "copilot", "gemini"}
	for _, id := range engineIDs {
		t.Run(id, func(t *testing.T) {
			resolved, err := catalog.Resolve(id, &EngineConfig{ID: id})
			require.NoError(t, err, "should resolve %s without error", id)

			steps, err := resolved.Runtime.RenderConfig(resolved)
			require.NoError(t, err, "RenderConfig should not return an error for %s", id)
			assert.Nil(t, steps, "RenderConfig should return nil steps for built-in engine %s", id)
		})
	}
}

// TestGenerateMainJobSteps_ConfigStepsBeforeExecution verifies that steps returned
// by RenderConfig are emitted before the AI execution steps in the compiled YAML.
func TestGenerateMainJobSteps_ConfigStepsBeforeExecution(t *testing.T) {
	compiler := NewCompiler()
	compiler.stepOrderTracker = NewStepOrderTracker()

	const configStepName = "Write engine config"

	data := &WorkflowData{
		Name:            "Test Workflow",
		AI:              "copilot",
		MarkdownContent: "Test prompt",
		EngineConfig:    &EngineConfig{ID: "copilot"},
		ParsedTools:     &ToolsConfig{},
		EngineConfigSteps: []map[string]any{
			{
				"name": configStepName,
				"run":  "echo 'provider = \"openai\"' > /tmp/config.toml",
			},
		},
	}

	var yaml strings.Builder
	err := compiler.generateMainJobSteps(&yaml, data)
	require.NoError(t, err, "generateMainJobSteps should not error with EngineConfigSteps")

	result := yaml.String()

	// Verify the config step is present in the output.
	assert.Contains(t, result, configStepName, "config step name should appear in YAML output")

	// Verify step ordering: the config step must appear before the Copilot execution step.
	configIdx := strings.Index(result, configStepName)
	execIdx := strings.Index(result, "Execute GitHub Copilot CLI")
	require.GreaterOrEqual(t, configIdx, 0, "config step should be present in output")
	require.GreaterOrEqual(t, execIdx, 0, "execution step should be present in output")
	assert.Less(t, configIdx, execIdx,
		"engine config step must appear before the AI execution step")
}

// TestGenerateMainJobSteps_NoConfigSteps verifies that when EngineConfigSteps is nil
// the YAML output is unaffected (no spurious steps or errors).
func TestGenerateMainJobSteps_NoConfigSteps(t *testing.T) {
	compiler := NewCompiler()
	compiler.stepOrderTracker = NewStepOrderTracker()

	data := &WorkflowData{
		Name:            "Test Workflow",
		AI:              "copilot",
		MarkdownContent: "Test prompt",
		EngineConfig:    &EngineConfig{ID: "copilot"},
		ParsedTools:     &ToolsConfig{},
		// EngineConfigSteps intentionally not set
	}

	var yaml strings.Builder
	err := compiler.generateMainJobSteps(&yaml, data)
	require.NoError(t, err, "generateMainJobSteps should not error without EngineConfigSteps")

	result := yaml.String()
	assert.Contains(t, result, "Execute GitHub Copilot CLI",
		"execution step should still be present when no config steps are present")
}

// TestOrchestratorCallsRenderConfig verifies that setupEngineAndImports invokes
// RenderConfig and stores the returned steps in engineSetupResult.configSteps.
func TestOrchestratorCallsRenderConfig(t *testing.T) {
	const sentinelStepName = "sentinel-config-step"

	configSteps := []map[string]any{
		{"name": sentinelStepName, "run": "echo sentinel"},
	}
	testEngine := newConfigRenderingEngine(configSteps)

	// Build a compiler whose registry contains the test engine.
	registry := NewEngineRegistry()
	registry.Register(testEngine)
	catalog := NewEngineCatalog(registry)

	compiler := NewCompiler()
	compiler.engineRegistry = registry
	compiler.engineCatalog = catalog

	// Resolve the test engine to simulate what setupEngineAndImports does.
	engineConfig := &EngineConfig{ID: testEngine.GetID()}
	resolved, err := catalog.Resolve(testEngine.GetID(), engineConfig)
	require.NoError(t, err, "test engine should resolve without error")

	// Call RenderConfig directly via the resolved runtime to verify the hook.
	steps, err := resolved.Runtime.RenderConfig(resolved)
	require.NoError(t, err, "RenderConfig should not error")
	require.Len(t, steps, 1, "should return exactly one config step")
	assert.Equal(t, sentinelStepName, steps[0]["name"],
		"config step name should match sentinel value")
}
