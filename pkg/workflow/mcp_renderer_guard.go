package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// guardExprSentinel is a prefix that marks a string value in the guard-policies map as a
// raw GitHub Actions expression that should be emitted verbatim (without surrounding JSON
// string quotes) in the final output.
//
// Background: json.MarshalIndent cannot emit non-JSON content verbatim (it validates
// json.RawMessage content), so we use a sentinel string that json.MarshalIndent can safely
// encode as part of a regular JSON string, then post-process the output to un-quote those
// values. Paired with toJSON() in the expression, this ensures the variable value is
// properly JSON-encoded at runtime even if it contains double quotes or backslashes.
const guardExprSentinel = "__GH_AW_GUARD_EXPR:"

// guardExprRE matches sentinel-prefixed expression values in the JSON output:
//
//	"__GH_AW_GUARD_EXPR:${{ expr }}"  →  ${{ expr }}
//	"__GH_AW_GUARD_EXPR:${ENV_VAR}"   →  ${ENV_VAR}
//
// Expressions are of the form ${{ ... }} (GitHub Actions expressions) or ${VAR}
// (shell environment variable references used for template injection prevention).
// They must not contain double quotes (our generated expressions use single-quoted
// strings inside the GitHub Actions expression or simple env var names,
// so this invariant holds for all compiler-generated values).
var guardExprRE = regexp.MustCompile(`"` + regexp.QuoteMeta(guardExprSentinel) + `(\$\{\{[^"]+\}\}|\$\{[A-Za-z_][A-Za-z0-9_]*\})"`)

// renderGuardPoliciesJSON renders a "guard-policies" JSON field at the given indent level.
// The policies map contains policy names (e.g., "allow-only") mapped to their configurations.
// Renders as the last field (no trailing comma) with the given base indent.
//
// Any string value that starts with guardExprSentinel is treated as a raw GitHub Actions
// expression. After json.MarshalIndent, those sentinel-prefixed strings are replaced with
// the un-quoted expression so that toJSON() can properly encode the value at runtime.
func renderGuardPoliciesJSON(yaml *strings.Builder, policies map[string]any, indent string) {
	if len(policies) == 0 {
		return
	}

	// Marshal to JSON with indentation, then re-indent to match the current indent level
	jsonBytes, err := json.MarshalIndent(policies, indent, "  ")
	if err != nil {
		mcpRendererLog.Printf("Failed to marshal guard-policies: %v", err)
		return
	}

	// Un-quote sentinel-prefixed expression values so they are emitted as raw
	// expressions (either GitHub Actions ${{ }} or shell ${VAR} references).
	// For example:
	//   Before: "blocked-users": "__GH_AW_GUARD_EXPR:${GH_AW_GUARD_BLOCKED_USERS}"
	//   After:  "blocked-users": ${GH_AW_GUARD_BLOCKED_USERS}
	// At runtime, bash expands the ${VAR} reference from the step env block.
	output := guardExprRE.ReplaceAllString(string(jsonBytes), `$1`)

	fmt.Fprintf(yaml, "%s\"guard-policies\": %s\n", indent, output)
}

// renderGuardPoliciesToml renders a "guard-policies" section in TOML format for a given server.
// The policies map contains policy names (e.g., "write-sink") mapped to their configurations.
func renderGuardPoliciesToml(yaml *strings.Builder, policies map[string]any, serverID string) {
	if len(policies) == 0 {
		return
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + serverID + ".\"guard-policies\"]\n")

	// Iterate over each policy (e.g., "write-sink")
	for policyName, policyConfig := range policies {
		yaml.WriteString("          \n")
		yaml.WriteString("          [mcp_servers." + serverID + ".\"guard-policies\"." + policyName + "]\n")

		// Extract policy fields (e.g., "accept")
		if configMap, ok := policyConfig.(map[string]any); ok {
			for fieldName, fieldValue := range configMap {
				// Handle array values (e.g., accept = ["private:github/gh-aw*"])
				if arrayValue, ok := fieldValue.([]string); ok {
					yaml.WriteString("          " + fieldName + " = [")
					for i, item := range arrayValue {
						if i > 0 {
							yaml.WriteString(", ")
						}
						yaml.WriteString("\"" + item + "\"")
					}
					yaml.WriteString("]\n")
				}
			}
		}
	}
}
