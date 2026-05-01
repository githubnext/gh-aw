package cli

// This file implements an MCP receiving middleware that transforms raw JSON schema
// "additional properties" validation errors into helpful, user-friendly messages
// with "Did you mean?" suggestions.
//
// Background: When the MCP SDK validates tool arguments against the input schema
// (which uses additionalProperties: false), it emits a raw message like:
//
//	validating "arguments": validating root: unexpected additional properties ["workflow-name"]
//
// This is surfaced directly to users, leaking internal validation details without
// any guidance on the correct parameter name.  The middleware here intercepts
// those tool-error results and replaces the message with a helpful alternative.

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolParamEntry holds the valid parameter names for a single MCP tool.
type toolParamEntry = []string

// argumentValidationMiddleware returns a mcp.Middleware that intercepts tool-call
// results containing "unexpected additional properties" validation errors and
// replaces them with a helpful message that names the unknown parameter, suggests
// a close match, and points to the tool's --help output.
//
// toolParams maps tool names to their list of valid JSON parameter names.  It is
// provided by the caller as a hardcoded registry (see mcpToolParams).
func argumentValidationMiddleware(toolParams map[string]toolParamEntry) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			result, err := next(ctx, method, req)
			if err != nil || method != "tools/call" {
				return result, err
			}

			// Check whether the result is a tool error containing a schema
			// "additional properties" validation message.
			toolResult, ok := result.(*mcp.CallToolResult)
			if !ok || !toolResult.IsError {
				return result, err
			}

			// Extract the error text from the first TextContent element.
			if len(toolResult.Content) == 0 {
				return result, err
			}
			textContent, ok := toolResult.Content[0].(*mcp.TextContent)
			if !ok {
				return result, err
			}
			errMsg := textContent.Text

			if !strings.Contains(errMsg, "unexpected additional properties") {
				return result, err
			}

			// Parse the unknown parameter names from the error text.
			unknownParams := extractUnknownParams(errMsg)
			if len(unknownParams) == 0 {
				return result, err
			}

			// Determine the tool name from the request so we can look up valid params.
			toolName := extractMCPToolName(req)
			validParams := toolParams[toolName]

			// Build a helpful replacement message.
			helpMsg := buildHelpfulParamError(toolName, unknownParams, validParams)

			// Return a modified tool result with the helpful message, preserving IsError.
			replaced := *toolResult
			replaced.Content = []mcp.Content{&mcp.TextContent{Text: helpMsg}}
			return &replaced, nil
		}
	}
}

// extractMCPToolName retrieves the tool name from a MCP Request by casting the
// request params to *mcp.CallToolParamsRaw.  Returns an empty string if the cast
// fails.
func extractMCPToolName(req mcp.Request) string {
	if p, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
		return p.Name
	}
	return ""
}

// extractUnknownParams parses the JSON-schema validation error string to extract
// the list of unknown parameter names.
//
// The expected format (from jsonschema-go) is:
//
//	unexpected additional properties ["name1" "name2"]
//
// which uses %q-style quoting for a []string.
var additionalPropsRE = regexp.MustCompile(`unexpected additional properties (.+)$`)
var quotedStringRE = regexp.MustCompile(`"([^"]+)"`)

func extractUnknownParams(errMsg string) []string {
	m := additionalPropsRE.FindStringSubmatch(errMsg)
	if m == nil {
		return nil
	}
	raw := m[1]
	matches := quotedStringRE.FindAllStringSubmatch(raw, -1)
	var params []string
	for _, sm := range matches {
		if sm[1] != "" {
			params = append(params, sm[1])
		}
	}
	return params
}

// buildHelpfulParamError constructs a human-readable error message that:
//   - names each unknown parameter
//   - suggests the closest valid parameter (if a good match is found)
//   - directs the user to the tool's --help output
func buildHelpfulParamError(toolName string, unknownParams []string, validParams []string) string {
	var sb strings.Builder

	for i, param := range unknownParams {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Unknown parameter '%s'.", param))
		if suggestion := findSimilarParam(param, validParams); suggestion != "" {
			sb.WriteString(fmt.Sprintf(" Did you mean '%s'?", suggestion))
		}
	}

	if toolName != "" {
		sb.WriteString(fmt.Sprintf("\nRun 'agenticworkflows %s --help' for usage.", toolName))
	}

	return sb.String()
}

// findSimilarParam returns the valid parameter name most similar to unknown, or
// an empty string if no parameter is close enough.
//
// Similarity is measured by the ratio of the longest-common-prefix length to the
// shorter of the two normalized strings.  A threshold of 0.7 (70%) is required.
//
// Normalization: lowercase, hyphens and underscores removed.
func findSimilarParam(unknown string, validParams []string) string {
	if len(validParams) == 0 {
		return ""
	}

	normUnknown := normalizeParamName(unknown)

	type candidate struct {
		name  string
		score float64
	}
	var best candidate

	for _, p := range validParams {
		normP := normalizeParamName(p)

		// Exact normalized match wins immediately.
		if normP == normUnknown {
			return p
		}

		lcp := longestCommonPrefixLen(normUnknown, normP)
		shorter := min(len(normP), len(normUnknown))
		if shorter == 0 {
			continue
		}
		score := float64(lcp) / float64(shorter)
		if score > best.score {
			best = candidate{name: p, score: score}
		}
	}

	const threshold = 0.7
	if best.score >= threshold {
		return best.name
	}
	return ""
}

// normalizeParamName lowercases name and removes hyphens and underscores, so
// that "workflow-name", "workflow_name", and "workflowname" all compare equal.
func normalizeParamName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}

// longestCommonPrefixLen returns the length of the longest common prefix of a
// and b.
func longestCommonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// mcpToolParams returns the registry of valid parameter names for every tool
// registered in the MCP server.  This is called once during server construction
// and the result is passed to argumentValidationMiddleware.
//
// The map key is the MCP tool name; the value is the sorted list of valid JSON
// parameter names taken from the corresponding *Args struct json tags.
//
// MAINTENANCE: When tool parameters change, update the corresponding entry here
// to match the json tags on the *Args struct in register*Tool functions:
//   - status       → registerStatusTool   in mcp_tools_readonly.go  (statusArgs)
//   - compile      → registerCompileTool  in mcp_tools_readonly.go  (compileArgs)
//   - logs         → registerLogsTool     in mcp_tools_privileged.go (logsArgs)
//   - audit        → registerAuditTool    in mcp_tools_privileged.go (auditArgs)
//   - audit-diff   → registerAuditDiffTool in mcp_tools_privileged.go (auditDiffArgs)
//   - checks       → registerChecksTool   in mcp_tools_readonly.go  (checksArgs)
//   - mcp-inspect  → registerMCPInspectTool in mcp_tools_readonly.go (mcpInspectArgs)
//   - add          → registerAddTool      in mcp_tools_management.go (addArgs)
//   - update       → registerUpdateTool   in mcp_tools_management.go (updateArgs)
//   - fix          → registerFixTool      in mcp_tools_management.go (fixArgs)
func mcpToolParams() map[string]toolParamEntry {
	params := map[string]toolParamEntry{
		// statusArgs in mcp_tools_readonly.go
		"status": {
			"pattern",
		},
		// compileArgs in mcp_tools_readonly.go
		"compile": {
			"workflows", "strict", "zizmor", "poutine", "actionlint",
			"runner-guard", "fix", "max_tokens",
		},
		// logsArgs in mcp_tools_privileged.go
		"logs": {
			"workflow_name", "count", "start_date", "end_date", "engine",
			"firewall", "no_firewall", "filtered_integrity", "branch",
			"after_run_id", "before_run_id", "timeout", "max_tokens", "artifacts",
		},
		// auditArgs in mcp_tools_privileged.go
		"audit": {
			"run_id_or_url", "run_ids_or_urls", "artifacts", "max_tokens",
		},
		// auditDiffArgs in mcp_tools_privileged.go
		"audit-diff": {
			"base_run_id", "compare_run_ids", "artifacts",
		},
		// checksArgs in mcp_tools_readonly.go
		"checks": {
			"pr_number", "repo",
		},
		// mcpInspectArgs in mcp_tools_readonly.go
		"mcp-inspect": {
			"workflow_file", "server", "tool",
		},
		// addArgs in mcp_tools_management.go
		"add": {
			"workflows", "number", "name",
		},
		// updateArgs in mcp_tools_management.go
		"update": {
			"workflows", "major", "force",
		},
		// fixArgs in mcp_tools_management.go
		"fix": {
			"workflows", "write", "list_codemods",
		},
	}

	// Sort each list for deterministic output in suggestions.
	for k, v := range params {
		sorted := make([]string, len(v))
		copy(sorted, v)
		sort.Strings(sorted)
		params[k] = sorted
	}

	return params
}
