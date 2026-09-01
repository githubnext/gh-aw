//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests in this file are executable guardrails for the "build-time invariant"
// panic sites documented in this package. Each test exercises the real code path
// that precedes a panic() so that a refactor which makes the panic reachable fails
// the test suite instead of only surfacing at runtime.

// TestEmbeddedModelAliasesAreLoadable guards the panics in getBuiltinOnlyAliasMap
// and BuiltinModelAliases, which fire only when the embedded model_aliases.json
// cannot be unmarshaled.
func TestEmbeddedModelAliasesAreLoadable(t *testing.T) {
	aliases, err := loadBuiltinModelAliases()
	require.NoError(t, err, "embedded model_aliases.json must unmarshal at build time")
	assert.NotEmpty(t, aliases)

	assert.NotEmpty(t, getBuiltinOnlyAliasMap())
	assert.NotEmpty(t, BuiltinModelAliases())
}

// TestEmbeddedToolsetPermissionsAreLoadable guards the panic in
// getToolsetPermissionsMap, which fires only when the embedded GitHub toolsets
// JSON cannot be unmarshaled.
func TestEmbeddedToolsetPermissionsAreLoadable(t *testing.T) {
	assert.NotEmpty(t, getToolsetPermissionsMap())
}

// TestBuiltinEnginesRegisterWithoutError guards the panic in NewEngineRegistry.
// Register only rejects a negative dedicatedLLMGatewayPort, so re-registering every
// built-in engine asserts exactly the failure mode the panic guards against.
func TestBuiltinEnginesRegisterWithoutError(t *testing.T) {
	registry := NewEngineRegistry()

	for _, id := range registry.GetSupportedEngines() {
		engine, err := registry.GetEngine(id)
		require.NoError(t, err)
		assert.NoError(t, NewEngineRegistry().Register(engine), "built-in engine %q must register without error", id)
	}
}

// TestMCPGatewayCustomEnvNamesMarshal guards the panic in
// writeMCPGatewayStepEnvWithCustomGatewayEnvNames, which fires only if the
// []string of env var names fails to marshal.
func TestMCPGatewayCustomEnvNamesMarshal(t *testing.T) {
	var stepEnv strings.Builder
	gatewayEnv := map[string]string{"CUSTOM_ONE": "value-one", "CUSTOM_TWO": "value-two"}
	writeMCPGatewayStepEnvForTest(&stepEnv, nil, nil, gatewayEnv)

	assert.Contains(t, stepEnv.String(), mcpGatewayCustomEnvNamesVar)
	assert.Contains(t, stepEnv.String(), `[\"CUSTOM_ONE\",\"CUSTOM_TWO\"]`)
}

// TestFileRenderConfigMarshals guards the panic in generateSafeOutputsSetup, which
// fires only if the fixed fileRenderConfig struct fails to marshal.
func TestFileRenderConfigMarshals(t *testing.T) {
	out, err := json.Marshal(fileRenderConfig{
		Files: []fileRenderItem{{
			Path:       "safeoutputs/config.json",
			ContentEnv: "GH_AW_SAFE_OUTPUTS_CONFIG",
		}},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"files":[{"path":"safeoutputs/config.json","content_env":"GH_AW_SAFE_OUTPUTS_CONFIG"}]}`, string(out))
}
