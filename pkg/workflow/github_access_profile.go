package workflow

// github_access_profile.go defines the single, typed resolver for a workflow's
// effective GitHub access profile.
//
// The author-facing selector is tools.github.mode, a homogeneous enum:
//
//	cli         - GitHub is reached through the pre-authenticated gh CLI, protected
//	              by the host policy proxy. No GitHub MCP server is registered and
//	              the token is never exposed to the agent container.
//	mcp-local   - GitHub is reached through a local Docker GitHub MCP server.
//	mcp-remote  - GitHub is reached through the hosted GitHub MCP service.
//
// Every compiler decision that used to branch on ad-hoc, independently-computed
// predicates (mode string parsing, features.cli-proxy fallback, MCP transport
// lookup, host-proxy gating) now derives from a single resolveGitHubAccessProfile
// result so the semantics can never drift apart.
//
// Legacy values are still accepted for backward compatibility and normalized:
//
//	gh-proxy, cli            -> cli
//	local  (mode or type)    -> mcp-local
//	remote (mode or type)    -> mcp-remote
//
// The `gh aw fix` codemods rewrite legacy frontmatter to the homogeneous enum.
//
// Note: tools.github.mode is distinct from tools.mcp-mode (and its legacy
// tools.cli-proxy boolean), which mounts all user-facing MCP servers as CLI
// wrappers on PATH and never influences the GitHub access mode resolved here.
// The two are orthogonal: tools.mcp-mode: cli applies to every configured MCP
// server, while tools.github.mode governs GitHub access specifically.

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// GitHubAccessMode is the resolved, author-facing GitHub access mode.
type GitHubAccessMode string

const (
	// GitHubAccessModeCLI reaches GitHub through the pre-authenticated gh CLI,
	// protected by the host policy proxy. No GitHub MCP server is registered.
	GitHubAccessModeCLI GitHubAccessMode = "cli"
	// GitHubAccessModeMCPLocal reaches GitHub through a local Docker GitHub MCP server.
	GitHubAccessModeMCPLocal GitHubAccessMode = "mcp-local"
	// GitHubAccessModeMCPRemote reaches GitHub through the hosted GitHub MCP service.
	GitHubAccessModeMCPRemote GitHubAccessMode = "mcp-remote"
)

// GitHubAccessProfile is the single resolved GitHub access profile for a workflow.
// It is produced by resolveGitHubAccessProfile and consumed everywhere a compiler
// decision depends on how the agent reaches GitHub.
type GitHubAccessProfile struct {
	// Mode is the effective access mode.
	Mode GitHubAccessMode
	// Explicit reports whether the author explicitly selected the mode via
	// tools.github.mode or tools.github.type (as opposed to a compiler default
	// or a legacy feature flag).
	Explicit bool
	// HasGitHubTool reports whether the workflow configures the GitHub tool at all.
	HasGitHubTool bool
}

// IsCLI reports whether GitHub is reached through the pre-authenticated gh CLI
// (protected by the host policy proxy) rather than a GitHub MCP server.
func (p GitHubAccessProfile) IsCLI() bool {
	return p.Mode == GitHubAccessModeCLI
}

// IsMCP reports whether GitHub is reached through a GitHub MCP server
// (local Docker or hosted remote).
func (p GitHubAccessProfile) IsMCP() bool {
	return p.Mode == GitHubAccessModeMCPLocal || p.Mode == GitHubAccessModeMCPRemote
}

// MCPTransport returns the GitHub MCP transport (local or remote) for MCP access
// modes. It is only meaningful when IsMCP is true; for CLI mode it returns the
// local default (no MCP server is registered).
func (p GitHubAccessProfile) MCPTransport() GitHubMCPMode {
	if p.Mode == GitHubAccessModeMCPRemote {
		return GitHubMCPModeRemote
	}
	return GitHubMCPModeLocal
}

// normalizeGitHubAccessMode maps an author-facing or legacy tools.github.mode
// value to a canonical GitHubAccessMode. It returns false for unrecognized values.
func normalizeGitHubAccessMode(value string) (GitHubAccessMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(GitHubAccessModeCLI), string(GitHubMCPModeGHProxy):
		return GitHubAccessModeCLI, true
	case string(GitHubAccessModeMCPLocal), string(GitHubMCPModeLocal):
		return GitHubAccessModeMCPLocal, true
	case string(GitHubAccessModeMCPRemote), string(GitHubMCPModeRemote):
		return GitHubAccessModeMCPRemote, true
	default:
		return "", false
	}
}

// normalizeGitHubTransportValue maps a tools.github.type (or legacy mode)
// transport value to an MCP access mode. Only local/remote (and their mcp-
// prefixed spellings) are valid transports.
func normalizeGitHubTransportValue(value string) (GitHubAccessMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(GitHubAccessModeMCPLocal), string(GitHubMCPModeLocal):
		return GitHubAccessModeMCPLocal, true
	case string(GitHubAccessModeMCPRemote), string(GitHubMCPModeRemote):
		return GitHubAccessModeMCPRemote, true
	default:
		return "", false
	}
}

// explicitGitHubAccessMode resolves the author's mode selection from a GitHub
// tool map. The canonical mode field takes precedence over the deprecated type
// field so every consumer agrees while partially migrated configurations remain
// accepted.
func explicitGitHubAccessMode(githubTool map[string]any) (GitHubAccessMode, bool) {
	if modeValue, exists := githubTool["mode"]; exists {
		if value, ok := modeValue.(string); ok {
			if mode, valid := normalizeGitHubAccessMode(value); valid {
				return mode, true
			}
		}
	}
	if typeValue, exists := githubTool["type"]; exists {
		if value, ok := typeValue.(string); ok {
			if mode, valid := normalizeGitHubTransportValue(value); valid {
				return mode, true
			}
		}
	}
	return "", false
}

// mcpOnlyGitHubFields lists the GitHub tool fields that only apply to the MCP
// access modes (they configure the GitHub MCP server itself). Using them with an
// explicit CLI mode is a validation error.
var mcpOnlyGitHubFields = []string{"toolsets", "allowed", "version", "args"}

// resolveGitHubAccessProfile computes the single effective GitHub access profile
// for a workflow. It is the one place mode/type/features/engine/runtime signals
// are reconciled; all other compiler decisions read the result.
//
// Resolution order (first match wins):
//  1. Explicit tools.github.mode / tools.github.type selected by the author. For
//     engines without MCP support (e.g. pi), enforceMCPProxyTools normalizes the
//     tools to an explicit cli mode before compilation, so those engines resolve
//     to cli here via this rule.
//  2. features.integrity-reactions enabled -> cli (reaction author identification
//     requires the host policy proxy; explicit MCP is rejected in validation).
//  3. Legacy features.cli-proxy: true -> cli (backward compatibility). This is a
//     GitHub-specific flag (it starts the GitHub-token-holding host proxy) and is
//     orthogonal to tools.mcp-mode: cli / legacy tools.cli-proxy, which mount all
//     user-facing MCP servers as CLI wrappers and never resolve GitHub access here.
//  4. No GitHub tool configured -> no access (mode carries no CLI semantics).
//  5. Otherwise (omitted mode) -> mcp-local, preserving existing behavior. `cli` is
//     the recommended homogeneous default surfaced by new-workflow scaffolds and docs,
//     but omitting the mode never silently changes an existing workflow's behavior.
func resolveGitHubAccessProfile(data *WorkflowData) GitHubAccessProfile {
	if data == nil {
		return GitHubAccessProfile{Mode: GitHubAccessModeMCPLocal}
	}

	githubToolValue, hasGitHubKey := data.Tools["github"]
	githubDisabled := hasGitHubKey && githubToolValue == false
	hasGitHub := hasGitHubKey && !githubDisabled
	githubTool, _ := githubToolValue.(map[string]any)

	// 1. Explicit author selection via mode, then legacy type.
	if hasGitHub && githubTool != nil {
		if mode, explicit := explicitGitHubAccessMode(githubTool); explicit {
			return GitHubAccessProfile{Mode: mode, Explicit: true, HasGitHubTool: true}
		}
	}

	// 2. integrity-reactions requires the host policy proxy: resolve to cli.
	if isFeatureEnabled(constants.IntegrityReactionsFeatureFlag, data) {
		return GitHubAccessProfile{Mode: GitHubAccessModeCLI, HasGitHubTool: hasGitHub}
	}

	// 3. Legacy features.cli-proxy: true selects cli. GitHub-specific; unrelated to
	// tools.mcp-mode: cli (which applies to all MCP servers, not just GitHub).
	if isFeatureEnabled(constants.CliProxyFeatureFlag, data) {
		return GitHubAccessProfile{Mode: GitHubAccessModeCLI, HasGitHubTool: hasGitHub}
	}

	// 4. No GitHub tool: there is no GitHub access to route.
	if !hasGitHub {
		return GitHubAccessProfile{Mode: GitHubAccessModeMCPLocal, HasGitHubTool: false}
	}

	// 5. Omitted mode resolves to mcp-local for backward compatibility.
	return GitHubAccessProfile{Mode: GitHubAccessModeMCPLocal, HasGitHubTool: true}
}
