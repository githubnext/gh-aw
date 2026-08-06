//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPGatewayCustomEnvValuesStayOutOfRunScript(t *testing.T) {
	gatewayEnv := map[string]string{
		"AAA_INJECT":   "legit; echo PWNED_$(id) > /tmp/pwned #",
		"BBB_NEWLINE":  "ok\n          echo PWNED_NEWLINE",
		"CCC_BACKTICK": "`touch /tmp/PWNED_BACKTICK`",
	}

	var stepEnv strings.Builder
	writeMCPGatewayStepEnv(&stepEnv, nil, nil, gatewayEnv)

	assert.Contains(t, stepEnv.String(), `GH_AW_MCP_GATEWAY_ENV_0: "legit; echo PWNED_$(id) > /tmp/pwned #"`)
	assert.Contains(t, stepEnv.String(), `GH_AW_MCP_GATEWAY_ENV_1: "ok\n          echo PWNED_NEWLINE"`)
	assert.Contains(t, stepEnv.String(), "GH_AW_MCP_GATEWAY_ENV_2: \"`touch /tmp/PWNED_BACKTICK`\"")
	assert.Contains(t, stepEnv.String(), `GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: "[\"AAA_INJECT\",\"BBB_NEWLINE\",\"CCC_BACKTICK\"]"`)
	assert.NotContains(t, stepEnv.String(), "AAA_INJECT:")
	assert.NotContains(t, stepEnv.String(), "BBB_NEWLINE:")
	assert.NotContains(t, stepEnv.String(), "CCC_BACKTICK:")

	var runScript strings.Builder
	writeMCPGatewayExports(&runScript, writeMCPGatewayExportsOptions{
		engine:        NewCopilotEngine(),
		workflowData:  &WorkflowData{},
		gatewayConfig: &MCPGatewayRuntimeConfig{Env: gatewayEnv},
		port:          8080,
		domain:        "localhost",
		payloadDir:    "/tmp/payloads",
	})

	assert.NotContains(t, runScript.String(), "AAA_INJECT")
	assert.NotContains(t, runScript.String(), "BBB_NEWLINE")
	assert.NotContains(t, runScript.String(), "CCC_BACKTICK")

	var containerCommand strings.Builder
	appendMCPGatewayCustomAndHTTPEnvFlags(
		&containerCommand,
		&WorkflowData{},
		&MCPGatewayRuntimeConfig{Env: gatewayEnv},
		nil,
		false,
		nil,
		nil,
		NewCopilotEngine(),
	)
	assert.Equal(t, " "+mcpGatewayCustomEnvMarker, containerCommand.String())
}

func TestMCPGatewayCustomEnvOverridesGeneratedStepEnv(t *testing.T) {
	var yaml strings.Builder
	writeMCPGatewayStepEnv(
		&yaml,
		map[string]string{"API_TOKEN": "${{ secrets.DEFAULT_TOKEN }}"},
		map[string]string{"TARGET_REPO": "${{ inputs.target_repo }}"},
		map[string]string{
			"API_TOKEN":   "custom-token",
			"TARGET_REPO": "custom-repo",
		},
	)

	output := yaml.String()
	require.Zero(t, strings.Count(output, "API_TOKEN:"))
	require.Zero(t, strings.Count(output, "TARGET_REPO:"))
	assert.Contains(t, output, `GH_AW_MCP_GATEWAY_ENV_0: "custom-token"`)
	assert.Contains(t, output, `GH_AW_MCP_GATEWAY_ENV_1: "custom-repo"`)
	assert.NotContains(t, output, "API_TOKEN:")
	assert.NotContains(t, output, "TARGET_REPO:")
}

func TestMCPGatewayCustomEnvDoesNotSetBashEnvOnHost(t *testing.T) {
	var yaml strings.Builder
	writeMCPGatewayStepEnv(&yaml, nil, nil, map[string]string{
		"BASH_ENV": "$(touch /tmp/pwned)",
	})

	assert.Contains(t, yaml.String(), `GH_AW_MCP_GATEWAY_ENV_0: "$(touch /tmp/pwned)"`)
	assert.NotContains(t, yaml.String(), "BASH_ENV:")
}
