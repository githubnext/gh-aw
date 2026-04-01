package workflow

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var fetchLog = logger.New("workflow:fetch")

// AddMCPFetchServerIfNeeded adds the mcp/fetch dockerized MCP server to the tools configuration
// if the engine doesn't have built-in web-fetch support and web-fetch tool is requested
func AddMCPFetchServerIfNeeded(tools map[string]any, engine CodingAgentEngine) (map[string]any, []string) {
	// Check if web-fetch tool is requested
	if _, hasWebFetch := tools["web-fetch"]; !hasWebFetch {
		fetchLog.Print("web-fetch tool not requested, skipping MCP fetch server")
		return tools, nil
	}

	// If the engine already supports web-fetch, no need to add MCP server
	if engine.SupportsWebFetch() {
		fetchLog.Print("Engine has built-in web-fetch support, skipping MCP fetch server")
		return tools, nil
	}

	fetchLog.Print("Adding MCP fetch server for web-fetch tool")

	// Create a copy of the tools map to avoid modifying the original
	updatedTools := make(map[string]any)
	maps.Copy(updatedTools, tools)

	// Remove the web-fetch tool since we'll replace it with an MCP server
	delete(updatedTools, "web-fetch")

	// Add the web-fetch server configuration
	// Note: The "container" key marks this as an MCP server with stdio transport.
	// The actual rendering is done by renderMCPFetchServerConfig() which uses
	// the standardized Docker command format for all engines.
	webFetchConfig := map[string]any{
		"container": "mcp/fetch",
	}

	// Add the web-fetch server to the tools
	updatedTools["web-fetch"] = webFetchConfig

	fetchLog.Print("Successfully added web-fetch MCP server configuration")

	// Return the updated tools and the list of added MCP servers
	return updatedTools, []string{"web-fetch"}
}

// computeWebFetchAllowedDomains returns the comma-separated allowed-domains list to enforce
// inside the mcp/fetch container. This mirrors the same allowlist AWF enforces for the agent
// container, closing the gap where mcp/fetch could otherwise reach non-allowlisted URLs.
//
// Returns empty string (no restriction) when:
//   - workflowData is nil or has no network permissions
//   - network.allowed contains "*" (unrestricted access)
//   - The firewall is not enabled for this workflow
//
// When a non-wildcard allowlist is present and the firewall is active the function returns
// the same comma-separated domain string that would be passed to AWF's --allow-domains flag,
// ensuring both the agent container and the mcp/fetch container enforce the same policy.
func computeWebFetchAllowedDomains(workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.NetworkPermissions == nil {
		return ""
	}

	// Wildcard = unrestricted access; do not restrict mcp/fetch either.
	if slices.Contains(workflowData.NetworkPermissions.Allowed, "*") {
		fetchLog.Print("Wildcard '*' in network.allowed, skipping mcp/fetch allowed-domains restriction")
		return ""
	}

	// Only restrict mcp/fetch when the AWF firewall is active.
	// Without the firewall, network restrictions are not enforced for the agent either.
	if !isFirewallEnabled(workflowData) {
		fetchLog.Print("Firewall not enabled, skipping mcp/fetch allowed-domains restriction")
		return ""
	}

	// Compute the allowed-domains string that matches what AWF uses for this engine.
	var engineID string
	if workflowData.EngineConfig != nil {
		engineID = workflowData.EngineConfig.ID
	} else if workflowData.AI != "" {
		engineID = workflowData.AI
	}

	allowedDomains := GetAllowedDomainsForEngine(
		constants.EngineName(engineID),
		workflowData.NetworkPermissions,
		workflowData.Tools,
		workflowData.Runtimes,
	)
	fetchLog.Printf("Computed mcp/fetch allowed-domains: %d chars", len(allowedDomains))
	return allowedDomains
}

// renderMCPFetchServerConfig renders the MCP fetch server configuration
// This is a shared function that can be used by all engines
// includeTools parameter adds "tools": ["*"] field for engines that require it (e.g., Copilot)
// guardPolicies parameter adds write-sink guard policies when derived from the GitHub guard-policy
// allowedDomains is a comma-separated list of domains to pass as --allowed-domains to mcp-server-fetch,
// enforcing the same allowlist inside the container that AWF enforces at the network level.
// Pass empty string to skip domain restriction (e.g., when allowed: ["*"] or firewall is disabled).
func renderMCPFetchServerConfig(yaml *strings.Builder, format string, indent string, isLast bool, includeTools bool, guardPolicies map[string]any, allowedDomains string) {
	fetchLog.Printf("Rendering MCP fetch server config: format=%s, includeTools=%v, hasAllowedDomains=%v", format, includeTools, allowedDomains != "")

	// Build entrypointArgs from the allowed-domains list.
	// mcp-server-fetch accepts: --allowed-domains domain1 domain2 domain3 ...
	var entrypointArgs []string
	if allowedDomains != "" {
		entrypointArgs = append(entrypointArgs, "--allowed-domains")
		for d := range strings.SplitSeq(allowedDomains, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				entrypointArgs = append(entrypointArgs, d)
			}
		}
	}

	switch format {
	case "json":
		// JSON format (for Claude, Copilot, Custom engines)
		// Use container key per MCP Gateway schema (container-based stdio server)
		yaml.WriteString(indent + "\"web-fetch\": {\n")

		hasEntrypointArgs := len(entrypointArgs) > 0
		hasGuardPolicies := len(guardPolicies) > 0

		// Determine comma placement: container is always first; more fields may follow.
		containerHasSuccessor := hasEntrypointArgs || hasGuardPolicies
		if containerHasSuccessor {
			yaml.WriteString(indent + "  \"container\": \"mcp/fetch\",\n")
		} else {
			yaml.WriteString(indent + "  \"container\": \"mcp/fetch\"\n")
		}

		if hasEntrypointArgs {
			// Render entrypointArgs with proper JSON escaping for each value.
			// Trailing comma only when guardPolicies follow.
			yaml.WriteString(indent + "  \"entrypointArgs\": [\n")
			for i, arg := range entrypointArgs {
				// json.Marshal produces a properly-escaped JSON string literal (including quotes).
				quotedArg, _ := json.Marshal(arg)
				yaml.WriteString(indent + "    " + string(quotedArg))
				if i < len(entrypointArgs)-1 {
					yaml.WriteString(",")
				}
				yaml.WriteString("\n")
			}
			if hasGuardPolicies {
				yaml.WriteString(indent + "  ],\n")
			} else {
				yaml.WriteString(indent + "  ]\n")
			}
		}

		if hasGuardPolicies {
			renderGuardPoliciesJSON(yaml, guardPolicies, indent+"  ")
		}

		if isLast {
			yaml.WriteString(indent + "}\n")
		} else {
			yaml.WriteString(indent + "},\n")
		}
	case "toml":
		// TOML format (for Codex engine)
		// Use container key per MCP Gateway schema (container-based stdio server)
		yaml.WriteString(indent + "\n")
		yaml.WriteString(indent + "[mcp_servers.\"web-fetch\"]\n")
		yaml.WriteString(indent + "container = \"mcp/fetch\"\n")
		// Render entrypointArgs as an inline TOML array.
		// Use json.Marshal for each value: TOML basic strings share JSON's escape sequences
		// for the characters that domain names could theoretically contain.
		if len(entrypointArgs) > 0 {
			yaml.WriteString(indent + "entrypointArgs = [")
			for i, arg := range entrypointArgs {
				if i > 0 {
					yaml.WriteString(", ")
				}
				quotedArg, _ := json.Marshal(arg)
				yaml.Write(quotedArg) //nolint:errcheck // strings.Builder.Write never returns an error
			}
			yaml.WriteString("]\n")
		}
		// Add guard policies as a separate TOML section if configured
		if len(guardPolicies) > 0 {
			renderGuardPoliciesToml(yaml, guardPolicies, "web-fetch")
		}
	}
}
