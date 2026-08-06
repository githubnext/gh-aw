package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateProcessOutputLogging(t *testing.T) {
	var yaml strings.Builder
	compiler := &Compiler{}

	compiler.generateProcessOutputLogging(&yaml, &WorkflowData{})

	output := yaml.String()
	assert.Contains(t, output, "- name: Log process output")
	assert.Contains(t, output, "if: always() && steps.redact-secrets-in-logs.outcome == 'success'")
	assert.Contains(t, output, "MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}")
	assert.Contains(t, output, "action_log.cjs")
	assert.Contains(t, output, "gateway-launch-stderr.log")
	assert.Contains(t, output, "/tmp/gh-aw/mcp-logs")
	// gateway-output.json is the full resolved MCP server config (includes secrets in
	// Authorization headers); it must never be printed to the Actions log, even redacted.
	assert.NotContains(t, output, "gateway-output.json")
}
