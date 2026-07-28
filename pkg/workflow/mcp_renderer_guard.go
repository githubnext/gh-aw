package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
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

// sinkVisibilityEnvVar is the environment variable name that holds the repository visibility
// at workflow runtime. It is set on the Start MCP Gateway step's env: block from the
// determine-automatic-lockdown step output, ensuring no ${{ }} expression appears inside
// the run: heredoc (which would be flagged as template injection by zizmor).
const sinkVisibilityEnvVar = "GH_AW_SINK_VISIBILITY"

// sinkVisibilityRuntimeExpr resolves repository visibility at workflow runtime for MCP guard policies.
// Uses a plain shell variable reference so the heredoc expands it to a bare string (e.g. "public"),
// while the surrounding JSON double-quotes produce a valid JSON string value.
const sinkVisibilityRuntimeExpr = "${" + sinkVisibilityEnvVar + "}"

// guardExprRE matches sentinel-prefixed expression values in the JSON output:
//
//	"__GH_AW_GUARD_EXPR:${{ expr }}"  →  ${{ expr }}
//
// Expressions are always of the form ${{ ... }} and must not contain double quotes
// (our generated expressions use single-quoted strings inside the GitHub Actions expression,
// so this invariant holds for all compiler-generated fallback values).
var guardExprRE = regexp.MustCompile(`"` + regexp.QuoteMeta(guardExprSentinel) + `(\$\{\{[^"]+\}\})"`)

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

	// Un-quote sentinel-prefixed expression values so they are emitted as raw GitHub Actions
	// expressions. For example:
	//   Before: "blocked-users": "__GH_AW_GUARD_EXPR:${{ toJSON(vars.X || '') }}"
	//   After:  "blocked-users": ${{ toJSON(vars.X || '') }}
	// At runtime, GitHub Actions evaluates toJSON() which properly JSON-encodes the value.
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

	// Iterate over each policy (e.g., "write-sink") in sorted order for deterministic output
	policyNames := make([]string, 0, len(policies))
	for policyName := range policies {
		policyNames = append(policyNames, policyName)
	}
	slices.Sort(policyNames)
	for _, policyName := range policyNames {
		policyConfig := policies[policyName]
		yaml.WriteString("          \n")
		yaml.WriteString("          [mcp_servers." + serverID + ".\"guard-policies\"." + policyName + "]\n")

		// Extract policy fields (e.g., "accept", "sink-visibility") in sorted order
		if configMap, ok := policyConfig.(map[string]any); ok {
			fieldNames := make([]string, 0, len(configMap))
			for fieldName := range configMap {
				fieldNames = append(fieldNames, fieldName)
			}
			slices.Sort(fieldNames)
			for _, fieldName := range fieldNames {
				fieldValue := configMap[fieldName]
				switch v := fieldValue.(type) {
				case []string:
					// Handle array values (e.g., accept = ["private:github/gh-aw*"])
					yaml.WriteString("          " + fieldName + " = [")
					for i, item := range v {
						if i > 0 {
							yaml.WriteString(", ")
						}
						yaml.WriteString("\"" + item + "\"")
					}
					yaml.WriteString("]\n")
				case string:
					// Handle string values (e.g., sink-visibility = "${GH_AW_SINK_VISIBILITY}").
					// Shell variable references like ${VAR} are expanded by bash from the unquoted
					// heredoc before TOML parsing, producing a valid TOML string at runtime.
					yaml.WriteString("          " + fieldName + " = \"" + v + "\"\n")
				}
			}
		}
	}
}
