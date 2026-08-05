//go:build !integration

package workflow

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineRegistry(t *testing.T) {
	t.Run("built-in engines are registered", func(t *testing.T) {
		registry := NewEngineRegistry()
		supportedEngines := registry.GetSupportedEngines()

		expectedEngineIDs := []string{"claude", "codex", "copilot", "gemini"}
		for _, engineID := range expectedEngineIDs {
			assert.True(t, slices.Contains(supportedEngines, engineID), "expected engine %q to be registered", engineID)
		}
	})

	t.Run("GetEngine returns engine by ID", func(t *testing.T) {
		tests := []struct {
			engineID string
		}{
			{engineID: "claude"},
			{engineID: "codex"},
			{engineID: "copilot"},
			{engineID: "gemini"},
		}

		for _, tt := range tests {
			t.Run(tt.engineID, func(t *testing.T) {
				registry := NewEngineRegistry()
				engine, err := registry.GetEngine(tt.engineID)
				require.NoError(t, err, "GetEngine(%q) should not return an error", tt.engineID)
				assert.Equal(t, tt.engineID, engine.GetID(), "engine ID should match requested ID")
			})
		}
	})

	t.Run("GetEngine returns error for unknown engine", func(t *testing.T) {
		registry := NewEngineRegistry()
		_, err := registry.GetEngine("nonexistent")
		assert.Error(t, err, "GetEngine should return an error for unknown engine ID")
	})

	t.Run("IsValidEngine", func(t *testing.T) {
		registry := NewEngineRegistry()

		validEngines := []string{"claude", "codex", "copilot", "gemini"}
		for _, id := range validEngines {
			assert.True(t, registry.IsValidEngine(id), "IsValidEngine(%q) should return true", id)
		}

		assert.False(t, registry.IsValidEngine("nonexistent"), "IsValidEngine should return false for unknown engine ID")
	})

	t.Run("GetDefaultEngine returns copilot", func(t *testing.T) {
		registry := NewEngineRegistry()
		defaultEngine := registry.GetDefaultEngine()
		require.NotNil(t, defaultEngine, "default engine should not be nil")
		assert.Equal(t, "copilot", defaultEngine.GetID(), "default engine should be copilot")
	})

	t.Run("GetEngineByPrefix matches engine", func(t *testing.T) {
		registry := NewEngineRegistry()
		engine, err := registry.GetEngineByPrefix("codex-experimental")
		require.NoError(t, err, "GetEngineByPrefix should not return an error for a valid prefix")
		assert.Equal(t, "codex", engine.GetID(), "engine matched by prefix 'codex-experimental' should be codex")
	})

	t.Run("GetEngineByPrefix returns error for non-matching prefix", func(t *testing.T) {
		registry := NewEngineRegistry()
		_, err := registry.GetEngineByPrefix("nonexistent-prefix")
		assert.Error(t, err, "GetEngineByPrefix should return an error for a non-matching prefix")
	})
}

func TestEngineRegistry_Register(t *testing.T) {
	t.Run("custom engine can be registered and retrieved", func(t *testing.T) {
		// Use direct struct initialization to start with an empty registry so
		// Register is the sole mechanism populating it in this test.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		customEngine := NewCopilotEngine()

		err := registry.Register(customEngine)
		require.NoError(t, err, "registering a valid engine should not return an error")

		engine, err := registry.GetEngine("copilot")
		require.NoError(t, err, "registered custom engine should be retrievable")
		assert.Equal(t, "copilot", engine.GetID(), "retrieved engine ID should match registered engine")
	})

	t.Run("registering an engine makes IsValidEngine return true", func(t *testing.T) {
		// Use direct struct initialization to start with an empty registry so
		// IsValidEngine behaviour before and after Register is clearly observable.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		assert.False(t, registry.IsValidEngine("claude"), "engine should not be valid before registration")

		err := registry.Register(NewClaudeEngine())
		require.NoError(t, err, "registering a valid engine should not return an error")
		assert.True(t, registry.IsValidEngine("claude"), "engine should be valid after registration")
	})

	t.Run("registering an engine with negative dedicatedLLMGatewayPort returns error", func(t *testing.T) {
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}

		// negativePortEngine wraps ClaudeEngine and returns -1 from
		// getDedicatedLLMGatewayPort, triggering the validation path in Register.
		err := registry.Register(&negativePortEngine{CodingAgentEngine: NewClaudeEngine()})
		require.Error(t, err, "registering an engine with dedicatedLLMGatewayPort = -1 should return an error")
		require.ErrorContains(t, err, "dedicatedLLMGatewayPort must be >= 0", "error message should describe the constraint")
		assert.False(t, registry.IsValidEngine("claude"), "invalid engine should not be registered on error")
	})
}

// negativePortEngine wraps a CodingAgentEngine and always reports port -1 so
// that tests can exercise the negative-port validation path in Register without
// requiring a real misconfigured engine. It must be defined at package level
// because Go only allows method declarations on package-level types.
type negativePortEngine struct {
	CodingAgentEngine
}

func (e *negativePortEngine) getDedicatedLLMGatewayPort() int { return -1 }

func TestGetGlobalEngineRegistry(t *testing.T) {
	t.Run("returns non-nil registry", func(t *testing.T) {
		registry := GetGlobalEngineRegistry()
		require.NotNil(t, registry, "global engine registry should not be nil")
	})

	t.Run("returns same singleton on repeated calls", func(t *testing.T) {
		registry1 := GetGlobalEngineRegistry()
		registry2 := GetGlobalEngineRegistry()
		assert.Same(t, registry1, registry2, "GetGlobalEngineRegistry should return the same singleton instance")
	})

	t.Run("singleton contains expected built-in engines", func(t *testing.T) {
		registry := GetGlobalEngineRegistry()
		expectedEngineIDs := []string{"claude", "codex", "copilot", "gemini"}
		supportedEngines := registry.GetSupportedEngines()
		for _, engineID := range expectedEngineIDs {
			assert.True(t, slices.Contains(supportedEngines, engineID), "global registry should contain built-in engine %q", engineID)
		}
	})
}

func TestEngineRegistry_GetAllAgentManifestFolders(t *testing.T) {
	t.Run("always includes .agents platform directory", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		assert.Contains(t, folders, ".agents", "manifest folders should always include the .agents platform directory")
	})

	t.Run("result is sorted", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		for i := 1; i < len(folders); i++ {
			assert.LessOrEqual(t, folders[i-1], folders[i], "manifest folders should be sorted alphabetically")
		}
	})

	t.Run("includes engine-specific config directories", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		// Claude and Copilot engines provide known config directory prefixes
		expectedFolders := []string{".agents", ".claude", ".gemini", ".github"}
		for _, folder := range expectedFolders {
			assert.Contains(t, folders, folder, "manifest folders should include engine config directory %q", folder)
		}
	})

	t.Run("no duplicates in result", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		seen := make(map[string]struct{})
		for _, folder := range folders {
			assert.NotContains(t, seen, folder, "manifest folders should not contain duplicates, found %q twice", folder)
			seen[folder] = struct{}{}
		}
	})

	t.Run("empty registry still includes .agents", func(t *testing.T) {
		// Use direct struct initialization so there are no engines; this verifies
		// that .agents is always appended regardless of registered engines.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		folders := registry.GetAllAgentManifestFolders()
		assert.Equal(t, []string{".agents"}, folders, "empty registry should still return .agents")
	})
}

// TestAllEnginesEmitTimeoutMinutes verifies that every registered agentic engine
// emits a timeout-minutes field on the agentic_execution step. This prevents
// silent 6-hour timeouts when timeout-minutes is not forwarded to the step.
func TestAllEnginesEmitTimeoutMinutes(t *testing.T) {
	defaultTimeout := strconv.Itoa(int(constants.DefaultAgenticWorkflowTimeout / time.Minute))

	tests := []struct {
		name         string
		workflowData *WorkflowData
		wantTimeout  string
		description  string
	}{
		{
			name:         "no timeout set — default emitted",
			workflowData: &WorkflowData{Name: "test-workflow"},
			wantTimeout:  defaultTimeout,
			description:  "engine should emit the default timeout when none is specified",
		},
		{
			name:         "explicit timeout-minutes: 15",
			workflowData: &WorkflowData{Name: "test-workflow", ParsedFrontmatter: &FrontmatterConfig{TimeoutMinutes: func() *TemplatableInt32 { v := TemplatableInt32("15"); return &v }()}},
			wantTimeout:  "15",
			description:  "engine should forward the explicit timeout value",
		},
		{
			name:         "timeout as expression",
			workflowData: &WorkflowData{Name: "test-workflow", ParsedFrontmatter: &FrontmatterConfig{TimeoutMinutes: func() *TemplatableInt32 { v := TemplatableInt32("${{ inputs.timeout }}"); return &v }()}},
			wantTimeout:  "${{ inputs.timeout }}",
			description:  "engine should forward GitHub Actions expressions verbatim",
		},
		{
			name:         "legacy timeout-minutes field (numeric)",
			workflowData: &WorkflowData{Name: "test-workflow", TimeoutMinutes: "timeout-minutes: 30"},
			wantTimeout:  "30",
			description:  "engine should honor already-extracted timeout-minutes values",
		},
		{
			name:         "legacy timeout-minutes field (expression)",
			workflowData: &WorkflowData{Name: "test-workflow", TimeoutMinutes: "timeout-minutes: ${{ inputs.timeout }}"},
			wantTimeout:  "${{ inputs.timeout }}",
			description:  "engine should honor already-extracted expression timeout values",
		},
		{
			name:         "legacy timeout-minutes field — malformed value falls back to default",
			workflowData: &WorkflowData{Name: "test-workflow", TimeoutMinutes: "timeout-minutes: not-a-number\ninjected: yaml"},
			wantTimeout:  defaultTimeout,
			description:  "engine must fall back to default for non-integer, non-expression raw values to prevent malformed YAML",
		},
	}

	registry := NewEngineRegistry()
	engineIDs := registry.GetSupportedEngines()

	for _, engineID := range engineIDs {
		t.Run(engineID, func(t *testing.T) {
			engine, err := registry.GetEngine(engineID)
			require.NoError(t, err)

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					execSteps := engine.GetExecutionSteps(tt.workflowData, "/tmp/gh-aw/test.log")
					require.NotEmpty(t, execSteps, "engine must return at least one execution step")

					// Find the agentic_execution step (it may not be the first step for engines
					// that prepend settings-write steps, e.g. Gemini).
					var agentStep GitHubActionStep
					for _, step := range execSteps {
						stepContent := strings.Join([]string(step), "\n")
						if strings.Contains(stepContent, "id: agentic_execution") {
							agentStep = step
							break
						}
					}
					require.NotNil(t, agentStep, "execution steps must include a step with id: agentic_execution")

					stepContent := strings.Join([]string(agentStep), "\n")
					expected := "timeout-minutes: " + tt.wantTimeout
					assert.Contains(t, stepContent, expected,
						"%s: %s\ngot step:\n%s", engineID, tt.description, stepContent)
				})
			}
		})
	}
}
