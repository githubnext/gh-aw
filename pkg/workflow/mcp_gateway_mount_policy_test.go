//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildMCPGatewayAllowedMountRoots verifies that the gateway's
// MCP_GATEWAY_ALLOWED_MOUNT_ROOTS value grants read-write access to the host
// paths our built-in MCP servers (safe-outputs, agentic-workflows) mount, so
// gh-aw-mcpg's trusted host-path mount policy does not reject them.
func TestBuildMCPGatewayAllowedMountRoots(t *testing.T) {
	t.Run("nil gateway config still includes builtin roots", func(t *testing.T) {
		roots := buildMCPGatewayAllowedMountRoots(nil)
		assert.Contains(t, roots, "${GITHUB_WORKSPACE}:rw")
		assert.Contains(t, roots, "${RUNNER_TEMP}/gh-aw:rw")
		assert.Contains(t, roots, "/tmp:rw")
		assert.Contains(t, roots, "/usr/bin/gh:ro")
	})

	t.Run("default gateway mounts are merged in", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{}
		ensureDefaultMCPGatewayConfig(&WorkflowData{SandboxConfig: &SandboxConfig{MCP: gatewayConfig}})
		roots := buildMCPGatewayAllowedMountRoots(gatewayConfig)

		assert.Contains(t, roots, "${GITHUB_WORKSPACE}:rw")
		assert.Contains(t, roots, "${RUNNER_TEMP}/gh-aw:rw")
		assert.Contains(t, roots, "/tmp:rw")
		assert.Contains(t, roots, "/usr/bin/gh:ro")
		assert.Contains(t, roots, "/opt:ro")
	})

	t.Run("custom rw mount widens an existing ro root", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/usr/bin/gh:/usr/bin/gh:rw"},
		}
		roots := buildMCPGatewayAllowedMountRoots(gatewayConfig)
		assert.Contains(t, roots, "/usr/bin/gh:rw")
		assert.NotContains(t, roots, "/usr/bin/gh:ro")
	})

	t.Run("existing rw root is not downgraded by a later ro mount", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/data:/data:rw", "/data:/data:ro"},
		}
		roots := buildMCPGatewayAllowedMountRoots(gatewayConfig)
		assert.Contains(t, roots, "/data:rw")
		assert.NotContains(t, roots, "/data:ro")
	})

	t.Run("output is deterministic", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/opt:/opt:ro", "/data:/data:rw"},
		}
		first := buildMCPGatewayAllowedMountRoots(gatewayConfig)
		second := buildMCPGatewayAllowedMountRoots(gatewayConfig)
		assert.Equal(t, first, second)
	})
}

// TestMCPGatewayContainerCommandIncludesAllowedMountRootsEnvFlag verifies the
// gateway container is launched with the MCP_GATEWAY_ALLOWED_MOUNT_ROOTS
// environment variable forwarded so the value exported at runtime reaches it.
func TestMCPGatewayContainerCommandIncludesAllowedMountRootsEnvFlag(t *testing.T) {
	var containerCmd strings.Builder
	appendMCPGatewayBaseEnvFlags(&containerCmd, "")
	assert.Contains(t, containerCmd.String(), " -e MCP_GATEWAY_ALLOWED_MOUNT_ROOTS")
}

// TestWriteMCPGatewayExportsIncludesAllowedMountRoots verifies the run script
// exports MCP_GATEWAY_ALLOWED_MOUNT_ROOTS before starting the gateway.
func TestWriteMCPGatewayExportsIncludesAllowedMountRoots(t *testing.T) {
	var runScript strings.Builder
	writeMCPGatewayExports(&runScript, writeMCPGatewayExportsOptions{
		engine:        NewCopilotEngine(),
		workflowData:  &WorkflowData{},
		gatewayConfig: &MCPGatewayRuntimeConfig{},
		port:          8080,
		domain:        "localhost",
		payloadDir:    "/tmp/payloads",
	})

	assert.Contains(t, runScript.String(), `export MCP_GATEWAY_ALLOWED_MOUNT_ROOTS="${GITHUB_WORKSPACE}:rw,${RUNNER_TEMP}/gh-aw:rw,/tmp:rw,/usr/bin/gh:ro"`)
}
