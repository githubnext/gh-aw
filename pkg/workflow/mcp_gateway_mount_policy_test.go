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
	t.Run("nil tools and gateway config still includes builtin roots", func(t *testing.T) {
		roots := buildMCPGatewayAllowedMountRoots(nil, nil)
		assert.Contains(t, roots, "${GITHUB_WORKSPACE}:rw")
		// Only the safe-outputs runtime subdirectory needs read-write access; the
		// rest of the gh-aw runtime tree stays read-only.
		assert.Contains(t, roots, "${RUNNER_TEMP}/gh-aw:ro")
		assert.Contains(t, roots, "${RUNNER_TEMP}/gh-aw/safeoutputs:rw")
		assert.Contains(t, roots, "/tmp:rw")
		assert.Contains(t, roots, "/usr/bin/gh:ro")
	})

	t.Run("default gateway mounts are merged in", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{}
		ensureDefaultMCPGatewayConfig(&WorkflowData{SandboxConfig: &SandboxConfig{MCP: gatewayConfig}})
		roots := buildMCPGatewayAllowedMountRoots(nil, gatewayConfig)

		assert.Contains(t, roots, "${GITHUB_WORKSPACE}:rw")
		assert.Contains(t, roots, "${RUNNER_TEMP}/gh-aw:ro")
		assert.Contains(t, roots, "${RUNNER_TEMP}/gh-aw/safeoutputs:rw")
		assert.Contains(t, roots, "/tmp:rw")
		assert.Contains(t, roots, "/usr/bin/gh:ro")
		assert.Contains(t, roots, "/opt:ro")
	})

	t.Run("custom rw mount widens an existing ro root", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/usr/bin/gh:/usr/bin/gh:rw"},
		}
		roots := buildMCPGatewayAllowedMountRoots(nil, gatewayConfig)
		assert.Contains(t, roots, "/usr/bin/gh:rw")
		assert.NotContains(t, roots, "/usr/bin/gh:ro")
	})

	t.Run("existing rw root is not downgraded by a later ro mount", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/data:/data:rw", "/data:/data:ro"},
		}
		roots := buildMCPGatewayAllowedMountRoots(nil, gatewayConfig)
		assert.Contains(t, roots, "/data:rw")
		assert.NotContains(t, roots, "/data:ro")
	})

	t.Run("mode-less gateway mount defaults to rw per Docker semantics", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/data:/data"},
		}
		roots := buildMCPGatewayAllowedMountRoots(nil, gatewayConfig)
		assert.Contains(t, roots, "/data:rw")
		assert.NotContains(t, roots, "/data:ro")
	})

	t.Run("output is deterministic", func(t *testing.T) {
		gatewayConfig := &MCPGatewayRuntimeConfig{
			Mounts: []string{"/opt:/opt:ro", "/data:/data:rw"},
		}
		first := buildMCPGatewayAllowedMountRoots(nil, gatewayConfig)
		second := buildMCPGatewayAllowedMountRoots(nil, gatewayConfig)
		assert.Equal(t, first, second)
	})

	t.Run("mcp-servers.<name>.mounts are included in the allowlist", func(t *testing.T) {
		tools := map[string]any{
			"my-custom-server": map[string]any{
				"container": "ghcr.io/example/my-tool:latest",
				"mounts":    []any{"/srv/data:/data:ro", "/srv/output:/output:rw"},
			},
		}
		roots := buildMCPGatewayAllowedMountRoots(tools, nil)
		assert.Contains(t, roots, "/srv/data:ro")
		assert.Contains(t, roots, "/srv/output:rw")
	})

	t.Run("tools.github.mounts are included in the allowlist", func(t *testing.T) {
		tools := map[string]any{
			"github": map[string]any{
				"mounts": []any{"/etc/gh-config:/etc/gh-config:ro"},
			},
		}
		roots := buildMCPGatewayAllowedMountRoots(tools, nil)
		assert.Contains(t, roots, "/etc/gh-config:ro")
	})

	t.Run("backslash-escaped mount vars normalize to the same root as the builtin entry", func(t *testing.T) {
		// Imported partials (e.g. shared/mcp/serena.md) escape "${GITHUB_WORKSPACE}"
		// as "\${GITHUB_WORKSPACE}" to survive GitHub Actions expression interpolation
		// during import merging; the gateway unescapes this before substitution.
		tools := map[string]any{
			"serena": map[string]any{
				"container": "ghcr.io/example/serena:latest",
				"mounts":    []any{`\${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:rw`},
			},
		}
		roots := buildMCPGatewayAllowedMountRoots(tools, nil)
		assert.Contains(t, roots, "${GITHUB_WORKSPACE}:rw")
		assert.NotContains(t, roots, `\${GITHUB_WORKSPACE}`)
	})

	t.Run("volume mounts embedded in args are included in the allowlist", func(t *testing.T) {
		tools := map[string]any{
			"github": map[string]any{
				"args": []any{"-v", "/opt/gh-data:/data:ro"},
			},
			"playwright": map[string]any{
				"args": []any{"--volume=/opt/pw-data:/data:rw"},
			},
			"my-custom-server": map[string]any{
				"container":      "ghcr.io/example/my-tool:latest",
				"entrypointArgs": []any{"--volume", "/opt/entry-data:/data"},
			},
		}
		roots := buildMCPGatewayAllowedMountRoots(tools, nil)
		assert.Contains(t, roots, "/opt/gh-data:ro")
		assert.Contains(t, roots, "/opt/pw-data:rw")
		// Mode-less mounts supplied via args are writable per Docker semantics.
		assert.Contains(t, roots, "/opt/entry-data:rw")
		assert.NotContains(t, roots, "/opt/entry-data:ro")
	})
}

// TestMCPGatewayContainerCommandIncludesAllowedMountRootsEnvFlag verifies the
// gateway container is launched with the MCP_GATEWAY_ALLOWED_MOUNT_ROOTS
// environment variable forwarded so the value exported at runtime reaches it.
func TestMCPGatewayContainerCommandIncludesAllowedMountRootsEnvFlag(t *testing.T) {
	var containerCmd strings.Builder
	appendMCPGatewayBaseEnvFlags(&containerCmd, "")
	assert.Contains(t, containerCmd.String(), " -e MCP_GATEWAY_ALLOWED_MOUNT_ROOTS")
	assert.Contains(t, containerCmd.String(), " -e RUNNER_TOOL_CACHE")
	for _, name := range optionalPRHeadEnvVars {
		assert.Contains(t, containerCmd.String(), " -e "+name)
	}
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

	assert.Contains(t, runScript.String(), `export MCP_GATEWAY_ALLOWED_MOUNT_ROOTS="${GITHUB_WORKSPACE}:rw,${RUNNER_TEMP}/gh-aw:ro,${RUNNER_TEMP}/gh-aw/safeoutputs:rw,/tmp:rw,/usr/bin/gh:ro"`)
	for _, name := range optionalPRHeadEnvVars {
		assert.Contains(t, runScript.String(), `export `+name+`="${`+name+`:-}"`)
	}
}

// TestWriteMCPGatewayExportsIncludesCustomServerMounts verifies that a custom
// containerized MCP server's declared mounts widen the exported allowlist.
func TestWriteMCPGatewayExportsIncludesCustomServerMounts(t *testing.T) {
	var runScript strings.Builder
	writeMCPGatewayExports(&runScript, writeMCPGatewayExportsOptions{
		engine:        NewCopilotEngine(),
		workflowData:  &WorkflowData{},
		gatewayConfig: &MCPGatewayRuntimeConfig{},
		tools: map[string]any{
			"my-custom-server": map[string]any{
				"container": "ghcr.io/example/my-tool:latest",
				"mounts":    []any{"/srv/data:/data:rw"},
			},
		},
		port:       8080,
		domain:     "localhost",
		payloadDir: "/tmp/payloads",
	})

	assert.Contains(t, runScript.String(), "/srv/data:rw")
}
