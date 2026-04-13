// Package workflow provides GitHub MCP server configuration and toolset management.
//
// # GitHub MCP Server Configuration
//
// This file manages the configuration of the GitHub MCP server, which provides
// AI agents with access to GitHub's API through the Model Context Protocol (MCP).
// It handles both local (Docker-based) and remote (hosted) deployment modes.
//
// Key responsibilities:
//   - Extracting GitHub tool configuration from workflow frontmatter
//   - Managing GitHub MCP server modes (local Docker vs remote hosted)
//   - Handling GitHub authentication tokens (custom, default, GitHub App)
//   - Managing read-only and lockdown security modes
//   - Expanding and managing GitHub toolsets (repos, issues, pull_requests, etc.)
//   - Handling allowed tool lists for fine-grained access control
//   - Determining Docker image versions for local mode
//   - Generating automatic lockdown detection steps
//   - Managing GitHub App token minting and invalidation
//
// GitHub MCP modes:
//   - Local (default): Runs GitHub MCP server in Docker container
//   - Remote: Uses hosted GitHub MCP service
//
// Security features:
//   - Read-only mode: Always enforced - write operations via GitHub MCP are not permitted
//   - GitHub lockdown mode: Restricts access to current repository only
//   - Automatic lockdown: Enables lockdown for public repositories with GH_AW_GITHUB_TOKEN
//   - Allowed tools: Restricts available GitHub API operations
//
// GitHub toolsets:
//   - default/action-friendly: Standard toolsets safe for GitHub Actions
//   - repos, issues, pull_requests, discussions, search, code_scanning
//   - secret_scanning, labels, releases, milestones, projects, gists
//   - teams, actions, packages (requires specific permissions)
//   - users (excluded from action-friendly due to token limitations)
//
// Token precedence:
//  1. GitHub App token (minted from app configuration)
//  2. Custom github-token from tool configuration
//  3. Top-level github-token from frontmatter
//  4. Default GITHUB_TOKEN secret
//
// Automatic lockdown detection:
// When lockdown is not explicitly set, a step is generated to automatically
// enable lockdown for public repositories ONLY when GH_AW_GITHUB_TOKEN is configured.
//
// Related files:
//   - mcp_renderer.go: Renders GitHub MCP configuration to YAML
//   - mcp_environment.go: Manages GitHub MCP environment variables
//   - mcp_setup_generator.go: Generates GitHub MCP setup steps
//   - safe_outputs_app.go: GitHub App token minting helpers
//
// Example configuration:
//
//	tools:
//	  github:
//	    mode: remote                    # or "local" for Docker
//	    github-token: ${{ secrets.PAT }}
//	    lockdown: true                  # or omit for automatic detection
//	    toolsets: [repos, issues, pull_requests]
//	    allowed: [get_repo, list_issues, get_pull_request]
package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/semverutil"
)

var githubConfigLog = logger.New("workflow:mcp_github_config")

// hasGitHubTool checks if the GitHub tool is configured (using ParsedTools)
func hasGitHubTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.GitHub != nil
}

// hasGitHubApp checks if a GitHub App is configured in the (merged) GitHub tool configuration
func hasGitHubApp(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		_, hasGitHubApp := toolConfig["github-app"]
		return hasGitHubApp
	}
	return false
}

// getGitHubType extracts the mode from GitHub tool configuration (local or remote)
func getGitHubType(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if modeSetting, exists := toolConfig["mode"]; exists {
			if stringValue, ok := modeSetting.(string); ok {
				githubConfigLog.Printf("GitHub MCP mode set explicitly: %s", stringValue)
				return stringValue
			}
		}
	}
	githubConfigLog.Print("GitHub MCP mode: local (default)")
	return "local" // default to local (Docker)
}

// getGitHubToken extracts the custom github-token from GitHub tool configuration
func getGitHubToken(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if tokenSetting, exists := toolConfig["github-token"]; exists {
			if stringValue, ok := tokenSetting.(string); ok {
				return stringValue
			}
		}
	}
	return ""
}

// getGitHubReadOnly returns true always, since the GitHub MCP server is always read-only.
// Setting read-only: false is not supported and will be flagged as a validation error.
func getGitHubReadOnly(_ any) bool {
	return true
}

// getGitHubLockdown checks if lockdown mode is enabled for GitHub tool
// Defaults to constants.DefaultGitHubLockdown (false)
func getGitHubLockdown(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if lockdownSetting, exists := toolConfig["lockdown"]; exists {
			if boolValue, ok := lockdownSetting.(bool); ok {
				return boolValue
			}
		}
	}
	return constants.DefaultGitHubLockdown
}

// hasGitHubLockdownExplicitlySet checks if lockdown field is explicitly set in GitHub tool config
func hasGitHubLockdownExplicitlySet(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		_, exists := toolConfig["lockdown"]
		return exists
	}
	return false
}

// getGitHubToolsets extracts the toolsets configuration from GitHub tool
// Expands "default" to individual toolsets for action-friendly compatibility
func getGitHubToolsets(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if toolsetsSetting, exists := toolConfig["toolsets"]; exists {
			// Handle array format only
			switch v := toolsetsSetting.(type) {
			case []any:
				// Convert array to comma-separated string
				toolsets := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						toolsets = append(toolsets, str)
					}
				}
				toolsetsStr := strings.Join(toolsets, ",")
				// Expand "default" to individual toolsets for action-friendly compatibility
				resolved := expandDefaultToolset(toolsetsStr)
				githubConfigLog.Printf("GitHub MCP toolsets resolved: %s", resolved)
				return resolved
			case []string:
				toolsetsStr := strings.Join(v, ",")
				// Expand "default" to individual toolsets for action-friendly compatibility
				resolved := expandDefaultToolset(toolsetsStr)
				githubConfigLog.Printf("GitHub MCP toolsets resolved: %s", resolved)
				return resolved
			}
		}
	}
	// default to action-friendly toolsets (excludes "users" which GitHub Actions tokens don't support)
	githubConfigLog.Print("GitHub MCP toolsets: using default action-friendly toolsets")
	return strings.Join(ActionFriendlyGitHubToolsets, ",")
}

// expandDefaultToolset expands "default" and "action-friendly" keywords to individual toolsets.
// This ensures that "default" and "action-friendly" in the source expand to action-friendly toolsets
// (excluding "users" which GitHub Actions tokens don't support).
func expandDefaultToolset(toolsetsStr string) string {
	if toolsetsStr == "" {
		return strings.Join(ActionFriendlyGitHubToolsets, ",")
	}

	// Split by comma and check if "default" or "action-friendly" is present
	toolsets := strings.Split(toolsetsStr, ",")
	var result []string
	seenToolsets := make(map[string]bool)

	for _, toolset := range toolsets {
		toolset = strings.TrimSpace(toolset)
		if toolset == "" {
			continue
		}

		if toolset == "default" || toolset == "action-friendly" {
			githubConfigLog.Printf("Expanding %q keyword to action-friendly toolsets", toolset)
			// Expand "default" or "action-friendly" to action-friendly toolsets (excludes "users")
			for _, dt := range ActionFriendlyGitHubToolsets {
				if !seenToolsets[dt] {
					result = append(result, dt)
					seenToolsets[dt] = true
				}
			}
		} else {
			// Keep other toolsets as-is (including "all", individual toolsets, etc.)
			if !seenToolsets[toolset] {
				result = append(result, toolset)
				seenToolsets[toolset] = true
			}
		}
	}

	return strings.Join(result, ",")
}

// getGitHubAllowedTools extracts the allowed tools list from GitHub tool configuration
// Returns the list of allowed tools, or nil if no allowed list is specified (which means all tools are allowed)
func getGitHubAllowedTools(githubTool any) []string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if allowedSetting, exists := toolConfig["allowed"]; exists {
			// Handle array format
			switch v := allowedSetting.(type) {
			case []any:
				// Convert array to string slice
				tools := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						tools = append(tools, str)
					}
				}
				return tools
			case []string:
				return v
			}
		}
	}
	return nil
}

// getGitHubGuardPolicies extracts guard policies from GitHub tool configuration.
// It reads the flat allowed-repos/repos/min-integrity/blocked-users/trusted-users/approval-labels fields
// and wraps them for MCP gateway rendering.
// When min-integrity is set but allowed-repos is not, repos defaults to "all" because the MCP
// Gateway requires repos to be present in the allow-only policy.
// Note: repos-only (without min-integrity) is rejected earlier by validateGitHubGuardPolicy,
// so this function will never be called with repos but without min-integrity in practice.
// When blocked-users, trusted-users, or approval-labels are set, their values are unioned with
// the org/repo variable fallback expressions so that a centrally-configured variable extends the
// per-workflow list rather than replacing it.
// Returns nil if no guard policies are configured.
func getGitHubGuardPolicies(githubTool any) map[string]any {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		// Support both 'allowed-repos' (preferred) and deprecated 'repos'
		repos, hasRepos := toolConfig["allowed-repos"]
		if !hasRepos {
			repos, hasRepos = toolConfig["repos"]
		}
		integrity, hasIntegrity := toolConfig["min-integrity"]
		if hasRepos || hasIntegrity {
			policy := map[string]any{}
			if hasRepos {
				policy["repos"] = repos
			} else {
				// Default repos to "all" when min-integrity is specified without repos.
				// The MCP Gateway requires repos in the allow-only policy.
				policy["repos"] = "all"
			}
			if hasIntegrity {
				policy["min-integrity"] = integrity
			}
			// blocked-users, trusted-users, and approval-labels are parsed at runtime by the
			// parse-guard-vars step. The step outputs proper JSON arrays (split on comma/newline,
			// validated, jq-encoded) from both the compile-time static values and the
			// GH_AW_GITHUB_* org/repo variables.
			policy["blocked-users"] = guardExprSentinel + "${{ steps.parse-guard-vars.outputs.blocked_users }}"
			policy["trusted-users"] = guardExprSentinel + "${{ steps.parse-guard-vars.outputs.trusted_users }}"
			policy["approval-labels"] = guardExprSentinel + "${{ steps.parse-guard-vars.outputs.approval_labels }}"
			return map[string]any{
				"allow-only": policy,
			}
		}
	}
	return nil
}

// DefaultEndorsementReactions are the default endorsement reactions injected when the
// integrity-reactions feature flag is enabled but no explicit endorsement-reactions are set.
var DefaultEndorsementReactions = []string{"THUMBS_UP", "HEART"}

// DefaultDisapprovalReactions are the default disapproval reactions injected when the
// integrity-reactions feature flag is enabled but no explicit disapproval-reactions are set.
var DefaultDisapprovalReactions = []string{"THUMBS_DOWN", "CONFUSED"}

// hasReactionFieldsInToolConfig returns true if any reaction-based integrity fields are
// explicitly set in the raw tool configuration map.
func hasReactionFieldsInToolConfig(toolConfig map[string]any) bool {
	_, hasEndorsement := toolConfig["endorsement-reactions"]
	_, hasDisapproval := toolConfig["disapproval-reactions"]
	_, hasDisapprovalIntegrity := toolConfig["disapproval-integrity"]
	_, hasEndorserMin := toolConfig["endorser-min-integrity"]
	return hasEndorsement || hasDisapproval || hasDisapprovalIntegrity || hasEndorserMin
}

// injectIntegrityReactionFields adds endorsement-reactions, disapproval-reactions,
// disapproval-integrity, and endorser-min-integrity into an existing allow-only policy
// map when the integrity-reactions feature flag is enabled and the MCPG version supports it.
//
// This function is used exclusively for proxy mode (DIFC proxy / CLI proxy). Reaction-based
// integrity is not supported in MCP gateway mode because the GitHub MCP server protocol does
// not expose reaction author information, which is required for integrity decisions.
//
//   - policy is the inner allow-only map (not the outer allow-only wrapper).
//   - toolConfig is the raw github tool configuration map.
//   - data contains workflow data including feature flags used to check if integrity-reactions is enabled.
//   - gatewayConfig contains MCP gateway version configuration used to version-gate the injection.
//
// When the feature flag is enabled and endorsement-reactions or disapproval-reactions are
// not explicitly set in toolConfig, sensible defaults are injected:
//   - endorsement-reactions: ["THUMBS_UP", "HEART"]
//   - disapproval-reactions: ["THUMBS_DOWN", "CONFUSED"]
//
// No-op when the feature flag is disabled or the MCPG version is too old.
func injectIntegrityReactionFields(policy map[string]any, toolConfig map[string]any, data *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig) {
	if !isFeatureEnabled(constants.IntegrityReactionsFeatureFlag, data) {
		return
	}
	if !mcpgSupportsIntegrityReactions(gatewayConfig) {
		return
	}
	if endorsement, ok := toolConfig["endorsement-reactions"]; ok {
		policy["endorsement-reactions"] = endorsement
	} else {
		policy["endorsement-reactions"] = DefaultEndorsementReactions
	}
	if disapproval, ok := toolConfig["disapproval-reactions"]; ok {
		policy["disapproval-reactions"] = disapproval
	} else {
		policy["disapproval-reactions"] = DefaultDisapprovalReactions
	}
	if disapprovalIntegrity, ok := toolConfig["disapproval-integrity"]; ok {
		policy["disapproval-integrity"] = disapprovalIntegrity
	}
	if endorserMinIntegrity, ok := toolConfig["endorser-min-integrity"]; ok {
		policy["endorser-min-integrity"] = endorserMinIntegrity
	}
}

// mcpgSupportsIntegrityReactions returns true when the effective MCPG version supports
// endorsement-reactions and disapproval-reactions in the allow-only policy (>= v0.2.18).
//
// Special cases:
//   - gatewayConfig is nil or has no Version: use DefaultMCPGatewayVersion for comparison.
//   - "latest": always returns true (latest is always a new release).
//   - Any semver string >= MCPGIntegrityReactionsMinVersion: returns true.
//   - Any semver string < MCPGIntegrityReactionsMinVersion: returns false.
//   - Non-semver string (e.g. a branch name): returns false (conservative).
func mcpgSupportsIntegrityReactions(gatewayConfig *MCPGatewayRuntimeConfig) bool {
	var version string
	if gatewayConfig != nil && gatewayConfig.Version != "" {
		version = gatewayConfig.Version
	} else {
		// No override → use the default version for comparison.
		version = string(constants.DefaultMCPGatewayVersion)
	}

	// "latest" means the newest release — always supports the field.
	if strings.EqualFold(version, "latest") {
		return true
	}

	minVersion := string(constants.MCPGIntegrityReactionsMinVersion)
	return semverutil.Compare(version, minVersion) >= 0
}

// deriveSafeOutputsGuardPolicyFromGitHub generates a safeoutputs guard-policy from GitHub guard-policy.
// When the GitHub MCP server has a guard-policy with repos, the safeoutputs MCP must also have
// a linked guard-policy with accept field derived from repos according to these rules:
//
// Rules by repos value:
//   - repos="all" or repos="public": accept=["*"] (allow all safe output operations)
//   - repos=["O/*"]: accept=["private:O"] (owner wildcard → strip wildcard)
//   - repos=["O/P*"]: accept=["private:O/P*"] (prefix wildcard → keep as-is)
//   - repos=["O/R"]: accept=["private:O/R"] (specific repo → keep as-is)
//
// This allows the gateway to read data from the GitHub MCP server and still write to safeoutputs.
// Returns nil if no GitHub guard policies are configured.
func deriveSafeOutputsGuardPolicyFromGitHub(githubTool any) map[string]any {
	githubPolicies := getGitHubGuardPolicies(githubTool)
	if githubPolicies == nil {
		return nil
	}

	// Extract the allow-only policy from GitHub guard policies
	allowOnly, ok := githubPolicies["allow-only"].(map[string]any)
	if !ok || allowOnly == nil {
		return nil
	}

	// Extract repos from the allow-only policy
	repos, hasRepos := allowOnly["repos"]
	if !hasRepos {
		return nil
	}

	// Convert repos to accept list according to the specification
	var acceptList []string

	switch r := repos.(type) {
	case string:
		// Single string value (e.g., "all", "public", or a pattern)
		switch r {
		case "all", "public":
			// For "all" or "public", accept all safe output operations
			acceptList = []string{"*"}
		default:
			// Single pattern - transform according to rules
			acceptList = []string{transformRepoPattern(r)}
		}
	case []any:
		// Array of patterns
		acceptList = make([]string, 0, len(r))
		for _, item := range r {
			if pattern, ok := item.(string); ok {
				acceptList = append(acceptList, transformRepoPattern(pattern))
			}
		}
	case []string:
		// Array of patterns (already strings)
		acceptList = make([]string, 0, len(r))
		for _, pattern := range r {
			acceptList = append(acceptList, transformRepoPattern(pattern))
		}
	default:
		// Unknown type, return nil
		githubConfigLog.Printf("Unknown repos type in guard-policy: %T", repos)
		return nil
	}

	// Build the write-sink policy for safeoutputs
	return map[string]any{
		"write-sink": map[string]any{
			"accept": acceptList,
		},
	}
}

// transformRepoPattern transforms a repos pattern to the corresponding accept pattern.
// Rules:
//   - "O/*"  → "private:O" (owner wildcard → strip wildcard)
//   - "O/P*" → "private:O/P*" (prefix wildcard → keep as-is)
//   - "O/R"  → "private:O/R" (specific repo → keep as-is)
func transformRepoPattern(pattern string) string {
	// Check if pattern ends with "/*" (owner wildcard)
	if owner, found := strings.CutSuffix(pattern, "/*"); found {
		// Strip the wildcard: "owner/*" → "private:owner"
		return "private:" + owner
	}
	// All other patterns (including "O/P*" prefix wildcards): add "private:" prefix
	return "private:" + pattern
}

// deriveWriteSinkGuardPolicyFromWorkflow derives a write-sink guard policy for non-GitHub MCP servers
// from the workflow's GitHub guard-policy configuration. This uses the same derivation as
// deriveSafeOutputsGuardPolicyFromGitHub, ensuring that as guard policies are rolled out, only
// GitHub inputs are filtered while outputs to non-GitHub servers are not restricted.
//
// Two cases produce a non-nil policy:
//  1. Explicit guard policy — when repos/min-integrity are set on the GitHub tool, a write-sink
//     policy is derived from those settings (e.g. "private:myorg/myrepo").
//  2. Auto-lockdown — when the GitHub tool is present without explicit guard policies and without
//     a GitHub App configured, auto-lockdown detection will set repos=all at runtime, so a
//     write-sink policy with accept=["*"] is returned to match that runtime behaviour.
//
// Returns nil when workflowData is nil, when no GitHub tool is present, or when a GitHub App is
// configured (auto-lockdown is skipped for GitHub App tokens, which are already repo-scoped).
func deriveWriteSinkGuardPolicyFromWorkflow(workflowData *WorkflowData) map[string]any {
	if workflowData == nil || workflowData.Tools == nil {
		return nil
	}
	githubTool, hasGitHub := workflowData.Tools["github"]
	if !hasGitHub {
		return nil
	}

	// Try to derive from explicit guard policy first
	policy := deriveSafeOutputsGuardPolicyFromGitHub(githubTool)
	if policy != nil {
		return policy
	}

	// When no explicit guard policy is configured but automatic lockdown detection would run
	// (GitHub tool present and not disabled, no GitHub App configured), return accept=["*"]
	// because automatic lockdown always sets repos=all at runtime.
	if githubTool != false && len(getGitHubGuardPolicies(githubTool)) == 0 && !hasGitHubApp(githubTool) {
		return map[string]any{
			"write-sink": map[string]any{
				"accept": []string{"*"},
			},
		}
	}

	return nil
}

func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := string(constants.DefaultGitHubMCPServerVersion) // Default Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			// Handle different version types
			switch v := versionSetting.(type) {
			case string:
				githubDockerImageVersion = v
			case int:
				githubDockerImageVersion = strconv.Itoa(v)
			case int64:
				githubDockerImageVersion = strconv.FormatInt(v, 10)
			case uint64:
				githubDockerImageVersion = strconv.FormatUint(v, 10)
			case float64:
				// Use %g to avoid trailing zeros and scientific notation for simple numbers
				githubDockerImageVersion = fmt.Sprintf("%g", v)
			}
		}
	}
	githubConfigLog.Printf("GitHub MCP Docker image version: %s", githubDockerImageVersion)
	return githubDockerImageVersion
}
