package cli

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// applyImportsToFrontmatter merges imported MCP servers and tools into frontmatter
// Returns a new frontmatter map with imports applied
func applyImportsToFrontmatter(frontmatter map[string]any, importsResult *parser.ImportsResult) (map[string]any, error) {
	mcpInspectLog.Print("Applying imports to frontmatter")

	// Create a copy of the frontmatter to avoid modifying the original
	result := make(map[string]any)
	for k, v := range frontmatter {
		result[k] = v
	}

	// If there are no imported MCP servers or tools, return as-is
	if importsResult.MergedMCPServers == "" && importsResult.MergedTools == "" {
		return result, nil
	}

	// Get existing mcp-servers from frontmatter
	var existingMCPServers map[string]any
	if mcpServersSection, exists := result["mcp-servers"]; exists {
		if mcpServers, ok := mcpServersSection.(map[string]any); ok {
			existingMCPServers = mcpServers
		}
	}
	if existingMCPServers == nil {
		existingMCPServers = make(map[string]any)
	}

	// Merge imported MCP servers using the workflow compiler's merge logic
	compiler := workflow.NewCompiler(false, "", "")
	mergedMCPServers, err := compiler.MergeMCPServers(existingMCPServers, importsResult.MergedMCPServers)
	if err != nil {
		errMsg := fmt.Sprintf("failed to merge imported MCP servers: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		return nil, fmt.Errorf("failed to merge imported MCP servers: %w", err)
	}

	// Update mcp-servers in the result
	if len(mergedMCPServers) > 0 {
		result["mcp-servers"] = mergedMCPServers
	}

	// Get existing tools from frontmatter
	var existingTools map[string]any
	if toolsSection, exists := result["tools"]; exists {
		if tools, ok := toolsSection.(map[string]any); ok {
			existingTools = tools
		}
	}
	if existingTools == nil {
		existingTools = make(map[string]any)
	}

	// Merge imported tools using the workflow compiler's merge logic
	mergedTools, err := compiler.MergeTools(existingTools, importsResult.MergedTools)
	if err != nil {
		errMsg := fmt.Sprintf("failed to merge imported tools: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(errMsg))
		return nil, fmt.Errorf("failed to merge imported tools: %w", err)
	}

	// Update tools in the result
	if len(mergedTools) > 0 {
		result["tools"] = mergedTools
	}

	return result, nil
}
