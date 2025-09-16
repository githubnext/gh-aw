package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// InspectWorkflowMCP inspects MCP servers used by a workflow and lists available tools, resources, and roots
func InspectWorkflowMCP(workflowFile string, serverFilter string, toolFilter string, verbose bool) error {
	workflowsDir := getWorkflowsDir()

	// If no workflow file specified, show available workflow files with MCP configs
	if workflowFile == "" {
		return listWorkflowsWithMCP(workflowsDir, verbose)
	}

	// Normalize the workflow file path
	if !strings.HasSuffix(workflowFile, ".md") {
		workflowFile += ".md"
	}

	workflowPath := filepath.Join(workflowsDir, workflowFile)
	if !filepath.IsAbs(workflowPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		workflowPath = filepath.Join(cwd, workflowPath)
	}

	// Check if file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		return fmt.Errorf("workflow file not found: %s", workflowPath)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Inspecting MCP servers in: %s", workflowPath)))
	}

	// Parse the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse workflow file: %w", err)
	}

	// Validate frontmatter before analyzing MCPs
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(workflowData.Frontmatter, workflowPath); err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Frontmatter validation failed: %v", err)))
			fmt.Println(console.FormatInfoMessage("Continuing with MCP inspection (validation errors may affect results)"))
		} else {
			return fmt.Errorf("frontmatter validation failed: %w", err)
		}
	} else if verbose {
		fmt.Println(console.FormatSuccessMessage("Frontmatter validation passed"))
	}

	// Validate MCP configurations specifically using compiler validation
	if toolsSection, hasTools := workflowData.Frontmatter["tools"]; hasTools {
		if tools, ok := toolsSection.(map[string]any); ok {
			if err := workflow.ValidateMCPConfigs(tools); err != nil {
				if verbose {
					fmt.Println(console.FormatWarningMessage(fmt.Sprintf("MCP configuration validation failed: %v", err)))
					fmt.Println(console.FormatInfoMessage("Continuing with MCP inspection (validation errors may affect results)"))
				} else {
					return fmt.Errorf("MCP configuration validation failed: %w", err)
				}
			} else if verbose {
				fmt.Println(console.FormatSuccessMessage("MCP configuration validation passed"))
			}
		}
	}

	// Extract MCP configurations
	mcpConfigs, err := parser.ExtractMCPConfigurations(workflowData.Frontmatter, serverFilter)
	if err != nil {
		return fmt.Errorf("failed to extract MCP configurations: %w", err)
	}

	if len(mcpConfigs) == 0 {
		if serverFilter != "" {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("No MCP servers matching filter '%s' found in workflow", serverFilter)))
		} else {
			fmt.Println(console.FormatWarningMessage("No MCP servers found in workflow"))
		}
		return nil
	}

	// Inspect each MCP server
	if toolFilter != "" {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d MCP server(s), looking for tool '%s'", len(mcpConfigs), toolFilter)))
	} else {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d MCP server(s) to inspect", len(mcpConfigs))))
	}
	fmt.Println()

	for i, config := range mcpConfigs {
		if i > 0 {
			fmt.Println()
		}
		if err := inspectMCPServer(config, toolFilter, verbose); err != nil {
			fmt.Println(console.FormatError(console.CompilerError{
				Type:    "error",
				Message: fmt.Sprintf("Failed to inspect MCP server '%s': %v", config.Name, err),
			}))
		}
	}

	// Always generate MCP configuration in memory using Claude engine
	if verbose {
		fmt.Println(console.FormatInfoMessage("Generating MCP configuration using Claude agentic engine..."))
	}
	
	if err := generateAndDisplayMCPConfig(workflowData, verbose); err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to generate MCP configuration: %v", err)))
		}
	}

	return nil
}

// generateAndDisplayMCPConfig generates MCP configuration using Claude engine and displays it
func generateAndDisplayMCPConfig(workflowData *parser.FrontmatterResult, verbose bool) error {
	// Create Claude engine to generate MCP configuration
	claudeEngine := workflow.NewClaudeEngine()
	
	// Extract tools from frontmatter
	tools := make(map[string]any)
	if toolsSection, hasTools := workflowData.Frontmatter["tools"]; hasTools {
		if toolsMap, ok := toolsSection.(map[string]any); ok {
			tools = toolsMap
		}
	}

	// Extract MCP tool names from existing configurations
	mcpConfigs, err := parser.ExtractMCPConfigurations(workflowData.Frontmatter, "")
	if err != nil {
		return fmt.Errorf("failed to extract MCP configurations: %w", err)
	}

	// Build list of MCP servers to include in config
	mcpTools := []string{}
	
	// Add existing MCP server configurations
	for _, config := range mcpConfigs {
		mcpTools = append(mcpTools, config.Name)
	}

	// Add standard servers if configured (avoid duplicates)
	if _, hasGithub := tools["github"]; hasGithub {
		found := false
		for _, existing := range mcpTools {
			if existing == "github" {
				found = true
				break
			}
		}
		if !found {
			mcpTools = append(mcpTools, "github")
		}
	}
	
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		found := false
		for _, existing := range mcpTools {
			if existing == "playwright" {
				found = true
				break
			}
		}
		if !found {
			mcpTools = append(mcpTools, "playwright")
		}
	}
	
	if _, hasSafeOutputs := workflowData.Frontmatter["safe-outputs"]; hasSafeOutputs {
		found := false
		for _, existing := range mcpTools {
			if existing == "safe-outputs" {
				found = true
				break
			}
		}
		if !found {
			mcpTools = append(mcpTools, "safe-outputs")
		}
	}

	if len(mcpTools) == 0 {
		if verbose {
			fmt.Println(console.FormatInfoMessage("No MCP tools found for configuration generation"))
		}
		return nil
	}

	// Create a minimal WorkflowData for MCP config generation
	workflowDataForMCP := &workflow.WorkflowData{
		Tools:              tools,
		NetworkPermissions: nil, // Will be populated if needed
	}

	// Generate the MCP configuration
	var mcpConfigBuilder strings.Builder
	claudeEngine.RenderMCPConfig(&mcpConfigBuilder, tools, mcpTools, workflowDataForMCP)
	
	fmt.Println()
	fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Generated MCP configuration for %d server(s)", len(mcpTools))))
	fmt.Println(console.FormatInfoMessage("Claude Engine MCP Configuration:"))
	fmt.Println()
	fmt.Println(mcpConfigBuilder.String())

	return nil
}

// listWorkflowsWithMCP shows available workflow files that contain MCP configurations
func listWorkflowsWithMCP(workflowsDir string, verbose bool) error {
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("no .github/workflows directory found")
	}

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to read workflow files: %w", err)
	}

	var workflowsWithMCP []string

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		// Validate frontmatter before analyzing MCPs (non-verbose mode to avoid spam)
		if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(workflowData.Frontmatter, file); err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping %s due to frontmatter validation: %v", filepath.Base(file), err)))
			}
			continue
		}

		mcpConfigs, err := parser.ExtractMCPConfigurations(workflowData.Frontmatter, "")
		if err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		if len(mcpConfigs) > 0 {
			workflowsWithMCP = append(workflowsWithMCP, filepath.Base(file))
		}
	}

	if len(workflowsWithMCP) == 0 {
		fmt.Println(console.FormatInfoMessage("No workflows with MCP servers found"))
		return nil
	}

	fmt.Println(console.FormatInfoMessage("Workflows with MCP servers:"))
	for _, workflow := range workflowsWithMCP {
		fmt.Printf("  • %s\n", workflow)
	}
	fmt.Printf("\nRun 'gh aw mcp inspect <workflow-name>' to inspect MCP servers in a specific workflow.\n")

	return nil
}


// NewMCPInspectSubCommand creates the mcp inspect subcommand
func NewMCPInspectSubCommand() *cobra.Command {
	var serverFilter string
	var toolFilter string

	cmd := &cobra.Command{
		Use:   "inspect [workflow-file]",
		Short: "Inspect MCP servers and list available tools, resources, and roots",
		Long: `Inspect MCP servers used by a workflow and display available tools, resources, and roots.

This command generates MCP configurations using the Claude agentic engine and launches
configured servers including github, playwright, and safe-outputs.

Examples:
  gh aw mcp inspect                    # List workflows with MCP servers
  gh aw mcp inspect weekly-research    # Inspect MCP servers in weekly-research.md  
  gh aw mcp inspect repomind --server repo-mind  # Inspect only the repo-mind server
  gh aw mcp inspect weekly-research --server github --tool create_issue  # Show details for a specific tool
  gh aw mcp inspect weekly-research -v # Verbose output with detailed connection info

The command will:
- Parse the workflow file to extract MCP server configurations
- Generate MCP configuration using the Claude agentic engine
- Start each MCP server (stdio, docker, http)
- Query available tools, resources, and roots
- Validate required secrets are available  
- Display results in formatted tables with error details`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var workflowFile string
			if len(args) > 0 {
				workflowFile = args[0]
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			if cmd.Parent() != nil && cmd.Parent().Parent() != nil {
				parentVerbose, _ := cmd.Parent().Parent().PersistentFlags().GetBool("verbose")
				verbose = verbose || parentVerbose
			}

			// Validate that tool flag requires server flag
			if toolFilter != "" && serverFilter == "" {
				return fmt.Errorf("--tool flag requires --server flag to be specified")
			}



			return InspectWorkflowMCP(workflowFile, serverFilter, toolFilter, verbose)
		},
	}

	cmd.Flags().StringVar(&serverFilter, "server", "", "Filter to inspect only the specified MCP server")
	cmd.Flags().StringVar(&toolFilter, "tool", "", "Show detailed information about a specific tool (requires --server)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output with detailed connection information")

	return cmd
}

