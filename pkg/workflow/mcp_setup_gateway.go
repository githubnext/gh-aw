package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

func generateMCPGatewaySetup(yaml *strings.Builder, tools map[string]any, mcpTools []string, engine CodingAgentEngine, workflowData *WorkflowData, hasAgenticWorkflows bool, safeOutputsInputEnvVars map[string]string) error {
	yaml.WriteString("      - name: Start MCP Gateway\n")
	yaml.WriteString("        id: start-mcp-gateway\n")
	mcpEnvVars := collectMCPEnvironmentVariables(tools, mcpTools, workflowData, hasAgenticWorkflows)
	writeMCPGatewayStepEnv(yaml, mcpEnvVars, safeOutputsInputEnvVars)
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -eo pipefail\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/mcp-config\"\n")
	if slices.Contains(mcpTools, "playwright") {
		yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/playwright\n")
		yaml.WriteString("          chmod 777 /tmp/gh-aw/mcp-logs/playwright\n")
	}
	ensureDefaultMCPGatewayConfig(workflowData)
	gatewayConfig := workflowData.SandboxConfig.MCP
	port, domain, payloadDir, payloadPathPrefix, payloadSizeThreshold := resolveMCPGatewayValues(workflowData, gatewayConfig)
	githubToolRaw, hasGitHub := tools["github"]
	githubTool, _ := githubToolRaw.(map[string]any)
	writeMCPGatewayExports(yaml, writeMCPGatewayExportsOptions{
		engine:               engine,
		workflowData:         workflowData,
		gatewayConfig:        gatewayConfig,
		hasGitHub:            hasGitHub,
		githubTool:           githubTool,
		port:                 port,
		domain:               domain,
		payloadDir:           payloadDir,
		payloadPathPrefix:    payloadPathPrefix,
		payloadSizeThreshold: payloadSizeThreshold,
	})
	containerCmd := buildMCPGatewayContainerCommand(buildMCPGatewayContainerCommandOptions{
		engine:                  engine,
		workflowData:            workflowData,
		gatewayConfig:           gatewayConfig,
		mcpEnvVars:              mcpEnvVars,
		payloadDir:              payloadDir,
		payloadPathPrefix:       payloadPathPrefix,
		hasGitHub:               hasGitHub,
		githubTool:              githubTool,
		tools:                   tools,
		safeOutputsInputEnvVars: safeOutputsInputEnvVars,
	})
	yaml.WriteString("          MCP_GATEWAY_UID=$(id -u 2>/dev/null || echo '0')\n")
	yaml.WriteString("          MCP_GATEWAY_GID=$(id -g 2>/dev/null || echo '0')\n")
	// Resolve Docker socket path and GID using the dedicated shell script.
	// The script handles override variables (GH_AW_DOCKER_SOCK_PATH, GH_AW_DOCKER_SOCK_GID),
	// DOCKER_HOST parsing, stat -Lc symlink following, and numeric validation.
	// See actions/setup/sh/resolve_docker_socket_gid.sh for implementation details.
	yaml.WriteString("          source \"${RUNNER_TEMP}/gh-aw/actions/resolve_docker_socket_gid.sh\"\n")
	cmdWithExpandableVars := buildDockerCommandWithExpandableVars(containerCmd)
	yaml.WriteString("          export MCP_GATEWAY_DOCKER_COMMAND=" + cmdWithExpandableVars + "\n")
	yaml.WriteString("          \n")
	return engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}

func writeMCPGatewayStepEnv(yaml *strings.Builder, mcpEnvVars map[string]string, safeOutputsInputEnvVars map[string]string) {
	if len(mcpEnvVars) == 0 && len(safeOutputsInputEnvVars) == 0 {
		return
	}
	yaml.WriteString("        env:\n")
	// Write MCP env vars first (sorted)
	envVarNames := sliceutil.MapKeys(mcpEnvVars)
	sort.Strings(envVarNames)
	for _, envVarName := range envVarNames {
		fmt.Fprintf(yaml, "          %s: %s\n", envVarName, mcpEnvVars[envVarName])
	}
	// Write safe-outputs input env vars (sorted); these must also be present in the
	// runner step environment so the docker -e flag can forward them to the container.
	inputVarNames := sliceutil.SortedKeys(safeOutputsInputEnvVars)
	for _, envVarName := range inputVarNames {
		fmt.Fprintf(yaml, "          %s: %s\n", envVarName, safeOutputsInputEnvVars[envVarName])
	}
}

func resolveMCPGatewayValues(workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig) (int, string, string, string, int) {
	port := gatewayConfig.Port
	if port == 0 {
		port = int(DefaultMCPGatewayPort)
	}
	domain := gatewayConfig.Domain
	if domain == "" {
		if workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled {
			domain = "localhost"
		} else if isDockerSbxRuntime(workflowData) {
			// docker-sbx microVM reaches host-published services via host.docker.internal
			// (the Docker bridge gateway). Use this as the MCP gateway domain so that the
			// CLI wrapper scripts generated inside the microVM point to the correct host.
			domain = "host.docker.internal"
		} else if isAWFNetworkIsolationEnabled(workflowData) {
			domain = "awmg-mcpg"
		} else {
			domain = "host.docker.internal"
		}
	}
	payloadDir := gatewayConfig.PayloadDir
	if payloadDir == "" {
		payloadDir = constants.DefaultMCPGatewayPayloadDir
	}
	payloadSizeThreshold := gatewayConfig.PayloadSizeThreshold
	if payloadSizeThreshold == 0 {
		payloadSizeThreshold = constants.DefaultMCPGatewayPayloadSizeThreshold
	}
	return port, domain, payloadDir, gatewayConfig.PayloadPathPrefix, payloadSizeThreshold
}

// writeMCPGatewayExportsOptions holds configuration for writeMCPGatewayExports.
type writeMCPGatewayExportsOptions struct {
	engine               CodingAgentEngine
	workflowData         *WorkflowData
	gatewayConfig        *MCPGatewayRuntimeConfig
	hasGitHub            bool
	githubTool           map[string]any
	port                 int
	domain               string
	payloadDir           string
	payloadPathPrefix    string
	payloadSizeThreshold int
}

func writeMCPGatewayExports(yaml *strings.Builder, opts writeMCPGatewayExportsOptions) {
	engine := opts.engine
	workflowData := opts.workflowData
	gatewayConfig := opts.gatewayConfig
	hasGitHub := opts.hasGitHub
	githubTool := opts.githubTool
	port := opts.port
	domain := opts.domain
	payloadDir := opts.payloadDir
	payloadPathPrefix := opts.payloadPathPrefix
	payloadSizeThreshold := opts.payloadSizeThreshold
	yaml.WriteString("          \n")
	yaml.WriteString("          # Export gateway environment variables for MCP config and gateway script\n")
	yaml.WriteString("          export MCP_GATEWAY_PORT=\"" + strconv.Itoa(port) + "\"\n")
	yaml.WriteString("          export MCP_GATEWAY_DOMAIN=\"" + domain + "\"\n")
	// MCP_GATEWAY_HOST_DOMAIN is the domain used by host-side clients (e.g. Gemini CLI).
	// When MCP_GATEWAY_DOMAIN is host.docker.internal (only reachable from containers),
	// or when network isolation is active (gateway on bridge; host reaches it via the
	// published 127.0.0.1 port), use localhost instead; otherwise inherit the domain.
	// Exception: for docker-sbx, the CLI wrappers run INSIDE the microVM, so they must
	// also use host.docker.internal (not localhost) to reach the published gateway port.
	// Exception: for Gemini under network isolation, use the topology hostname (awmg-mcpg)
	// instead of localhost. The Gemini CLI honors HTTP_PROXY but ignores NO_PROXY, so
	// localhost:8080 would be tunneled through the squid egress proxy and denied. The
	// awmg-mcpg topology hostname is already in the firewall allowlist.
	hostDomain := domain
	if isDockerSbxRuntime(workflowData) {
		hostDomain = "host.docker.internal"
	} else if engine.GetID() == "gemini" && isAWFNetworkIsolationEnabled(workflowData) {
		// domain is "awmg-mcpg" when network isolation is active; preserve it.
		hostDomain = domain
	} else if domain == "host.docker.internal" || isAWFNetworkIsolationEnabled(workflowData) {
		hostDomain = "localhost"
	}
	yaml.WriteString("          export MCP_GATEWAY_HOST_DOMAIN=\"" + hostDomain + "\"\n")
	if gatewayConfig.APIKey == "" {
		yaml.WriteString("          MCP_GATEWAY_API_KEY=$(openssl rand -base64 45 | tr -d '/+=')\n")
		yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")
		yaml.WriteString("          export MCP_GATEWAY_API_KEY\n")
	} else {
		yaml.WriteString("          export MCP_GATEWAY_API_KEY=\"" + gatewayConfig.APIKey + "\"\n")
		yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")
	}
	yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_DIR=\"" + payloadDir + "\"\n")
	yaml.WriteString("          mkdir -p \"${MCP_GATEWAY_PAYLOAD_DIR}\"\n")
	if payloadPathPrefix != "" {
		yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_PATH_PREFIX=\"" + payloadPathPrefix + "\"\n")
	}
	yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_SIZE_THRESHOLD=\"" + strconv.Itoa(payloadSizeThreshold) + "\"\n")
	yaml.WriteString("          export DEBUG=\"*\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          export GH_AW_ENGINE=\"" + engine.GetID() + "\"\n")
	if cliServers := getMCPCLIExcludeFromAgentConfig(workflowData); len(cliServers) > 0 {
		cliServersJSON, err := json.Marshal(cliServers)
		if err == nil {
			escapedCLIServersJSON := shellEscapeArg(string(cliServersJSON))
			yaml.WriteString("          export GH_AW_MCP_CLI_SERVERS=" + escapedCLIServersJSON + "\n")
		}
	}
	if hasGitHub && getGitHubType(githubTool) == GitHubMCPModeRemote && engine.GetID() == "copilot" {
		yaml.WriteString("          export GITHUB_PERSONAL_ACCESS_TOKEN=\"$GITHUB_MCP_SERVER_TOKEN\"\n")
	}
	if len(gatewayConfig.Env) > 0 {
		envVarNames := sliceutil.MapKeys(gatewayConfig.Env)
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			fmt.Fprintf(yaml, "          export %s=%s\n", envVarName, gatewayConfig.Env[envVarName])
		}
	}
}

// buildMCPGatewayContainerCommandOptions holds configuration for buildMCPGatewayContainerCommand.
type buildMCPGatewayContainerCommandOptions struct {
	engine                  CodingAgentEngine
	workflowData            *WorkflowData
	gatewayConfig           *MCPGatewayRuntimeConfig
	mcpEnvVars              map[string]string
	payloadDir              string
	payloadPathPrefix       string
	hasGitHub               bool
	githubTool              map[string]any
	tools                   map[string]any
	safeOutputsInputEnvVars map[string]string
}

func buildMCPGatewayContainerCommand(opts buildMCPGatewayContainerCommandOptions) string {
	engine := opts.engine
	workflowData := opts.workflowData
	gatewayConfig := opts.gatewayConfig
	mcpEnvVars := opts.mcpEnvVars
	payloadDir := opts.payloadDir
	payloadPathPrefix := opts.payloadPathPrefix
	hasGitHub := opts.hasGitHub
	githubTool := opts.githubTool
	tools := opts.tools
	safeOutputsInputEnvVars := opts.safeOutputsInputEnvVars
	containerImage := gatewayConfig.Container
	if gatewayConfig.Version != "" {
		containerImage += ":" + gatewayConfig.Version
	} else {
		containerImage += ":" + string(constants.DefaultMCPGatewayVersion)
	}
	// Apply container_pins mapping from aw.json so the runtime docker run command
	// targets the redirected registry (e.g. an internal mirror) rather than the
	// default public registry.
	containerImage = applyContainerPinMappingFromData(containerImage, workflowData)
	var containerCmd strings.Builder
	// Pre-size the builder to avoid reallocations. The base flags from
	// appendMCPGatewayBaseEnvFlags alone write ~2KB of -e flags; allocating
	// 2048 bytes upfront covers the common case without overcommitting.
	containerCmd.Grow(2048)
	containerCmd.WriteString("docker run -i --rm")
	if isAWFNetworkIsolationEnabled(workflowData) {
		containerCmd.WriteString(" --network bridge")
		if isDockerSbxRuntime(workflowData) {
			// docker-sbx: publish to 0.0.0.0 so the microVM can reach the gateway via
			// host.docker.internal (the Docker bridge gateway, 172.17.0.1).
			containerCmd.WriteString(" -p 0.0.0.0:${MCP_GATEWAY_PORT}:${MCP_GATEWAY_PORT}")
		} else {
			// Publish the gateway port to the host so host-side clients (e.g. Gemini CLI)
			// can reach the gateway at localhost:${MCP_GATEWAY_PORT}.
			containerCmd.WriteString(" -p 127.0.0.1:${MCP_GATEWAY_PORT}:${MCP_GATEWAY_PORT}")
		}
	} else {
		containerCmd.WriteString(" --network host")
	}
	containerCmd.WriteString(" --name awmg-mcpg")
	if !isAWFNetworkIsolationEnabled(workflowData) {
		containerCmd.WriteString(" --add-host host.docker.internal:127.0.0.1")
	} else if shouldRewriteLocalhostToDocker(workflowData) {
		// In bridge (network-isolation) mode the container's loopback differs from the
		// host's, so host.docker.internal:127.0.0.1 would not resolve to the host.
		// Use host-gateway (Docker 20.10+) instead so the gateway container can reach
		// any host-side server (mcp-scripts HTTP server, custom HTTP MCP tools with
		// localhost URLs) that is running directly on the runner host.
		containerCmd.WriteString(" --add-host host.docker.internal:host-gateway")
	}
	containerCmd.WriteString(" --user ${MCP_GATEWAY_UID}:${MCP_GATEWAY_GID}")
	containerCmd.WriteString(" --group-add ${DOCKER_SOCK_GID}")
	containerCmd.WriteString(" -v ${DOCKER_SOCK_PATH}:/var/run/docker.sock")
	appendMCPGatewayBaseEnvFlags(&containerCmd, payloadPathPrefix)
	appendMCPGatewayConditionalEnvFlags(&containerCmd, workflowData, engine, hasGitHub, githubTool, tools)
	appendMCPGatewaySafeOutputsInputEnvFlags(&containerCmd, safeOutputsInputEnvVars)
	appendMCPGatewayCustomAndHTTPEnvFlags(&containerCmd, workflowData, gatewayConfig, mcpEnvVars, hasGitHub, githubTool, tools, engine)
	if payloadDir != "" {
		containerCmd.WriteString(" -v " + payloadDir + ":" + payloadDir + ":rw")
	}
	for _, mount := range gatewayConfig.Mounts {
		containerCmd.WriteString(" -v " + mount)
	}
	if gatewayConfig.Entrypoint != "" {
		containerCmd.WriteString(" --entrypoint " + shellEscapeArg(gatewayConfig.Entrypoint))
	}
	containerCmd.WriteString(" " + containerImage)
	for _, arg := range gatewayConfig.EntrypointArgs {
		containerCmd.WriteString(" " + shellEscapeArg(arg))
	}
	for _, arg := range gatewayConfig.Args {
		containerCmd.WriteString(" " + shellEscapeArg(arg))
	}
	return containerCmd.String()
}

func appendMCPGatewayBaseEnvFlags(containerCmd *strings.Builder, payloadPathPrefix string) {
	containerCmd.WriteString(" -e MCP_GATEWAY_PORT")
	containerCmd.WriteString(" -e MCP_GATEWAY_DOMAIN")
	containerCmd.WriteString(" -e MCP_GATEWAY_API_KEY")
	containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_DIR")
	if payloadPathPrefix != "" {
		containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_PATH_PREFIX")
	}
	containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_SIZE_THRESHOLD")
	// Override DOCKER_HOST inside the gateway to match the fixed mount destination,
	// regardless of what the runner's DOCKER_HOST was (custom path, tcp://, etc.).
	containerCmd.WriteString(" -e DOCKER_HOST=unix:///var/run/docker.sock")
	containerCmd.WriteString(" -e DEBUG")
	containerCmd.WriteString(" -e MCP_GATEWAY_LOG_DIR")
	containerCmd.WriteString(" -e GH_AW_MCP_LOG_DIR")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_CONFIG_PATH")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_TOOLS_PATH")
	containerCmd.WriteString(" -e " + compilerenv.PolicyAllowCreatePullRequest)
	containerCmd.WriteString(" -e GH_AW_ASSETS_BRANCH")
	containerCmd.WriteString(" -e GH_AW_ASSETS_MAX_SIZE_KB")
	containerCmd.WriteString(" -e GH_AW_ASSETS_ALLOWED_EXTS")
	containerCmd.WriteString(" -e DEFAULT_BRANCH")
	containerCmd.WriteString(" -e GITHUB_MCP_SERVER_TOKEN")
	containerCmd.WriteString(" -e GITHUB_MCP_GUARD_MIN_INTEGRITY")
	containerCmd.WriteString(" -e GITHUB_MCP_GUARD_REPOS")
	containerCmd.WriteString(" -e " + sinkVisibilityEnvVar)
	containerCmd.WriteString(" -e GITHUB_REPOSITORY")
	containerCmd.WriteString(" -e GITHUB_SERVER_URL")
	containerCmd.WriteString(" -e GITHUB_SHA")
	containerCmd.WriteString(" -e GITHUB_WORKSPACE")
	containerCmd.WriteString(" -e GITHUB_TOKEN")
	containerCmd.WriteString(" -e GITHUB_RUN_ID")
	containerCmd.WriteString(" -e GITHUB_RUN_NUMBER")
	containerCmd.WriteString(" -e GITHUB_RUN_ATTEMPT")
	containerCmd.WriteString(" -e GITHUB_JOB")
	containerCmd.WriteString(" -e GITHUB_ACTION")
	containerCmd.WriteString(" -e GITHUB_EVENT_NAME")
	containerCmd.WriteString(" -e GITHUB_EVENT_PATH")
	containerCmd.WriteString(" -e GITHUB_ACTOR")
	containerCmd.WriteString(" -e GITHUB_ACTOR_ID")
	containerCmd.WriteString(" -e GITHUB_TRIGGERING_ACTOR")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW_REF")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW_SHA")
	containerCmd.WriteString(" -e GITHUB_REF")
	containerCmd.WriteString(" -e GITHUB_REF_NAME")
	containerCmd.WriteString(" -e GITHUB_REF_TYPE")
	containerCmd.WriteString(" -e GITHUB_HEAD_REF")
	containerCmd.WriteString(" -e GITHUB_BASE_REF")
	containerCmd.WriteString(" -e RUNNER_TEMP")
}

func appendMCPGatewayConditionalEnvFlags(containerCmd *strings.Builder, workflowData *WorkflowData, engine CodingAgentEngine, hasGitHub bool, githubTool map[string]any, tools map[string]any) {
	if hasGitHub && getGitHubType(githubTool) == GitHubMCPModeRemote && engine.GetID() == "copilot" {
		containerCmd.WriteString(" -e GITHUB_PERSONAL_ACCESS_TOKEN")
	}
	if IsMCPScriptsEnabled(workflowData.MCPScripts) {
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_PORT")
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_API_KEY")
	}
	if workflowData.OTLPEndpoint != "" {
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_TRACE_ID")
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_PARENT_SPAN_ID")
		// Pass OTEL_EXPORTER_OTLP_HEADERS as an env var so that auth credentials
		// are not embedded in the stdin JSON config pipe. mcpg reads this env var
		// as the standard OTel mechanism for providing OTLP authentication headers.
		containerCmd.WriteString(" -e OTEL_EXPORTER_OTLP_HEADERS")
	}
	if hasGitHubOIDCAuthInTools(tools) {
		containerCmd.WriteString(" -e ACTIONS_ID_TOKEN_REQUEST_URL")
		containerCmd.WriteString(" -e ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	}
}

// appendMCPGatewaySafeOutputsInputEnvFlags adds -e flags for GH_AW_INPUT_* environment variables
// that are referenced by the safe-outputs config. These variables are written into config.json as
// ${GH_AW_INPUT_…} shell-style placeholders at compile time and must be resolvable inside the
// containerised safe-outputs MCP server at runtime.
func appendMCPGatewaySafeOutputsInputEnvFlags(containerCmd *strings.Builder, safeOutputsInputEnvVars map[string]string) {
	if len(safeOutputsInputEnvVars) == 0 {
		return
	}
	envVarNames := sliceutil.SortedKeys(safeOutputsInputEnvVars)
	for _, envVarName := range envVarNames {
		containerCmd.WriteString(" -e " + envVarName)
	}
}

func appendMCPGatewayCustomAndHTTPEnvFlags(containerCmd *strings.Builder, workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, mcpEnvVars map[string]string, hasGitHub bool, githubTool map[string]any, tools map[string]any, engine CodingAgentEngine) {
	if len(gatewayConfig.Env) > 0 {
		envVarNames := sliceutil.MapKeys(gatewayConfig.Env)
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			containerCmd.WriteString(" -e " + envVarName)
		}
	}
	if len(mcpEnvVars) == 0 {
		return
	}
	addedEnvVars := buildAddedGatewayEnvVarSet(workflowData, gatewayConfig, hasGitHub, githubTool, tools, engine)
	var envVarNames []string
	for envVarName := range mcpEnvVars {
		if !setutil.Contains(addedEnvVars, envVarName) {
			envVarNames = append(envVarNames, envVarName)
		}
	}
	sort.Strings(envVarNames)
	for _, envVarName := range envVarNames {
		containerCmd.WriteString(" -e " + envVarName)
	}
	if mcpSetupGeneratorLog.Enabled() && len(envVarNames) > 0 {
		mcpSetupGeneratorLog.Printf("Added %d HTTP MCP environment variables to gateway container: %v", len(envVarNames), envVarNames)
	}
}

func buildAddedGatewayEnvVarSet(workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, hasGitHub bool, githubTool map[string]any, tools map[string]any, engine CodingAgentEngine) map[string]struct{} {
	addedEnvVars := make(map[string]struct{})
	standardEnvVars := []string{
		"MCP_GATEWAY_PORT", "MCP_GATEWAY_DOMAIN", "MCP_GATEWAY_API_KEY", "MCP_GATEWAY_PAYLOAD_DIR", "DEBUG",
		"MCP_GATEWAY_LOG_DIR", "GH_AW_MCP_LOG_DIR", "GH_AW_SAFE_OUTPUTS",
		"GH_AW_SAFE_OUTPUTS_CONFIG_PATH", "GH_AW_SAFE_OUTPUTS_TOOLS_PATH", compilerenv.PolicyAllowCreatePullRequest,
		"GH_AW_ASSETS_BRANCH", "GH_AW_ASSETS_MAX_SIZE_KB", "GH_AW_ASSETS_ALLOWED_EXTS",
		"DEFAULT_BRANCH", "GITHUB_MCP_SERVER_TOKEN", "GITHUB_MCP_GUARD_MIN_INTEGRITY", "GITHUB_MCP_GUARD_REPOS",
		sinkVisibilityEnvVar,
		"GITHUB_REPOSITORY", "GITHUB_SERVER_URL", "GITHUB_SHA", "GITHUB_WORKSPACE",
		"RUNNER_TEMP",
		"GITHUB_TOKEN", "GITHUB_RUN_ID", "GITHUB_RUN_NUMBER", "GITHUB_RUN_ATTEMPT",
		"GITHUB_JOB", "GITHUB_ACTION", "GITHUB_EVENT_NAME", "GITHUB_EVENT_PATH",
		"GITHUB_ACTOR", "GITHUB_ACTOR_ID", "GITHUB_TRIGGERING_ACTOR",
		"GITHUB_WORKFLOW", "GITHUB_WORKFLOW_REF", "GITHUB_WORKFLOW_SHA",
		"GITHUB_REF", "GITHUB_REF_NAME", "GITHUB_REF_TYPE", "GITHUB_HEAD_REF", "GITHUB_BASE_REF",
	}
	for _, envVar := range standardEnvVars {
		addedEnvVars[envVar] = struct{}{}
	}
	if hasGitHub && getGitHubType(githubTool) == GitHubMCPModeRemote && engine.GetID() == "copilot" {
		addedEnvVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = struct{}{}
	}
	if IsMCPScriptsEnabled(workflowData.MCPScripts) {
		addedEnvVars["GH_AW_MCP_SCRIPTS_PORT"] = struct{}{}
		addedEnvVars["GH_AW_MCP_SCRIPTS_API_KEY"] = struct{}{}
	}
	if workflowData.OTLPEndpoint != "" {
		addedEnvVars["GITHUB_AW_OTEL_TRACE_ID"] = struct{}{}
		addedEnvVars["GITHUB_AW_OTEL_PARENT_SPAN_ID"] = struct{}{}
		addedEnvVars["OTEL_EXPORTER_OTLP_HEADERS"] = struct{}{}
	}
	if hasGitHubOIDCAuthInTools(tools) {
		addedEnvVars["ACTIONS_ID_TOKEN_REQUEST_URL"] = struct{}{}
		addedEnvVars["ACTIONS_ID_TOKEN_REQUEST_TOKEN"] = struct{}{}
	}
	for envVarName := range gatewayConfig.Env {
		addedEnvVars[envVarName] = struct{}{}
	}
	return addedEnvVars
}
