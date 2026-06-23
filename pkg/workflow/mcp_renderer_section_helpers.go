package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

func writeJSONStringMapEntries(yaml *strings.Builder, values map[string]string, indent string) {
	for i, key := range sortedMapKeys(values) {
		comma := ","
		if i == len(values)-1 {
			comma = ""
		}
		fmt.Fprintf(yaml, "%s%s: %s%s\n", indent, mustMarshalJSONString(key), mustMarshalJSONString(values[key]), comma)
	}
}

func mustMarshalJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "\"\""
	}
	return string(encoded)
}

func writeJSONStringMapSection(yaml *strings.Builder, indent, name string, values map[string]string, trailingComma bool) {
	fmt.Fprintf(yaml, "%s\"%s\": {\n", indent, name)
	writeJSONStringMapEntries(yaml, values, indent+"  ")
	if trailingComma {
		fmt.Fprintf(yaml, "%s},\n", indent)
		return
	}
	fmt.Fprintf(yaml, "%s}\n", indent)
}

func writeTOMLInlineStringMapSection(yaml *strings.Builder, indent, name string, values map[string]string) {
	fmt.Fprintf(yaml, "%s%s = { ", indent, name)
	for i, key := range sortedMapKeys(values) {
		if i > 0 {
			yaml.WriteString(", ")
		}
		fmt.Fprintf(yaml, "\"%s\" = \"%s\"", key, values[key])
	}
	yaml.WriteString(" }\n")
}

// buildGitHubMCPEnvVars builds the common GitHub MCP environment map used by
// local, remote, and TOML renderers.
//
// hostValue should be a full URL (for example https://hostname, with no
// trailing slash) because github-mcp-server expects GITHUB_HOST in the same
// format that GitHub Actions exposes via GITHUB_SERVER_URL (for example
// https://github.com or https://myorg.ghe.com).
func buildGitHubMCPEnvVars(tokenValue, hostValue string, readOnly, lockdown bool, toolsets string) map[string]string {
	envVars := map[string]string{
		"GITHUB_PERSONAL_ACCESS_TOKEN": tokenValue,
		"GITHUB_HOST":                  hostValue,
	}

	if readOnly {
		envVars["GITHUB_READ_ONLY"] = "1"
	}

	if lockdown {
		envVars["GITHUB_LOCKDOWN_MODE"] = "1"
	}

	if toolsets != "" {
		envVars["GITHUB_TOOLSETS"] = toolsets
	}

	return envVars
}

func buildGitHubMCPRemoteHeaders(authValue string, readOnly, lockdown bool, toolsets string) map[string]string {
	headers := map[string]string{
		"Authorization": authValue,
	}

	if readOnly {
		headers["X-MCP-Readonly"] = "true"
	}

	if lockdown {
		headers["X-MCP-Lockdown"] = "true"
	}

	if toolsets != "" {
		headers["X-MCP-Toolsets"] = toolsets
	}

	return headers
}

func hasGitHubMCPGuardPolicies(guardPolicies map[string]any, guardPoliciesFromStep bool) bool {
	return len(guardPolicies) > 0 || guardPoliciesFromStep
}

func renderGitHubMCPGuardPolicies(yaml *strings.Builder, guardPolicies map[string]any, guardPoliciesFromStep bool, indent string) {
	if guardPoliciesFromStep {
		renderGuardPoliciesJSON(yaml, map[string]any{
			"allow-only": map[string]any{
				"min-integrity": "$GITHUB_MCP_GUARD_MIN_INTEGRITY",
				"repos":         "$GITHUB_MCP_GUARD_REPOS",
			},
		}, indent)
		return
	}

	if len(guardPolicies) > 0 {
		renderGuardPoliciesJSON(yaml, guardPolicies, indent)
	}
}
