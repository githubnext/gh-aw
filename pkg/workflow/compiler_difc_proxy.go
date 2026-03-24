package workflow

// Package workflow provides DIFC proxy injection for pre-agent gh CLI steps.
//
// # DIFC Proxy Injection
//
// When DIFC guards are configured (min-integrity set), the compiler injects
// a temporary proxy (awmg-proxy) that routes pre-agent gh CLI calls through
// integrity filtering. This ensures that steps like fetching issues for task
// weighting and cloning repo-memory branches see DIFC-filtered API responses,
// matching the integrity guarantees the agent itself operates under.
//
// The proxy uses the same container image as the MCP gateway (gh-aw-mcpg)
// but runs in "proxy" mode with --guards-mode filter (graceful degradation)
// and --tls (required by the gh CLI HTTPS-only constraint).
//
// Injection conditions (both must be true):
//  1. GitHub tool has explicit guard policies (min-integrity set)
//  2. Pre-agent steps set GH_TOKEN (custom steps or repo-memory steps)
//
// Proxy lifecycle within the main job:
//  1. Start proxy — after "Configure gh CLI" step, before custom steps
//  2. Custom steps + repo-memory steps run with GH_HOST=localhost:18443 (set via $GITHUB_ENV)
//  3. Stop proxy — before MCP gateway starts (generateMCPSetup)
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

// hasDIFCProxyNeeded returns true if the DIFC proxy should be injected.
//
// The proxy is only needed when:
//  1. The GitHub tool has explicit guard policies (min-integrity is set), and
//  2. There are pre-agent steps that may call the gh CLI (identified by GH_TOKEN use
//     in custom steps, or by the presence of repo-memory configuration whose clone
//     steps always set GH_TOKEN).
func hasDIFCProxyNeeded(data *WorkflowData) bool {
	if data == nil {
		return false
	}

	// Check if GitHub tool has explicit guard policies (min-integrity set)
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return false
	}

	if len(getGitHubGuardPolicies(githubTool)) == 0 {
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
// The heuristic checks:
//   - Custom steps (from data.CustomSteps) contain the string "GH_TOKEN"
//   - Repo-memory is configured (its clone steps always set GH_TOKEN: ${{ github.token }})
func hasPreAgentStepsWithGHToken(data *WorkflowData) bool {
	if data == nil {
		return false
	}

	// Check if custom steps reference GH_TOKEN
	if strings.Contains(data.CustomSteps, "GH_TOKEN") {
		difcProxyLog.Print("Custom steps contain GH_TOKEN, proxy needed")
		return true
	}

	// Check if repo-memory is configured (clone steps always use GH_TOKEN)
	if data.RepoMemoryConfig != nil && len(data.RepoMemoryConfig.Memories) > 0 {
		difcProxyLog.Print("Repo-memory configured (uses GH_TOKEN for clone), proxy needed")
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
// Returns an empty string if no guard policy fields are found.
func getDIFCProxyPolicyJSON(githubTool any) string {
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

// generateStartDIFCProxyStep generates a step that starts the DIFC proxy container
// before pre-agent gh CLI steps. The proxy routes gh API calls through integrity filtering.
//
// The step is only emitted when hasDIFCProxyNeeded returns true.
// The generated step calls start_difc_proxy.sh with the guard policy JSON and container image.
func (c *Compiler) generateStartDIFCProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !hasDIFCProxyNeeded(data) {
		return
	}

	difcProxyLog.Print("Generating Start DIFC proxy step")

	githubTool := data.Tools["github"]

	// Get MCP server token (same token the gateway uses for the GitHub MCP server)
	customGitHubToken := getGitHubToken(githubTool)
	effectiveToken := getEffectiveGitHubToken(customGitHubToken)

	// Build the simplified guard policy JSON (static fields only)
	policyJSON := getDIFCProxyPolicyJSON(githubTool)
	if policyJSON == "" {
		difcProxyLog.Print("Could not build DIFC proxy policy JSON, skipping proxy start")
		return
	}

	// Resolve the container image from the MCP gateway configuration
	// (proxy uses the same image as the gateway, just in "proxy" mode)
	ensureDefaultMCPGatewayConfig(data)
	gatewayConfig := data.SandboxConfig.MCP

	containerImage := gatewayConfig.Container
	if gatewayConfig.Version != "" {
		containerImage += ":" + gatewayConfig.Version
	} else {
		containerImage += ":" + string(constants.DefaultMCPGatewayVersion)
	}

	yaml.WriteString("      - name: Start DIFC proxy for pre-agent gh calls\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_TOKEN: %s\n", effectiveToken)
	yaml.WriteString("        run: |\n")
	// The policy JSON contains only static values from the workflow frontmatter
	// (min-integrity and repos). It never contains GitHub Actions expressions (${{ }})
	// because getDIFCProxyPolicyJSON() only includes compile-time values, making
	// single-quoting safe here.
	fmt.Fprintf(yaml, "          bash ${RUNNER_TEMP}/gh-aw/actions/start_difc_proxy.sh '%s' '%s'\n", policyJSON, containerImage)
}

// generateStopDIFCProxyStep generates a step that stops the DIFC proxy container
// before the MCP gateway starts. The proxy must be stopped first to avoid
// double-filtering: the gateway uses the same guard policy for the agent phase.
//
// The step is only emitted when hasDIFCProxyNeeded returns true.
func (c *Compiler) generateStopDIFCProxyStep(yaml *strings.Builder, data *WorkflowData) {
	if !hasDIFCProxyNeeded(data) {
		return
	}

	difcProxyLog.Print("Generating Stop DIFC proxy step")

	yaml.WriteString("      - name: Stop DIFC proxy\n")
	yaml.WriteString("        run: bash ${RUNNER_TEMP}/gh-aw/actions/stop_difc_proxy.sh\n")
}

// difcProxyLogPaths returns the artifact paths for DIFC proxy logs.
// Returns an empty slice when the proxy is not needed.
func difcProxyLogPaths(data *WorkflowData) []string {
	if !hasDIFCProxyNeeded(data) {
		return nil
	}
	// proxy-logs/ contains TLS certs and container stderr from the proxy
	// (mcp-logs/ is already collected as part of standard MCP logging)
	return []string{"/tmp/gh-aw/proxy-logs/"}
}
