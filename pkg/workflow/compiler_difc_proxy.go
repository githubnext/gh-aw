package workflow

// Package workflow provides DIFC proxy injection for pre-agent gh CLI steps.
//
// # DIFC Proxy Injection
//
// When DIFC guards are configured (min-integrity set), the compiler injects
// a temporary proxy (awmg-proxy) that routes pre-agent gh CLI calls through
// integrity filtering. This ensures that custom steps referencing GH_TOKEN see
// DIFC-filtered API responses, matching the integrity guarantees the agent
// itself operates under.
//
// Note: repo-memory clone steps use a direct "git clone https://x-access-token:${GH_TOKEN}@..."
// URL derived from GITHUB_SERVER_URL, not GH_HOST, so they bypass the proxy even when it
// is running. Only gh CLI calls that honour GH_HOST are actually filtered.
//
// The proxy uses the same container image as the MCP gateway (gh-aw-mcpg)
// but runs in "proxy" mode with --guards-mode filter (graceful degradation)
// and --tls (required by the gh CLI HTTPS-only constraint).
//
// Injection conditions:
//
//	Main job:     GitHub tool has explicit guard policies (min-integrity set) AND
//	              custom steps set GH_TOKEN
//	Indexing job: GitHub tool has explicit guard policies (min-integrity set)
//
// Proxy lifecycle within the main job:
//  1. Start proxy — after "Configure gh CLI" step, before custom steps
//  2. Custom steps run with GH_HOST=localhost:18443, GITHUB_API_URL, GITHUB_GRAPHQL_URL,
//     and NODE_EXTRA_CA_CERTS set (via $GITHUB_ENV)
//  3. Stop proxy — before MCP gateway starts (generateMCPSetup); always runs
//     even if earlier steps failed (if: always(), continue-on-error: true)
//
// Proxy lifecycle within the indexing job:
//  1. Start proxy — before index-building steps
//  2. Steps run with all proxy env vars set (GH_HOST, GITHUB_API_URL, GITHUB_GRAPHQL_URL,
//     NODE_EXTRA_CA_CERTS); Octokit calls in actions/github-script are intercepted
//  3. Stop proxy — after steps; always runs (if: always(), continue-on-error: true)
//
// Guard policy note:
//
// The proxy policy uses only the static fields from the workflow's frontmatter
// (min-integrity and repos). The dynamic blocked-users and approval-labels fields
// (which reference outputs from the parse-guard-vars step) are NOT included,
// because that step runs after the proxy starts. Basic integrity filtering is
// still enforced through min-integrity and repos.
//
// Log directories:
//
// The proxy and gateway share /tmp/gh-aw/mcp-logs/ for JSONL output (both append
// to rpc-messages.jsonl in chronological order). The proxy also writes TLS certs
// and container stderr to /tmp/gh-aw/proxy-logs/.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var difcProxyLog = logger.New("workflow:difc_proxy")

// hasDIFCGuardsConfigured returns true if the GitHub tool has explicit guard policies configured
// (min-integrity is set) AND the DIFC proxy has not been explicitly disabled via
// tools.github.integrity-proxy: false.
// This is the base condition for DIFC proxy injection.
func hasDIFCGuardsConfigured(data *WorkflowData) bool {
	if data == nil {
		return false
	}
	if !isIntegrityProxyEnabled(data) {
		difcProxyLog.Print("integrity-proxy disabled via tools.github.integrity-proxy: false, skipping DIFC proxy injection")
		return false
	}
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return false
	}
	return len(getGitHubGuardPolicies(githubTool)) > 0
}

// isIntegrityProxyEnabled returns true unless the user has explicitly disabled the DIFC proxy
// by setting tools.github.integrity-proxy: false.
// The proxy is enabled by default (opt-out model): absent or true → enabled; false → disabled.
func isIntegrityProxyEnabled(data *WorkflowData) bool {
	if data == nil {
		return true
	}
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub {
		return true
	}
	toolConfig, ok := githubTool.(map[string]any)
	if !ok {
		return true
	}
	val, hasField := toolConfig["integrity-proxy"]
	if !hasField {
		return true // default: enabled
	}
	if enabled, ok := val.(bool); ok {
		return enabled
	}
	return true
}

// hasDIFCProxyNeeded returns true if the DIFC proxy should be injected in the main job.
//
// The proxy is only needed when:
//  1. The GitHub tool has explicit guard policies (min-integrity is set), and
//  2. There are pre-agent steps that may call the gh CLI (identified by GH_TOKEN use
//     in custom steps, or by the presence of repo-memory configuration whose clone
//     steps always set GH_TOKEN).
func hasDIFCProxyNeeded(data *WorkflowData) bool {
	if !hasDIFCGuardsConfigured(data) {
		difcProxyLog.Print("No explicit guard policies configured, skipping DIFC proxy injection")
		return false
	}

	// Check if there are pre-agent steps that set GH_TOKEN
	if !hasPreAgentStepsWithGHToken(data) {
		difcProxyLog.Print("No pre-agent steps with GH_TOKEN, skipping DIFC proxy injection")
		return false
	}

	difcProxyLog.Print("DIFC proxy needed: guard policies configured and pre-agent steps have GH_TOKEN")
	return true
}

// hasPreAgentStepsWithGHToken returns true if there are pre-agent steps that set GH_TOKEN.
//
// The heuristic checks whether custom steps (from data.CustomSteps) reference GH_TOKEN.
//
// Note: repo-memory clone steps use a direct "git clone https://x-access-token:${GH_TOKEN}@..."
// URL derived from GITHUB_SERVER_URL, not GH_HOST, so they are not intercepted by the proxy
// and are therefore not counted here.
func hasPreAgentStepsWithGHToken(data *WorkflowData) bool {
	if data == nil {
		return false
	}

	// Check if custom steps reference GH_TOKEN
	if strings.Contains(data.CustomSteps, "GH_TOKEN") {
		difcProxyLog.Print("Custom steps contain GH_TOKEN, proxy needed")
		return true
	}

	return false
}

// getDIFCProxyPolicyJSON returns a JSON-encoded guard policy for the DIFC proxy.
//
// Unlike the gateway policy (which includes dynamic blocked-users and approval-labels
// from step outputs), the proxy policy only includes the static fields available at
// compile time: min-integrity and repos. This is because the proxy starts before the
// parse-guard-vars step that produces those dynamic outputs.
//
// When the integrity-reactions feature flag is enabled and the MCPG version supports it,
// reaction fields (endorsement-reactions, disapproval-reactions, disapproval-integrity,
// endorser-min-integrity) are also included in the proxy policy.
//
// Returns an empty string if no guard policy fields are found.
func getDIFCProxyPolicyJSON(githubTool any, data *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig) string {
	toolConfig, ok := githubTool.(map[string]any)
	if !ok {
		return ""
	}

	policy := make(map[string]any)

	// Support both 'allowed-repos' (preferred) and deprecated 'repos'
	repos, hasRepos := toolConfig["allowed-repos"]
	if !hasRepos {
		repos, hasRepos = toolConfig["repos"]
	}
	integrity, hasIntegrity := toolConfig["min-integrity"]

	if !hasRepos && !hasIntegrity {
		return ""
	}

	if hasRepos {
		policy["repos"] = repos
	} else {
		// Default repos to "all" when min-integrity is specified without repos
		policy["repos"] = "all"
	}

	if hasIntegrity {
		policy["min-integrity"] = integrity
	}

	// Inject reaction fields when the feature flag is enabled and MCPG supports it.
	injectIntegrityReactionFields(policy, toolConfig, data, gatewayConfig)

	guardPolicy := map[string]any{
		"allow-only": policy,
	}

	jsonBytes, err := json.Marshal(guardPolicy)
	if err != nil {
		difcProxyLog.Printf("Failed to marshal DIFC proxy policy: %v", err)
		return ""
	}

	return string(jsonBytes)
}

// resolveProxyContainerImage returns the full container image reference (container:version)
// for the DIFC/CLI proxy, falling back to the default MCP gateway version if none is configured.
func resolveProxyContainerImage(gatewayConfig *MCPGatewayRuntimeConfig) string {
	version := gatewayConfig.Version
	if version == "" {
		version = string(constants.DefaultMCPGatewayVersion)
	}
	return gatewayConfig.Container + ":" + version
}

// buildStartDIFCProxyStepYAML returns the YAML for the "Start DIFC proxy" step,
// or an empty string if proxy injection is not needed or the policy cannot be built.
// This is the shared implementation used by both the main job and the indexing job.
func (c *Compiler) buildStartDIFCProxyStepYAML(data *WorkflowData) string {
	difcProxyLog.Print("Building Start DIFC proxy step YAML")

	githubTool := data.Tools["github"]

	// Get MCP server token (same token the gateway uses for the GitHub MCP server)
	customGitHubToken := getGitHubToken(githubTool)
	effectiveToken := getEffectiveGitHubToken(customGitHubToken)

	// Build the simplified guard policy JSON (static fields only)
	// (plus reaction fields when integrity-reactions feature flag is enabled)
	ensureDefaultMCPGatewayConfig(data)
	policyJSON := getDIFCProxyPolicyJSON(githubTool, data, data.SandboxConfig.MCP)
	if policyJSON == "" {
		difcProxyLog.Print("Could not build DIFC proxy policy JSON, skipping proxy start")
		return ""
	}

	// Resolve the container image from the MCP gateway configuration
	// (proxy uses the same image as the gateway, just in "proxy" mode)
	containerImage := resolveProxyContainerImage(data.SandboxConfig.MCP)

	var sb strings.Builder
	sb.WriteString("      - name: Start DIFC proxy for pre-agent gh calls\n")
	sb.WriteString("        env:\n")
	fmt.Fprintf(&sb, "          GH_TOKEN: %s\n", effectiveToken)
	sb.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	// Store policy and image in env vars to avoid shell-quoting issues with
	// inline JSON arguments and to keep the run: command clean.
	fmt.Fprintf(&sb, "          DIFC_PROXY_POLICY: '%s'\n", policyJSON)
	fmt.Fprintf(&sb, "          DIFC_PROXY_IMAGE: '%s'\n", containerImage)
	sb.WriteString("        run: |\n")
	sb.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/start_difc_proxy.sh\"\n")
	return sb.String()
}

// generateStartDIFCProxyStep generates a step that starts the DIFC proxy container
// before pre-agent gh CLI steps. The proxy routes gh API calls through integrity filtering.
//
// The step is only emitted when hasDIFCProxyNeeded returns true.
// The generated step calls start_difc_proxy.sh with the guard policy JSON and container image.
func (c *Compiler) generateStartDIFCProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !hasDIFCProxyNeeded(data) {
		return
	}

	step := c.buildStartDIFCProxyStepYAML(data)
	if step != "" {
		yaml.WriteString(step)
	}
}

// buildSetGHRepoStepYAML returns the YAML for the "Set GH_REPO for proxied steps" step.
//
// start_difc_proxy.sh writes GH_HOST=localhost:18443 to GITHUB_ENV so that the gh CLI
// routes through the proxy. However, gh CLI infers the target repository from the git
// remote, which uses the original host (github.com / GHEC host). When GH_HOST is the
// proxy address, gh fails to resolve the repository because the host doesn't match.
//
// Rather than overwriting GH_HOST (which would bypass the DIFC proxy's integrity
// filtering), this step sets GH_REPO=$GITHUB_REPOSITORY. The gh CLI respects GH_REPO
// to determine the target repository without needing to match the git remote host.
// GH_HOST stays at localhost:18443 so all gh CLI traffic continues routing through
// the proxy for integrity filtering.
func buildSetGHRepoStepYAML() string {
	var sb strings.Builder
	sb.WriteString("      - name: Set GH_REPO for proxied steps\n")
	sb.WriteString("        run: |\n")
	sb.WriteString("          echo \"GH_REPO=${GITHUB_REPOSITORY}\" >> \"$GITHUB_ENV\"\n")
	return sb.String()
}

// generateSetGHRepoAfterDIFCProxyStep injects a step that sets GH_REPO=$GITHUB_REPOSITORY
// after start_difc_proxy.sh and before user-defined setup steps.
//
// The proxy sets GH_HOST=localhost:18443 in GITHUB_ENV, which causes gh CLI to fail
// resolving the repository from the git remote. Setting GH_REPO tells gh which repo
// to target without changing the proxy routing (GH_HOST stays at the proxy address).
//
// The step is only emitted when hasDIFCProxyNeeded returns true.
func (c *Compiler) generateSetGHRepoAfterDIFCProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !hasDIFCProxyNeeded(data) {
		return
	}

	difcProxyLog.Print("Generating Set GH_REPO step after DIFC proxy start")
	yaml.WriteString(buildSetGHRepoStepYAML())
}

// generateStopDIFCProxyStep generates a step that stops the DIFC proxy container
// before the MCP gateway starts. The proxy must be stopped first to avoid
// double-filtering: the gateway uses the same guard policy for the agent phase.
//
// The step runs even if earlier steps failed (if: always(), continue-on-error: true)
// to ensure the proxy container and CA cert are always cleaned up.
//
// The step is only emitted when hasDIFCProxyNeeded returns true.
func (c *Compiler) generateStopDIFCProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !hasDIFCProxyNeeded(data) {
		return
	}

	difcProxyLog.Print("Generating Stop DIFC proxy step")

	yaml.WriteString("      - name: Stop DIFC proxy\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/stop_difc_proxy.sh\"\n")
}

// isCliProxyNeeded returns true if the CLI proxy should be started on the host.
//
// The CLI proxy is needed when:
//  1. The cli-proxy feature flag is enabled, and
//  2. The AWF sandbox (firewall) is enabled, and
//  3. The AWF version supports CLI proxy flags
func isCliProxyNeeded(data *WorkflowData) bool {
	if !isFeatureEnabled(constants.CliProxyFeatureFlag, data) {
		return false
	}
	if !isFirewallEnabled(data) {
		return false
	}
	firewallConfig := getFirewallConfig(data)
	if !awfSupportsCliProxy(firewallConfig) {
		difcProxyLog.Printf("Skipping CLI proxy: AWF version too old")
		return false
	}
	return true
}

// generateStartCliProxyStep generates a step that starts the difc-proxy container
// on the host before the AWF execution step. AWF's cli-proxy sidecar connects
// to this host proxy via host.docker.internal:18443.
//
// The step is only emitted when isCliProxyNeeded returns true.
func (c *Compiler) generateStartCliProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !isCliProxyNeeded(data) {
		return
	}

	step := c.buildStartCliProxyStepYAML(data)
	if step != "" {
		yaml.WriteString(step)
	}
}

// defaultCliProxyPolicyJSON is the fallback guard policy for the CLI proxy when no
// guard policy is explicitly configured in the workflow frontmatter.
// The DIFC proxy requires a --policy flag to forward requests; without it, all API
// calls return HTTP 503 with body "proxy enforcement not configured".
// This default allows all repos with no integrity filtering — the most permissive
// policy that still satisfies the proxy's requirement.
const defaultCliProxyPolicyJSON = `{"allow-only":{"repos":"all","min-integrity":"none"}}`

// buildStartCliProxyStepYAML returns the YAML for the "Start CLI proxy" step,
// or an empty string if the proxy cannot be configured.
func (c *Compiler) buildStartCliProxyStepYAML(data *WorkflowData) string {
	difcProxyLog.Print("Building Start CLI proxy step YAML")

	githubTool := data.Tools["github"]

	// Get token for the proxy
	customGitHubToken := getGitHubToken(githubTool)
	effectiveToken := getEffectiveGitHubToken(customGitHubToken)

	// Build the guard policy JSON (static fields only, plus reaction fields when enabled).
	// The CLI proxy requires a policy to forward requests — without one, all API
	// calls return HTTP 503 ("proxy enforcement not configured"). Use the default
	// permissive policy when no guard policy is configured in the frontmatter.
	ensureDefaultMCPGatewayConfig(data)
	policyJSON := getDIFCProxyPolicyJSON(githubTool, data, data.SandboxConfig.MCP)
	if policyJSON == "" {
		policyJSON = defaultCliProxyPolicyJSON
		difcProxyLog.Print("No guard policy configured, using default CLI proxy policy")
	}

	// Resolve the container image from the MCP gateway configuration
	containerImage := resolveProxyContainerImage(data.SandboxConfig.MCP)

	var sb strings.Builder
	sb.WriteString("      - name: Start CLI proxy\n")
	sb.WriteString("        env:\n")
	fmt.Fprintf(&sb, "          GH_TOKEN: %s\n", effectiveToken)
	sb.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	fmt.Fprintf(&sb, "          CLI_PROXY_POLICY: '%s'\n", policyJSON)
	fmt.Fprintf(&sb, "          CLI_PROXY_IMAGE: '%s'\n", containerImage)
	sb.WriteString("        run: |\n")
	sb.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/start_cli_proxy.sh\"\n")
	return sb.String()
}

// generateStopCliProxyStep generates a step that stops the CLI proxy container
// after the AWF execution step.
//
// The step runs even if earlier steps failed (if: always(), continue-on-error: true)
// to ensure the proxy container is always cleaned up.
//
// The step is only emitted when isCliProxyNeeded returns true.
func (c *Compiler) generateStopCliProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !isCliProxyNeeded(data) {
		return
	}

	difcProxyLog.Print("Generating Stop CLI proxy step")

	yaml.WriteString("      - name: Stop CLI proxy\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/stop_cli_proxy.sh\"\n")
}

// difcProxyLogPaths returns the artifact paths for DIFC proxy logs.
// Returns an empty slice when no DIFC proxy is needed or configured.
func difcProxyLogPaths(data *WorkflowData) []string {
	// Return proxy-logs path if proxy is needed in either the main job or the indexing job.
	// hasDIFCGuardsConfigured covers the indexing job case (guard policies alone are sufficient).
	if !hasDIFCGuardsConfigured(data) {
		return nil
	}
	// proxy-logs/ contains TLS certs and container stderr from the proxy.
	// Exclude proxy-tls/ to avoid uploading TLS material (mcp-logs/ is already
	// collected as part of standard MCP logging).
	return []string{
		"/tmp/gh-aw/proxy-logs/",
		"!/tmp/gh-aw/proxy-logs/proxy-tls/",
	}
}
