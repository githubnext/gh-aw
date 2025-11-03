package cli

import (
	"fmt"
	"sort"
	"strings"
)

// generateMermaidGraph generates a Mermaid flowchart visualization of taint flows
func generateMermaidGraph(analysis *TaintAnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("```mermaid\n")
	sb.WriteString("flowchart TD\n")
	sb.WriteString("    %% Taint Flow Analysis\n\n")

	// Create nodes for sources
	sb.WriteString("    %% Taint Sources\n")
	sourceNodes := make(map[string]string)
	for i, source := range analysis.Sources {
		nodeID := fmt.Sprintf("SRC%d", i)
		sourceNodes[source.Name] = nodeID
		
		style := getSourceStyle(source.Type)
		label := formatNodeLabel(source.Type, source.Name)
		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]%s\n", nodeID, label, style))
	}
	sb.WriteString("\n")

	// Create nodes for sinks
	sb.WriteString("    %% Taint Sinks\n")
	sinkNodes := make(map[string]string)
	for i, sink := range analysis.Sinks {
		nodeID := fmt.Sprintf("SNK%d", i)
		sinkNodes[sink.Name] = nodeID
		
		style := getSinkStyle(sink.Type)
		label := formatNodeLabel(sink.Type, sink.Name)
		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]%s\n", nodeID, label, style))
	}
	sb.WriteString("\n")

	// Create edges for paths
	sb.WriteString("    %% Taint Flows\n")
	for _, path := range analysis.Paths {
		srcNode := sourceNodes[path.Source.Name]
		sinkNode := sinkNodes[path.Sink.Name]
		
		if srcNode != "" && sinkNode != "" {
			edgeStyle := getEdgeStyle(path.IsUnsafe)
			sb.WriteString(fmt.Sprintf("    %s -->%s %s\n", srcNode, edgeStyle, sinkNode))
		}
	}

	// Add legend
	sb.WriteString("\n    %% Legend\n")
	sb.WriteString("    classDef unsafe fill:#ff6b6b,stroke:#c92a2a,stroke-width:3px,color:#fff\n")
	sb.WriteString("    classDef safe fill:#51cf66,stroke:#2f9e44,stroke-width:2px,color:#000\n")
	sb.WriteString("    classDef medium fill:#ffd43b,stroke:#fab005,stroke-width:2px,color:#000\n")
	sb.WriteString("    classDef critical fill:#c92a2a,stroke:#862e2e,stroke-width:4px,color:#fff\n")

	sb.WriteString("```\n")

	return sb.String()
}

// generateFullReport generates a complete analysis report with Mermaid graph
func generateFullReport(analysis *TaintAnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("# Taint Flow Analysis Report\n\n")

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Workflows Analyzed**: %d\n", analysis.WorkflowCount))
	sb.WriteString(fmt.Sprintf("- **Taint Sources**: %d\n", len(analysis.Sources)))
	sb.WriteString(fmt.Sprintf("- **Taint Sinks**: %d\n", len(analysis.Sinks)))
	sb.WriteString(fmt.Sprintf("- **Total Paths**: %d\n", len(analysis.Paths)))
	sb.WriteString(fmt.Sprintf("- **Unsafe Paths**: %d\n\n", len(analysis.UnsafePaths)))

	if len(analysis.UnsafePaths) > 0 {
		sb.WriteString("⚠️ **WARNING**: Unsafe taint flow paths detected!\n\n")
	} else {
		sb.WriteString("✅ **No unsafe taint flow paths detected**\n\n")
	}

	// Taint Sources Section
	sb.WriteString("## Taint Sources\n\n")
	if len(analysis.Sources) == 0 {
		sb.WriteString("No taint sources identified.\n\n")
	} else {
		// Group sources by type
		sourcesByType := groupSourcesByType(analysis.Sources)
		for sourceType, sources := range sourcesByType {
			sb.WriteString(fmt.Sprintf("### %s (%d)\n\n", formatSourceType(sourceType), len(sources)))
			for _, source := range sources {
				sb.WriteString(fmt.Sprintf("- **%s** - `%s` (Severity: %s)\n", 
					source.Name, source.WorkflowFile, source.Severity))
			}
			sb.WriteString("\n")
		}
	}

	// Taint Sinks Section
	sb.WriteString("## Taint Sinks\n\n")
	if len(analysis.Sinks) == 0 {
		sb.WriteString("No taint sinks identified.\n\n")
	} else {
		// Group sinks by type
		sinksByType := groupSinksByType(analysis.Sinks)
		for sinkType, sinks := range sinksByType {
			sb.WriteString(fmt.Sprintf("### %s (%d)\n\n", formatSinkType(sinkType), len(sinks)))
			for _, sink := range sinks {
				sb.WriteString(fmt.Sprintf("- **%s** - `%s` (Severity: %s)\n", 
					sink.Name, sink.WorkflowFile, sink.Severity))
			}
			sb.WriteString("\n")
		}
	}

	// Unsafe Paths Section
	if len(analysis.UnsafePaths) > 0 {
		sb.WriteString("## ⚠️ Unsafe Taint Paths\n\n")
		sb.WriteString("The following paths represent potentially unsafe data flows that require attention:\n\n")
		
		for i, path := range analysis.UnsafePaths {
			sb.WriteString(fmt.Sprintf("### Path %d: %s → %s\n\n", i+1, 
				formatSourceType(path.Source.Type), formatSinkType(path.Sink.Type)))
			sb.WriteString(fmt.Sprintf("- **Source**: %s (`%s`)\n", path.Source.Name, path.Source.WorkflowFile))
			sb.WriteString(fmt.Sprintf("- **Sink**: %s (`%s`)\n", path.Sink.Name, path.Sink.WorkflowFile))
			sb.WriteString(fmt.Sprintf("- **Risk**: %s\n", path.Reason))
			sb.WriteString("\n**Recommendation**: ")
			sb.WriteString(getRecommendation(path))
			sb.WriteString("\n\n")
		}
	}

	// Safe Paths Section (optional, only show if there are any)
	safePaths := 0
	for _, path := range analysis.Paths {
		if !path.IsUnsafe {
			safePaths++
		}
	}
	
	if safePaths > 0 {
		sb.WriteString(fmt.Sprintf("## ✅ Safe Taint Paths (%d)\n\n", safePaths))
		sb.WriteString("The following paths have appropriate safeguards:\n\n")
		
		for i, path := range analysis.Paths {
			if !path.IsUnsafe {
				sb.WriteString(fmt.Sprintf("%d. %s → %s (in `%s`)\n", i+1, 
					formatSourceType(path.Source.Type), 
					formatSinkType(path.Sink.Type),
					path.Source.WorkflowFile))
			}
		}
		sb.WriteString("\n")
	}

	// Visualization Section
	sb.WriteString("## Taint Flow Visualization\n\n")
	sb.WriteString(generateMermaidGraph(analysis))
	sb.WriteString("\n")

	// Recommendations Section
	sb.WriteString("## General Recommendations\n\n")
	sb.WriteString("1. **Input Validation**: Always validate and sanitize external inputs before use\n")
	sb.WriteString("2. **Safe Outputs**: Use `safe-outputs` configuration for GitHub API operations\n")
	sb.WriteString("3. **Command Execution**: Avoid executing commands with untrusted input\n")
	sb.WriteString("4. **File Operations**: Validate content before writing to files\n")
	sb.WriteString("5. **Least Privilege**: Grant minimum required permissions to workflows\n")
	sb.WriteString("6. **Review AI Output**: Always review AI-generated content before execution\n\n")

	return sb.String()
}

// Helper functions for formatting and styling

func getSourceStyle(sourceType string) string {
	switch sourceType {
	case "agentic_workflow":
		return ":::unsafe"
	case "web_search", "user_input":
		return ":::unsafe"
	case "web_fetch", "github_api":
		return ":::medium"
	default:
		return ""
	}
}

func getSinkStyle(sinkType string) string {
	switch sinkType {
	case "command_execution":
		return ":::critical"
	case "file_write":
		return ":::unsafe"
	case "github_api", "external_api":
		return ":::medium"
	default:
		return ""
	}
}

func getEdgeStyle(isUnsafe bool) string {
	if isUnsafe {
		return "|❌ UNSAFE|"
	}
	return "|✅|"
}

func formatNodeLabel(nodeType, name string) string {
	// Extract just the workflow name and tool
	parts := strings.Split(name, ":")
	if len(parts) > 1 {
		return fmt.Sprintf("%s\\n%s", formatSourceType(nodeType), parts[len(parts)-1])
	}
	return fmt.Sprintf("%s\\n%s", formatSourceType(nodeType), name)
}

func formatSourceType(sourceType string) string {
	switch sourceType {
	case "agentic_workflow":
		return "Agentic Workflow"
	case "web_search":
		return "Web Search"
	case "web_fetch":
		return "Web Fetch"
	case "github_api":
		return "GitHub API"
	case "user_input":
		return "User Input"
	default:
		return strings.Title(strings.ReplaceAll(sourceType, "_", " "))
	}
}

func formatSinkType(sinkType string) string {
	switch sinkType {
	case "file_write":
		return "File Write"
	case "command_execution":
		return "Command Execution"
	case "github_api":
		return "GitHub API"
	case "external_api":
		return "External API"
	default:
		return strings.Title(strings.ReplaceAll(sinkType, "_", " "))
	}
}

func groupSourcesByType(sources []TaintSource) map[string][]TaintSource {
	grouped := make(map[string][]TaintSource)
	for _, source := range sources {
		grouped[source.Type] = append(grouped[source.Type], source)
	}
	// Sort keys for consistent output
	return grouped
}

func groupSinksByType(sinks []TaintSink) map[string][]TaintSink {
	grouped := make(map[string][]TaintSink)
	for _, sink := range sinks {
		grouped[sink.Type] = append(grouped[sink.Type], sink)
	}
	return grouped
}

func getRecommendation(path TaintPath) string {
	switch {
	case path.Sink.Type == "command_execution":
		return "Add input validation and use allowlists for commands. Consider using safe-outputs instead of direct execution."
	case path.Sink.Type == "file_write":
		return "Validate and sanitize content before writing. Use safe-outputs for managed file operations."
	case path.Source.Type == "web_search" || path.Source.Type == "web_fetch":
		return "Implement strict content validation and sanitization for external web content."
	case path.Source.Type == "user_input":
		return "Always sanitize user input and implement allowlists for expected values."
	case strings.Contains(path.Sink.Name, "mcp"):
		return "Review MCP server configuration and ensure proper authentication and validation."
	default:
		return "Review this data flow and ensure appropriate validation and sanitization are in place."
	}
}

// sortUnsafePathsBySeverity sorts unsafe paths by severity (critical first)
func sortUnsafePathsBySeverity(paths []TaintPath) {
	sort.Slice(paths, func(i, j int) bool {
		severityOrder := map[string]int{
			"critical": 0,
			"high":     1,
			"medium":   2,
			"low":      3,
		}
		
		sevi := severityOrder[paths[i].Sink.Severity]
		sevj := severityOrder[paths[j].Sink.Severity]
		
		return sevi < sevj
	})
}
