package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// WorkflowMCPMetadata contains metadata about MCP servers in a workflow
type WorkflowMCPMetadata struct {
	FilePath    string
	FileName    string
	BaseName    string
	MCPConfigs  []parser.MCPServerConfig
	Frontmatter map[string]any
}

// ScanWorkflowsForMCP scans workflow files for MCP configurations
// Returns metadata for workflows that contain MCP servers
func ScanWorkflowsForMCP(workflowsDir string, serverFilter string, verbose bool) ([]WorkflowMCPMetadata, error) {
	// Check if the workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflows directory not found: %s", workflowsDir)
	}

	// Find all .md files in the workflows directory
	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to search for workflow files: %w", err)
	}

	var results []WorkflowMCPMetadata

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		frontmatterData, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		mcpConfigs, err := parser.ExtractMCPConfigurations(frontmatterData.Frontmatter, serverFilter)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Error extracting MCP from %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		if len(mcpConfigs) > 0 {
			baseName := strings.TrimSuffix(filepath.Base(file), ".md")
			results = append(results, WorkflowMCPMetadata{
				FilePath:    file,
				FileName:    filepath.Base(file),
				BaseName:    baseName,
				MCPConfigs:  mcpConfigs,
				Frontmatter: frontmatterData.Frontmatter,
			})
		}
	}

	return results, nil
}
