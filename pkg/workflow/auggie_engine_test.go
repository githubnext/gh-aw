//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuggieEngineSecretValidationUsesSessionSecret(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("auggie")
	require.NoError(t, err)

	workflowData := &WorkflowData{
		Name: "test-auggie",
		Tools: map[string]any{
			"github": map[string]any{"toolsets": []string{"repos"}},
		},
		ParsedTools: &ToolsConfig{
			GitHub: &GitHubToolConfig{},
		},
	}

	step := engine.GetSecretValidationStep(workflowData)
	stepContent := strings.Join(step, "\n")

	assert.Contains(t, stepContent, "AUGMENT_SESSION_AUTH", "Auggie should validate its session secret")
	assert.NotContains(t, stepContent, "GITHUB_MCP_SERVER_TOKEN", "Auggie secret validation should not accept unrelated GitHub MCP secrets as substitutes")
}

func TestAuggieEngineExecutionUsesScopedModelEnvVars(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("auggie")
	require.NoError(t, err)

	t.Run("agent run uses agent model override env var", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "auggie-agent"}, "/tmp/test.log")
		stepContent := strings.Join(steps[len(steps)-1], "\n")

		assert.Contains(t, stepContent, constants.EnvVarModelAgentAuggie+": ${{ vars."+constants.EnvVarModelAgentAuggie+" || '' }}")
		assert.Contains(t, stepContent, `${`+constants.EnvVarModelAgentAuggie+`:+ --model "$`+constants.EnvVarModelAgentAuggie+`"}`)
		assert.NotContains(t, stepContent, constants.EnvVarModelDetectionAuggie+":")
	})

	t.Run("detection run uses detection model override env var", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{
			Name:           "auggie-detection",
			IsDetectionRun: true,
		}, "/tmp/test.log")
		stepContent := strings.Join(steps[len(steps)-1], "\n")

		assert.Contains(t, stepContent, constants.EnvVarModelDetectionAuggie+": ${{ vars."+constants.EnvVarModelDetectionAuggie+" || '' }}")
		assert.Contains(t, stepContent, `${`+constants.EnvVarModelDetectionAuggie+`:+ --model "$`+constants.EnvVarModelDetectionAuggie+`"}`)
		assert.NotContains(t, stepContent, constants.EnvVarModelAgentAuggie+":")
	})
}

func TestAuggieEngineUsesRunnerTempExpressionForMCPConfig(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("auggie")
	require.NoError(t, err)

	steps := engine.GetExecutionSteps(&WorkflowData{
		Name: "auggie-mcp",
		Tools: map[string]any{
			"github": map[string]any{"toolsets": []string{"repos"}},
		},
		ParsedTools: &ToolsConfig{
			GitHub: &GitHubToolConfig{},
		},
	}, "/tmp/test.log")
	stepContent := strings.Join(steps[len(steps)-1], "\n")

	assert.Contains(t, stepContent, "${{ runner.temp }}/gh-aw/mcp-config/mcp-servers.json")
	assert.NotContains(t, stepContent, "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json")
}

func TestAuggieEngineAWFExcludesAuxiliaryButNotSessionSecret(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("auggie")
	require.NoError(t, err)

	steps := engine.GetExecutionSteps(&WorkflowData{
		Name: "auggie-awf",
		Tools: map[string]any{
			"github": map[string]any{"toolsets": []string{"repos"}},
		},
		ParsedTools: &ToolsConfig{
			GitHub: &GitHubToolConfig{},
		},
		NetworkPermissions: &NetworkPermissions{
			Allowed: []string{"defaults", "github"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}, "/tmp/test.log")
	stepContent := strings.Join(steps[len(steps)-1], "\n")

	assert.Contains(t, stepContent, "--exclude-env GITHUB_MCP_SERVER_TOKEN")
	assert.NotContains(t, stepContent, "--exclude-env AUGMENT_SESSION_AUTH")
	// The runtime still needs the session secret bound in the execution env.
	assert.Contains(t, stepContent, "AUGMENT_SESSION_AUTH: ${{ secrets.AUGMENT_SESSION_AUTH }}")
}
