package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// filterOutSafeOutputs removes safe-outputs MCP servers from the list since they are
// handled by the workflow compiler and not actual MCP servers that can be inspected
func filterOutSafeOutputs(configs []parser.MCPServerConfig) []parser.MCPServerConfig {
	var filteredConfigs []parser.MCPServerConfig
	for _, config := range configs {
		if config.Name != "safe-outputs" {
			filteredConfigs = append(filteredConfigs, config)
		}
	}
	return filteredConfigs
}

// applyImportsToFrontmatter merges imported MCP servers and tools into frontmatter
// Returns a new frontmatter map with imports applied
func applyImportsToFrontmatter(frontmatter map[string]any, importsResult *parser.ImportsResult) (map[string]any, error) {
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
		return nil, fmt.Errorf("failed to merge imported tools: %w", err)
	}

	// Update tools in the result
	if len(mergedTools) > 0 {
		result["tools"] = mergedTools
	}

	return result, nil
}

// InspectWorkflowMCP inspects MCP servers used by a workflow and lists available tools, resources, and roots
func InspectWorkflowMCP(workflowFile string, serverFilter string, toolFilter string, verbose bool) error {
	workflowsDir := getWorkflowsDir()

	// If no workflow file specified, show available workflow files with MCP configs
	if workflowFile == "" {
		return listWorkflowsWithMCP(workflowsDir, verbose)
	}

	// Load the workflow file with MCP configurations
	workflowInfo, err := loadWorkflowWithMCP(workflowFile, serverFilter)
	if err != nil {
		return err
	}

	workflowPath := workflowInfo.FilePath

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Inspecting MCP servers in: %s", workflowPath)))
	}

	// Validate frontmatter before analyzing MCPs
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(workflowInfo.Frontmatter, workflowPath); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Frontmatter validation failed: %v", err)))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Continuing with MCP inspection (validation errors may affect results)"))
		} else {
			return fmt.Errorf("frontmatter validation failed: %w", err)
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Frontmatter validation passed"))
	}

	// Process imports from frontmatter to merge imported MCP servers
	markdownDir := filepath.Dir(workflowPath)
	importsResult, err := parser.ProcessImportsFromFrontmatterWithManifest(workflowInfo.Frontmatter, markdownDir)
	if err != nil {
		return fmt.Errorf("failed to process imports from frontmatter: %w", err)
	}

	// Apply imported MCP servers to frontmatter
	frontmatterWithImports, err := applyImportsToFrontmatter(workflowInfo.Frontmatter, importsResult)
	if err != nil {
		return fmt.Errorf("failed to apply imports: %w", err)
	}

	// Validate MCP configurations specifically using compiler validation
	if toolsSection, hasTools := frontmatterWithImports["tools"]; hasTools {
		if tools, ok := toolsSection.(map[string]any); ok {
			if err := workflow.ValidateMCPConfigs(tools); err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("MCP configuration validation failed: %v", err)))
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Continuing with MCP inspection (validation errors may affect results)"))
				} else {
					return fmt.Errorf("MCP configuration validation failed: %w", err)
				}
			} else if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("MCP configuration validation passed"))
			}
		}
	}

	// Extract MCP configurations from frontmatter with imports applied
	mcpConfigs, err := parser.ExtractMCPConfigurations(frontmatterWithImports, serverFilter)
	if err != nil {
		return fmt.Errorf("failed to extract MCP configurations: %w", err)
	}

	// Filter out safe-outputs MCP servers for inspection
	mcpConfigs = filterOutSafeOutputs(mcpConfigs)

	if len(mcpConfigs) == 0 {
		if serverFilter != "" {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No MCP servers matching filter '%s' found in workflow", serverFilter)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No MCP servers found in workflow"))
		}
		return nil
	}

	// Inspect each MCP server
	if toolFilter != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d MCP server(s), looking for tool '%s'", len(mcpConfigs), toolFilter)))
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

	return nil
}

// listWorkflowsWithMCP shows available workflow files that contain MCP configurations
func listWorkflowsWithMCP(workflowsDir string, verbose bool) error {
	// Scan workflows directory for workflows with MCP servers
	workflowInfos, err := scanWorkflowsDirectory(workflowsDir, "", verbose)
	if err != nil {
		// Provide more specific error for missing directory
		if os.IsNotExist(err) {
			return fmt.Errorf("no .github/workflows directory found")
		}
		return fmt.Errorf("failed to read workflow files: %w", err)
	}

	var workflowsWithMCP []string

	for _, info := range workflowInfos {
		// Validate frontmatter before including in list (non-verbose mode to avoid spam)
		if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(info.Frontmatter, info.FilePath); err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping %s due to frontmatter validation: %v", filepath.Base(info.FilePath), err)))
			}
			continue
		}

		// Filter out safe-outputs MCP servers for inspection
		filteredConfigs := filterOutSafeOutputs(info.MCPConfigs)

		if len(filteredConfigs) > 0 {
			workflowsWithMCP = append(workflowsWithMCP, filepath.Base(info.FilePath))
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

// NewMCPInspectCommand creates the mcp inspect command
func NewMCPInspectCommand() *cobra.Command {
	var serverFilter string
	var toolFilter string
	var spawnInspector bool

	cmd := &cobra.Command{
		Use:   "mcp inspect [workflow-file]",
		Short: "Inspect MCP servers and list available tools, resources, and roots",
		Long: `Inspect MCP servers used by a workflow and display available tools, resources, and roots.

This command starts each MCP server configured in the workflow, queries its capabilities,
and displays the results in a formatted table. It supports stdio, Docker, and HTTP MCP servers.

Examples:
  gh aw mcp inspect                    # List workflows with MCP servers
  gh aw mcp inspect weekly-research    # Inspect MCP servers in weekly-research.md  
  gh aw mcp inspect repomind --server repo-mind  # Inspect only the repo-mind server
  gh aw mcp inspect weekly-research --server github --tool create_issue  # Show details for a specific tool
  gh aw mcp inspect weekly-research -v # Verbose output with detailed connection info
  gh aw mcp inspect weekly-research --inspector  # Launch @modelcontextprotocol/inspector

The command will:
- Parse the workflow file to extract MCP server configurations
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
			if cmd.Parent() != nil {
				parentVerbose, _ := cmd.Parent().PersistentFlags().GetBool("verbose")
				verbose = verbose || parentVerbose
			}

			// Validate that tool flag requires server flag
			if toolFilter != "" && serverFilter == "" {
				return fmt.Errorf("--tool flag requires --server flag to be specified")
			}

			// Handle spawn inspector flag
			if spawnInspector {
				return spawnMCPInspector(workflowFile, serverFilter, verbose)
			}

			return InspectWorkflowMCP(workflowFile, serverFilter, toolFilter, verbose)
		},
	}

	cmd.Flags().StringVar(&serverFilter, "server", "", "Filter to inspect only the specified MCP server")
	cmd.Flags().StringVar(&toolFilter, "tool", "", "Show detailed information about a specific tool (requires --server)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output with detailed connection information")
	cmd.Flags().BoolVar(&spawnInspector, "inspector", false, "Launch the official @modelcontextprotocol/inspector tool")

	return cmd
}

// spawnMCPInspector launches the official @modelcontextprotocol/inspector tool
// and spawns any stdio MCP servers beforehand
func spawnMCPInspector(workflowFile string, serverFilter string, verbose bool) error {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found. Please install Node.js and npm to use the MCP inspector: %w", err)
	}

	var mcpConfigs []parser.MCPServerConfig
	var serverProcesses []*exec.Cmd
	var wg sync.WaitGroup

	// If workflow file is specified, extract MCP configurations and start servers
	if workflowFile != "" {
		workflowsDir := constants.GetWorkflowDir()

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

		// Parse the workflow file to extract MCP configurations
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			return err
		}

		workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			return err
		}

		// Process imports from frontmatter to merge imported MCP servers
		markdownDir := filepath.Dir(workflowPath)
		importsResult, err := parser.ProcessImportsFromFrontmatterWithManifest(workflowData.Frontmatter, markdownDir)
		if err != nil {
			return fmt.Errorf("failed to process imports from frontmatter: %w", err)
		}

		// Apply imported MCP servers to frontmatter
		frontmatterWithImports, err := applyImportsToFrontmatter(workflowData.Frontmatter, importsResult)
		if err != nil {
			return fmt.Errorf("failed to apply imports: %w", err)
		}

		// Extract MCP configurations from frontmatter with imports applied
		mcpConfigs, err = parser.ExtractMCPConfigurations(frontmatterWithImports, serverFilter)
		if err != nil {
			return err
		}

		if len(mcpConfigs) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d MCP server(s) in workflow:", len(mcpConfigs))))
			for _, config := range mcpConfigs {
				fmt.Printf("  • %s (%s)\n", config.Name, config.Type)
			}
			fmt.Println()

			// Start stdio MCP servers in the background
			stdioServers := []parser.MCPServerConfig{}
			for _, config := range mcpConfigs {
				if config.Type == "stdio" {
					stdioServers = append(stdioServers, config)
				}
			}

			if len(stdioServers) > 0 {
				fmt.Println(console.FormatInfoMessage("Starting stdio MCP servers..."))

				for _, config := range stdioServers {
					if verbose {
						fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Starting server: %s", config.Name)))
					}

					// Create the command for the MCP server
					var cmd *exec.Cmd
					if config.Container != "" {
						// Docker container mode
						args := append([]string{"run", "--rm", "-i"}, config.Args...)
						cmd = exec.Command("docker", args...)
					} else {
						// Direct command mode
						if config.Command == "" {
							fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Skipping server %s: no command specified", config.Name)))
							continue
						}
						cmd = exec.Command(config.Command, config.Args...)
					}

					// Set environment variables
					cmd.Env = os.Environ()
					for key, value := range config.Env {
						// Resolve environment variable references
						resolvedValue := os.ExpandEnv(value)
						cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, resolvedValue))
					}

					// Start the server process
					if err := cmd.Start(); err != nil {
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to start server %s: %v", config.Name, err)))
						continue
					}

					serverProcesses = append(serverProcesses, cmd)

					// Monitor the process in the background
					wg.Add(1)
					go func(serverCmd *exec.Cmd, serverName string) {
						defer wg.Done()
						if err := serverCmd.Wait(); err != nil && verbose {
							fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Server %s exited with error: %v", serverName, err)))
						}
					}(cmd, config.Name)

					if verbose {
						fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Started server: %s (PID: %d)", config.Name, cmd.Process.Pid)))
					}
				}

				// Give servers a moment to start up
				time.Sleep(2 * time.Second)
				fmt.Println(console.FormatSuccessMessage("All stdio servers started successfully"))
			}

			fmt.Println(console.FormatInfoMessage("Configuration details for MCP inspector:"))
			for _, config := range mcpConfigs {
				fmt.Printf("\n📡 %s (%s):\n", config.Name, config.Type)
				switch config.Type {
				case "stdio":
					if config.Container != "" {
						fmt.Printf("  Container: %s\n", config.Container)
					} else {
						fmt.Printf("  Command: %s\n", config.Command)
						if len(config.Args) > 0 {
							fmt.Printf("  Args: %s\n", strings.Join(config.Args, " "))
						}
					}
				case "http":
					fmt.Printf("  URL: %s\n", config.URL)
				}
				if len(config.Env) > 0 {
					fmt.Printf("  Environment Variables: %v\n", config.Env)
				}
			}
			fmt.Println()
		} else {
			fmt.Println(console.FormatWarningMessage("No MCP servers found in workflow"))
			return nil
		}
	}

	// Set up cleanup function for stdio servers
	defer func() {
		if len(serverProcesses) > 0 {
			fmt.Println(console.FormatInfoMessage("Cleaning up MCP servers..."))
			for i, cmd := range serverProcesses {
				if cmd.Process != nil {
					if err := cmd.Process.Kill(); err != nil && verbose {
						fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to kill server process %d: %v", cmd.Process.Pid, err)))
					}
				}
				// Give each process a chance to clean up
				if i < len(serverProcesses)-1 {
					time.Sleep(100 * time.Millisecond)
				}
			}
			// Wait for all background goroutines to finish (with timeout)
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// All finished
			case <-time.After(5 * time.Second):
				// Timeout waiting for cleanup
				if verbose {
					fmt.Println(console.FormatWarningMessage("Timeout waiting for server cleanup"))
				}
			}
		}
	}()

	fmt.Println(console.FormatInfoMessage("Launching @modelcontextprotocol/inspector..."))
	fmt.Println(console.FormatInfoMessage("Visit http://localhost:5173 after the inspector starts"))
	if len(serverProcesses) > 0 {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("%d stdio MCP server(s) are running in the background", len(serverProcesses))))
		fmt.Println(console.FormatInfoMessage("Configure them in the inspector using the details shown above"))
	}

	cmd := exec.Command("npx", "@modelcontextprotocol/inspector")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// NewMCPInspectSubcommand creates the mcp inspect subcommand
// This is the former mcp inspect command now nested under mcp
func NewMCPInspectSubcommand() *cobra.Command {
	var serverFilter string
	var toolFilter string
	var spawnInspector bool

	cmd := &cobra.Command{
		Use:   "inspect [workflow-file]",
		Short: "Inspect MCP servers and list available tools, resources, and roots",
		Long: `Inspect MCP servers used by a workflow and display available tools, resources, and roots.

This command starts each MCP server configured in the workflow, queries its capabilities,
and displays the results in a formatted table. It supports stdio, Docker, and HTTP MCP servers.

Examples:
  gh aw mcp inspect                    # List workflows with MCP servers
  gh aw mcp inspect weekly-research    # Inspect MCP servers in weekly-research.md  
  gh aw mcp inspect weekly-research --server github --tool create_issue  # Show details for a specific tool
  gh aw mcp inspect weekly-research -v # Verbose output with detailed connection info
  gh aw mcp inspect weekly-research --inspector  # Launch @modelcontextprotocol/inspector

The command will:
- Parse the workflow file to extract MCP server configurations
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
			// Check for verbose flag from parent commands (root and mcp)
			if cmd.Parent() != nil {
				if parentVerbose, _ := cmd.Parent().PersistentFlags().GetBool("verbose"); parentVerbose {
					verbose = true
				}
				if cmd.Parent().Parent() != nil {
					if rootVerbose, _ := cmd.Parent().Parent().PersistentFlags().GetBool("verbose"); rootVerbose {
						verbose = true
					}
				}
			}

			// Validate that tool flag requires server flag
			if toolFilter != "" && serverFilter == "" {
				return fmt.Errorf("--tool flag requires --server flag to be specified")
			}

			// Handle spawn inspector flag
			if spawnInspector {
				return spawnMCPInspector(workflowFile, serverFilter, verbose)
			}

			return InspectWorkflowMCP(workflowFile, serverFilter, toolFilter, verbose)
		},
	}

	cmd.Flags().StringVar(&serverFilter, "server", "", "Filter to inspect only the specified MCP server")
	cmd.Flags().StringVar(&toolFilter, "tool", "", "Show detailed information about a specific tool (requires --server)")
	cmd.Flags().BoolVar(&spawnInspector, "inspector", false, "Launch the official @modelcontextprotocol/inspector tool")

	return cmd
}
