//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file covers the apple-container MCP gateway transport end to end, from
// frontmatter through to the generated lock file.
//
// The transport has four moving parts that must agree, and each is asserted
// separately so a break names the half that drifted:
//
//	1. gh-aw publishes awmg-mcpg on macOS loopback AppleContainerMCPGatewayHostPort.
//	2. gh-aw declares that port to AWF as appleContainer.mcpGatewayUpstreamPort.
//	3. AWF probes it, then publishes mcp-gateway.sock into the zero-NIC guest.
//	4. The agent's MCP client config addresses the guest relay at
//	   http://127.0.0.1:AppleContainerMCPGatewayGuestPort.
//
// Step 3 belongs to AWF (gh-aw-firewall#7768); everything else is asserted here.

// compileAppleContainerWorkflow compiles a minimal apple-container workflow and
// returns the generated lock file.
func compileAppleContainerWorkflow(t *testing.T, frontmatter string) string {
	t.Helper()

	compiler := NewCompiler()
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.md")

	require.NoError(t, os.WriteFile(inputFile, []byte(frontmatter), 0644))
	require.NoError(t, compiler.CompileWorkflow(inputFile), "apple-container workflow should compile")

	content, err := os.ReadFile(stringutil.MarkdownToLockFile(inputFile))
	require.NoError(t, err)
	return string(content)
}

const appleContainerWorkflowFrontmatter = `---
on: workflow_dispatch
engine: copilot
runs-on: [self-hosted, macOS, ARM64]
sandbox:
  agent:
    runtime: apple-container
    version: "` + string(constants.AWFAppleContainerMinVersion) + `"
tools:
  github:
    mode: remote
    toolsets: [repos]
---

# Apple Container MCP transport
`

func TestAppleContainerCompilesWithMCPGateway(t *testing.T) {
	t.Parallel()

	lockFile := compileAppleContainerWorkflow(t, appleContainerWorkflowFrontmatter)

	hostPort := strconv.Itoa(constants.AppleContainerMCPGatewayHostPort)
	guestPort := strconv.Itoa(constants.AppleContainerMCPGatewayGuestPort)

	// 1. The gateway container is published on macOS loopback, on the dedicated
	//    host port, and bound to 127.0.0.1 only. Binding 0.0.0.0 would put an
	//    API-key-authenticated gateway holding a Docker socket mount on every
	//    interface the self-hosted Mac sits on.
	//
	//    The container port is written as a shell expansion of MCP_GATEWAY_PORT,
	//    quoted the way the gateway command builder emits expandable variables.
	assert.Contains(t, lockFile, "-p 127.0.0.1:"+hostPort+`:'"${MCP_GATEWAY_PORT}"'`,
		"the gateway must be published on the dedicated loopback host port")
	assert.NotContains(t, lockFile, "-p 0.0.0.0:"+hostPort,
		"the gateway must never be published on all interfaces")

	// 2. The host port is declared to AWF, which is the only way the mcp-gateway
	//    capability is activated.
	assert.Contains(t, lockFile, `\"mcpGatewayUpstreamPort\":`+hostPort,
		"AWF must be told which loopback port to relay")

	// 4. The agent addresses the guest relay over loopback inside the VM.
	assert.Contains(t, lockFile, `export MCP_GATEWAY_DOMAIN="127.0.0.1"`,
		"the agent reaches the gateway through the guest relay on loopback")
	assert.Contains(t, lockFile, `export MCP_GATEWAY_PORT="`+guestPort+`"`,
		"the guest relay port is compiled into AWF's contract and is not negotiable")

	// Host-side probes must use the published port, not the guest port.
	assert.Contains(t, lockFile, `export MCP_GATEWAY_HOST_PORT="`+hostPort+`"`,
		"health checks run on the host and must probe the published port")
}

// TestAppleContainerGatewayCredentialTargetsPublishedPort is a security
// regression test.
//
// The Stop MCP Gateway step POSTs to /close with the gateway API key in an
// Authorization header. Before this layer the published port and the gateway's
// own port were always equal, so passing MCP_GATEWAY_PORT was harmless. Under
// apple-container they differ (published 9100, container 8080), and nothing gh-aw
// owns listens on 8080 on the host — so sending the credential to that port would
// hand it to whatever local process happens to own it on a long-lived
// self-hosted runner, silently, because the curl failure is tolerated.
func TestAppleContainerGatewayCredentialTargetsPublishedPort(t *testing.T) {
	t.Parallel()

	lockFile := compileAppleContainerWorkflow(t, appleContainerWorkflowFrontmatter)

	assert.Contains(t, lockFile, "MCP_GATEWAY_HOST_PORT: ${{ steps.start-mcp-gateway.outputs.gateway-host-port }}",
		"the stop step must receive the published port so it never sends the API key elsewhere")

	// The output the step above consumes has to actually be produced. The start
	// script is a repository file rather than generated output, so it is read
	// directly; a missing output would leave MCP_GATEWAY_HOST_PORT empty and fall
	// the stop script back to the gateway's own port.
	startScript, err := os.ReadFile(filepath.Join("..", "..", "actions", "setup", "sh", "start_mcp_gateway.sh"))
	require.NoError(t, err)
	assert.Contains(t, string(startScript), "gateway-host-port=${MCP_GATEWAY_HOST_PORT}",
		"the start step must publish the host port as an output")

	stopScript, err := os.ReadFile(filepath.Join("..", "..", "actions", "setup", "sh", "stop_mcp_gateway.sh"))
	require.NoError(t, err)
	assert.Contains(t, string(stopScript), `MCP_GATEWAY_HOST_PORT:-$MCP_GATEWAY_PORT`,
		"the stop script must prefer the published port")
	assert.NotContains(t, string(stopScript), "http://localhost:${MCP_GATEWAY_PORT}/close",
		"the API key must never be sent to the gateway's container port on the host")
}

// TestAppleContainerEmitsBothSelectorsWithUpstreamPort proves the AWF config
// carries the complete apple-container contract as one unit.
func TestAppleContainerEmitsBothSelectorsWithUpstreamPort(t *testing.T) {
	t.Parallel()

	lockFile := compileAppleContainerWorkflow(t, appleContainerWorkflowFrontmatter)

	assert.Contains(t, lockFile, `\"containerRuntime\":\"apple-container\"`)
	assert.Contains(t, lockFile, `\"previewEnabled\":true`)
	assert.Contains(t, lockFile, `\"mcpGatewayUpstreamPort\":`+strconv.Itoa(constants.AppleContainerMCPGatewayHostPort))
}

// TestAppleContainerLockFileHasNoGuestNetworkRoute is the security regression
// test for the whole layer: nothing generated for this runtime may give the
// guest a NIC, a Docker socket, or a Docker-network peer.
func TestAppleContainerLockFileHasNoGuestNetworkRoute(t *testing.T) {
	t.Parallel()

	lockFile := compileAppleContainerWorkflow(t, appleContainerWorkflowFrontmatter)

	for _, forbidden := range []string{
		"topologyAttach",
		"--topology-attach",
		"--enable-host-access",
		"--allow-host-ports",
		"--dind",
		"--no-network-isolation",
	} {
		assert.NotContains(t, lockFile, forbidden,
			"apple-container must not emit %q: it would give the zero-NIC guest a route out", forbidden)
	}

	// The gateway container legitimately mounts the host Docker socket — it is
	// host infrastructure that starts nested MCP servers — but the guest must
	// never receive one. Assert the agent-side AWF arguments carry no socket.
	assert.NotContains(t, lockFile, "--docker-host-path-prefix")
}

// TestAppleContainerLockFileEmitsProvisioningAndTeardown asserts the generated
// job actually carries the provisioning sequence and releases the runner
// afterwards. The step generators are unit-tested above; this proves they are
// wired into the compiled workflow at all.
func TestAppleContainerLockFileEmitsProvisioningAndTeardown(t *testing.T) {
	t.Parallel()

	lockFile := compileAppleContainerWorkflow(t, appleContainerWorkflowFrontmatter)

	for _, script := range []string{
		"apple_container_host_preflight.sh",
		"apple_container_setup_cli.sh",
		"apple_container_start_services.sh",
		"apple_container_teardown.sh",
	} {
		assert.Contains(t, lockFile, script, "generated job must run %s", script)
	}

	// Provisioning must precede the AWF installation, and teardown must follow the
	// agent. Index comparison is the only way to assert that from a flat file.
	preflightIdx := strings.Index(lockFile, "apple_container_host_preflight.sh")
	cliIdx := strings.Index(lockFile, "apple_container_setup_cli.sh")
	servicesIdx := strings.Index(lockFile, "apple_container_start_services.sh")
	awfIdx := strings.Index(lockFile, "install_awf_binary.sh")
	teardownIdx := strings.Index(lockFile, "apple_container_teardown.sh")

	assert.Less(t, preflightIdx, cliIdx, "an ineligible host must be rejected before anything is installed")
	assert.Less(t, cliIdx, servicesIdx, "the CLI must exist before its services are started")
	assert.Less(t, servicesIdx, awfIdx, "Apple Container must be ready before AWF is installed and run")
	assert.Less(t, awfIdx, teardownIdx, "teardown runs after the agent")
}

// TestAppleContainerLockFileKeepsDockerInfrastructure guards the half of the
// topology that does not move: Squid, the API proxy and the MCP gateway all stay
// on Docker, and only the agent crosses the hypervisor boundary.
func TestAppleContainerLockFileKeepsDockerInfrastructure(t *testing.T) {
	t.Parallel()

	lockFile := compileAppleContainerWorkflow(t, appleContainerWorkflowFrontmatter)

	assert.Contains(t, lockFile, "download_docker_images.sh",
		"infrastructure images are still pre-pulled into Docker")
	assert.Contains(t, lockFile, "--name awmg-mcpg",
		"the MCP gateway still runs as a host Docker container")
	assert.Contains(t, lockFile, "docker run -i --rm --network bridge",
		"the gateway keeps its bridge network; only the agent loses its NIC")
}

// against AWF's reserved set. AWF throws when the upstream port collides with a
// port one of its own sidecars binds, and the failure would only appear on a
// runner nobody can easily reproduce.
func TestAppleContainerUpstreamPortAvoidsAWFReservedPorts(t *testing.T) {
	t.Parallel()

	port := constants.AppleContainerMCPGatewayHostPort

	assert.NotEqual(t, 3128, port, "3128 is Squid")
	for reserved := 10000; reserved <= 10004; reserved++ {
		assert.NotEqual(t, reserved, port, "%d is an API proxy provider port", reserved)
	}
	assert.Greater(t, port, 1024, "the port must not require privileges to bind")
	assert.Less(t, port, 32768, "stay below the ephemeral range so the fixed port is not raced")
}

// TestAppleContainerRejectsNonDefaultMCPPort covers the one part of the contract
// an author can break: the guest-facing port is fixed by AWF.
func TestAppleContainerRejectsNonDefaultMCPPort(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	workflowData.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{Port: 9999}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err)
	require.ErrorContains(t, err, "sandbox.mcp.port")
	require.ErrorContains(t, err, strconv.Itoa(constants.AppleContainerMCPGatewayGuestPort))

	// The default is accepted, and so is an unset port.
	workflowData.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{Port: constants.AppleContainerMCPGatewayGuestPort}
	require.NoError(t, validateSandboxConfig(workflowData))

	workflowData.SandboxConfig.MCP = nil
	require.NoError(t, validateSandboxConfig(workflowData))
}

// TestAppleContainerUpstreamPortOmittedWithoutGateway keeps the capability
// request honest: AWF probes the upstream port before publishing the socket, so
// declaring a port no gateway listens on would fail the run.
func TestAppleContainerUpstreamPortOmittedWithoutGateway(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	assert.True(t, hasMCPGatewayForAppleContainer(workflowData))

	workflowData.Tools = map[string]any{}
	assert.False(t, hasMCPGatewayForAppleContainer(workflowData),
		"no MCP servers means no gateway to relay")

	// Other runtimes never request the capability at all.
	other := newAppleContainerWorkflow()
	other.SandboxConfig.Agent.Runtime = AgentRuntimeDocker
	assert.False(t, hasMCPGatewayForAppleContainer(other))
}

// TestAppleContainerAWFConfigShape parses the generated AWF config rather than
// substring-matching it, so a field that lands in the wrong section is caught.
func TestAppleContainerAWFConfigShape(t *testing.T) {
	t.Parallel()

	workflowData := newAppleContainerWorkflow()
	configJSON, err := BuildAWFConfigJSON(AWFCommandConfig{WorkflowData: workflowData})
	require.NoError(t, err)

	var parsed struct {
		Container struct {
			ContainerRuntime string `json:"containerRuntime"`
		} `json:"container"`
		AppleContainer struct {
			PreviewEnabled         bool `json:"previewEnabled"`
			MCPGatewayUpstreamPort int  `json:"mcpGatewayUpstreamPort"`
		} `json:"appleContainer"`
		Network struct {
			TopologyAttach []string `json:"topologyAttach"`
		} `json:"network"`
	}
	require.NoError(t, json.Unmarshal([]byte(configJSON), &parsed))

	assert.Equal(t, "apple-container", parsed.Container.ContainerRuntime)
	assert.True(t, parsed.AppleContainer.PreviewEnabled)
	assert.Equal(t, constants.AppleContainerMCPGatewayHostPort, parsed.AppleContainer.MCPGatewayUpstreamPort)
	assert.Empty(t, parsed.Network.TopologyAttach,
		"AWF rejects topologyAttach for this runtime; the capability socket replaces it")
}

// TestOtherRuntimesKeepTopologyGatewayAccess is the collateral-damage guard: the
// Linux runtimes must still reach the gateway exactly as before.
func TestOtherRuntimesKeepTopologyGatewayAccess(t *testing.T) {
	t.Parallel()

	for _, runtime := range []AgentRuntime{AgentRuntimeDocker, AgentRuntimeGVisor, AgentRuntimeCloudHypervisor} {
		t.Run(string(runtime), func(t *testing.T) {
			t.Parallel()
			workflowData := newAppleContainerWorkflow()
			workflowData.SandboxConfig.Agent.Runtime = runtime
			workflowData.RunsOn = "runs-on: ubuntu-latest"

			assert.Contains(t, buildAWFTopologyAttachList(workflowData), "awmg-mcpg",
				"every other runtime still bridges the gateway over AWF's Docker network")

			configJSON, err := BuildAWFConfigJSON(AWFCommandConfig{WorkflowData: workflowData})
			require.NoError(t, err)
			assert.NotContains(t, configJSON, "mcpGatewayUpstreamPort",
				"the Apple Container capability must not leak into other runtimes")
			assert.NotContains(t, configJSON, "appleContainer")
		})
	}
}
