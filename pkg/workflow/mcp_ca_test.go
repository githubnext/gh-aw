//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPCACertFileCompilation(t *testing.T) {
	for _, engine := range []string{"copilot", "claude", "codex", "gemini"} {
		t.Run(engine, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "private-ca.md")
			require.NoError(t, os.WriteFile(file, []byte(`---
on: workflow_dispatch
strict: false
engine: `+engine+`
permissions:
  contents: read
mcp-servers:
  private:
    type: http
    url: https://private.example/mcp
    ca-cert-file: "${{ runner.temp }}/private-ca.pem"
    headers:
      Authorization: "Bearer ${{ secrets.PRIVATE_TOKEN }}"
  public:
    url: https://public.example/mcp
---
Use the MCP servers.
`), 0600))
			compiler := NewCompiler()
			compiler.SetSkipValidation(true)
			data, err := compiler.ParseWorkflowFile(file)
			require.NoError(t, err)
			output, _, _, err := compiler.generateYAML(data, file)
			require.NoError(t, err)
			require.Contains(t, output, `"caCertFile": "\${RUNNER_TEMP}/private-ca.pem"`)
			require.Equal(t, 1, strings.Count(output, `"caCertFile"`))
			require.NotRegexp(t, `(NODE_EXTRA_CA_CERTS|SSL_CERT_FILE).*private-ca\.pem`, output)
			require.NotContains(t, output, "update-ca-certificates")
		})
	}
}

func TestMCPCACertFileValidation(t *testing.T) {
	for _, value := range []any{"", false, "../ca.pem", "$(id)", "${{ secrets.CA }}"} {
		config := map[string]any{"type": "http", "url": "https://example.com", "ca-cert-file": value}
		require.ErrorContains(t, validateMCPRequirements("private", config, config), "ca-cert-file")
	}
	config := map[string]any{"container": "example/server", "ca-cert-file": "ca.pem"}
	require.ErrorContains(t, validateMCPRequirements("private", config, config), "only supported for HTTP")
	config = map[string]any{"url": "http://example.com", "ca-cert-file": "ca.pem"}
	require.ErrorContains(t, validateMCPRequirements("private", config, config), "requires an HTTPS URL")
	config["url"] = "https://example.com"
	require.NoError(t, validateMCPRequirements("private", config, config))
	require.Equal(t, "ca.pem", buildSchemaMCPConfig(config)["ca-cert-file"])
	_, err := getRawMCPConfig(config)
	require.NoError(t, err)
}

func TestMCPCACertFileTypedRoundTrip(t *testing.T) {
	raw := map[string]any{
		"type": "http", "url": "https://private.example", "ca-cert-file": ".github/ca.pem",
	}
	tools, err := ParseToolsConfig(map[string]any{"private": raw})
	require.NoError(t, err)
	config := tools.Custom["private"]
	require.Equal(t, ".github/ca.pem", config.CACertFile)
	require.NotContains(t, config.CustomFields, "ca-cert-file")
	require.Equal(t, raw, mcpServerConfigToMap(config))

	config.CACertFile = ".github/updated-ca.pem"
	serialized := mcpServerConfigToMap(config)
	require.Equal(t, ".github/updated-ca.pem", serialized["ca-cert-file"])
	tools, err = ParseToolsConfig(map[string]any{"private": serialized})
	require.NoError(t, err)
	require.Equal(t, ".github/updated-ca.pem", tools.Custom["private"].CACertFile)

	config.CACertFile = ""
	serialized = mcpServerConfigToMap(config)
	require.NotContains(t, serialized, "ca-cert-file")
	tools, err = ParseToolsConfig(map[string]any{"private": serialized})
	require.NoError(t, err)
	require.Empty(t, tools.Custom["private"].CACertFile)
}

func TestMCPCACertFileOldGateway(t *testing.T) {
	data := &WorkflowData{SandboxConfig: &SandboxConfig{MCP: &MCPGatewayRuntimeConfig{Version: "v0.4.34"}}}
	tools := map[string]any{"private": map[string]any{"url": "https://private.example", "ca-cert-file": "ca.pem"}}
	var output strings.Builder
	err := generateMCPGatewaySetup(&output, tools, []string{"private"}, NewCopilotEngine(), data, false, nil)
	require.ErrorContains(t, err, "requires MCP Gateway v0.4.35")
}
