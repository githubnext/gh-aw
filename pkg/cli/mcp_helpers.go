package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// WorkflowMCPInfo represents a workflow file with its MCP configurations
type WorkflowMCPInfo struct {
	FilePath    string                   // Full path to the workflow file
	Name        string                   // Base name without .md extension
	Frontmatter map[string]any           // Parsed frontmatter
	MCPConfigs  []parser.MCPServerConfig // Extracted MCP server configurations
}

// scanWorkflowsDirectory scans the workflows directory for .md files and extracts MCP configurations.
// It returns a slice of WorkflowMCPInfo for workflows that contain MCP servers.
//
// Parameters:
//   - workflowsDir: Path to the workflows directory (typically .github/workflows)
//   - serverFilter: Optional filter for MCP server names. Empty string means no filtering (returns all servers)
//   - verbose: If true, prints warning messages for skipped files
func scanWorkflowsDirectory(workflowsDir string, serverFilter string, verbose bool) ([]WorkflowMCPInfo, error) {
	// Check if the workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflows directory not found: %s", workflowsDir)
	}

	// Find all .md files in the workflows directory
	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to search for workflow files: %w", err)
	}

	var workflowInfos []WorkflowMCPInfo

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

		// Only include workflows that have MCP configurations
		if len(mcpConfigs) > 0 {
			baseName := strings.TrimSuffix(filepath.Base(file), ".md")
			workflowInfos = append(workflowInfos, WorkflowMCPInfo{
				FilePath:    file,
				Name:        baseName,
				Frontmatter: frontmatterData.Frontmatter,
				MCPConfigs:  mcpConfigs,
			})
		}
	}

	return workflowInfos, nil
}

// loadWorkflowWithMCP loads a workflow file and extracts its frontmatter and MCP configurations.
//
// Parameters:
//   - workflowFile: Path to the workflow file (with or without .md extension)
//   - serverFilter: Optional filter for MCP server names. Empty string means no filtering (returns all servers)
//
// Returns:
//   - WorkflowMCPInfo containing the parsed workflow data and MCP configurations
//   - Error if the file cannot be read or parsed
func loadWorkflowWithMCP(workflowFile string, serverFilter string) (*WorkflowMCPInfo, error) {
	// Resolve the workflow file path
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		return nil, err
	}

	// Convert to absolute path if needed
	if !filepath.IsAbs(workflowPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		workflowPath = filepath.Join(cwd, workflowPath)
	}

	// Parse the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow file: %w", err)
	}

	// Extract MCP configurations
	mcpConfigs, err := parser.ExtractMCPConfigurations(workflowData.Frontmatter, serverFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to extract MCP configurations: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(workflowPath), ".md")

	return &WorkflowMCPInfo{
		FilePath:    workflowPath,
		Name:        baseName,
		Frontmatter: workflowData.Frontmatter,
		MCPConfigs:  mcpConfigs,
	}, nil
}
