//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPydanticEngine(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("pydantic")
	require.NoError(t, err)

	t.Run("engine identity", func(t *testing.T) {
		assert.Equal(t, "pydantic", engine.GetID(), "Engine ID should be 'pydantic'")
		assert.Equal(t, "Pydantic AI", engine.GetDisplayName(), "Display name should be 'Pydantic AI'")
		assert.NotEmpty(t, engine.GetDescription(), "Description should not be empty")
		assert.True(t, engine.IsExperimental(), "Pydantic engine should be experimental")
	})

	t.Run("capabilities", func(t *testing.T) {
		capabilities := engine.GetCapabilities()
		assert.True(t, capabilities.MaxTurns, "Should support max turns")
	})

	t.Run("model env var name", func(t *testing.T) {
		assert.Equal(t, "PYDANTIC_AI_MODEL", engine.GetModelEnvVarName(), "Should return PYDANTIC_AI_MODEL")
	})

	t.Run("required secrets basic", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should require COPILOT_GITHUB_TOKEN for Copilot routing")
	})

	t.Run("required secrets with anthropic model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			EngineConfig: &EngineConfig{
				Model: "anthropic/claude-sonnet-4-20250514",
			},
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "ANTHROPIC_API_KEY", "Should require ANTHROPIC_API_KEY for anthropic/* models")
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN for anthropic/* models")
	})

	t.Run("required secrets with copilot-requests permission", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
			Permissions: "permissions:\n  copilot-requests: write",
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN when permissions.copilot-requests is write")
	})

	t.Run("declared output files", func(t *testing.T) {
		outputFiles := engine.GetDeclaredOutputFiles()
		assert.Empty(t, outputFiles, "Should have no declared output files")
	})

	t.Run("agent manifest files", func(t *testing.T) {
		files := engine.GetAgentManifestFiles()
		assert.Contains(t, files, "AGENTS.md", "Should include AGENTS.md")
	})
}

func TestPydanticEngineInstallation(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("pydantic")
	require.NoError(t, err)

	t.Run("standard installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		// Should have at least: uv setup + install pydantic-ai + verify
		assert.GreaterOrEqual(t, len(steps), 3, "Should have at least 3 installation steps")

		// Find install step
		var installStep string
		for _, step := range steps {
			content := strings.Join(step, "\n")
			if strings.Contains(content, "pydantic-ai") && strings.Contains(content, "uv tool install") {
				installStep = content
				break
			}
		}
		require.NotEmpty(t, installStep, "Should find a step installing pydantic-ai")

		// Find verify step
		var verifyStep string
		for _, step := range steps {
			content := strings.Join(step, "\n")
			if strings.Contains(content, "pydantic-ai --version") {
				verifyStep = content
				break
			}
		}
		require.NotEmpty(t, verifyStep, "Should find pydantic-ai --version verify step")
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/pydantic-ai",
			},
		}

		steps := engine.GetInstallationSteps(workflowData)
		assert.Empty(t, steps, "Should skip installation when custom command is specified")
	})

	t.Run("with firewall includes AWF install", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		hasAWFInstall := false
		for _, step := range steps {
			stepContent := strings.Join(step, "\n")
			if strings.Contains(stepContent, "awf") || strings.Contains(stepContent, "firewall") {
				hasAWFInstall = true
				break
			}
		}
		assert.True(t, hasAWFInstall, "Should include AWF installation step when firewall is enabled")
	})
}

func TestPydanticEngineExecution(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("pydantic")
	require.NoError(t, err)

	t.Run("basic execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 1, "Should generate one execution step")

		stepContent := strings.Join(steps[0], "\n")

		assert.Contains(t, stepContent, "name: Execute Pydantic AI", "Should have correct step name")
		assert.Contains(t, stepContent, "id: agentic_execution", "Should have agentic_execution ID")
		assert.Contains(t, stepContent, "pydantic-ai run", "Should invoke pydantic-ai run command")
		assert.Contains(t, stepContent, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, "Should include prompt argument")
		assert.Contains(t, stepContent, "/tmp/test.log", "Should include log file")
		assert.Contains(t, stepContent, "NO_PROXY: "+constants.AWFNoProxyHosts, "Should set NO_PROXY env var")
	})

	t.Run("with model sets PYDANTIC_AI_MODEL", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Model: "copilot/gpt-4o",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 1, "Should generate one execution step")

		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "PYDANTIC_AI_MODEL: copilot/gpt-4o", "Should set PYDANTIC_AI_MODEL env var")
	})

	t.Run("basic execution with copilot-requests permission", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test-workflow",
			Permissions: "permissions:\n  copilot-requests: write",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 1, "Should generate one execution step")

		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "OPENAI_API_KEY: ${{ github.token }}", "Should set OPENAI_API_KEY from github.token when permissions.copilot-requests is write")
	})
}
