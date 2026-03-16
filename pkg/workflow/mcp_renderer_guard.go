package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// renderGuardPoliciesJSON renders a "guard-policies" JSON field at the given indent level.
// The policies map contains policy names (e.g., "allow-only") mapped to their configurations.
// Renders as the last field (no trailing comma) with the given base indent.
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

	fmt.Fprintf(yaml, "%s\"guard-policies\": %s\n", indent, string(jsonBytes))
}

// renderGuardPoliciesToml renders a "guard-policies" section in TOML format for a given server.
// The policies map contains policy names (e.g., "write-sink", "allow-only") mapped to their configurations.
func renderGuardPoliciesToml(yaml *strings.Builder, policies map[string]any, serverID string) {
	if len(policies) == 0 {
		return
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + serverID + ".\"guard-policies\"]\n")

	// Iterate over each policy (e.g., "write-sink", "allow-only")
	for policyName, policyConfig := range policies {
		yaml.WriteString("          \n")
		yaml.WriteString("          [mcp_servers." + serverID + ".\"guard-policies\"." + policyName + "]\n")

		// Extract policy fields (e.g., "accept", "repos", "min-integrity")
		if configMap, ok := policyConfig.(map[string]any); ok {
			// Collect and sort keys for deterministic output
			keys := make([]string, 0, len(configMap))
			for k := range configMap {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, fieldName := range keys {
				fieldValue := configMap[fieldName]
				switch v := fieldValue.(type) {
				case string:
					// Handle string values (e.g., repos = "all", min-integrity = "approved")
					yaml.WriteString("          " + fieldName + " = \"" + v + "\"\n")
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
				}
			}
		}
	}
}
