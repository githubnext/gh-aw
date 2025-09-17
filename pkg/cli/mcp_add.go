package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// listAvailableServers shows a list of available MCP servers from the registry
func listAvailableServers(registryURL string, verbose bool) error {
	// Create registry client
	registryClient := NewMCPRegistryClient(registryURL)

	// Search for all servers (empty query)
	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Fetching available MCP servers from registry: %s", registryClient.registryURL)))
	}

	servers, err := registryClient.SearchServers("")
	if err != nil {
		return fmt.Errorf("failed to fetch MCP servers: %w", err)
	}

	if verbose {
		fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Retrieved %d servers from registry", len(servers))))
		if len(servers) > 0 {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("First server example - ID: '%s', Name: '%s', Description: '%s'",
				servers[0].ID, servers[0].Name, servers[0].Description)))
		}
	}

	if len(servers) == 0 {
		fmt.Println(console.FormatWarningMessage("No MCP servers found in the registry"))
		return nil
	}

	// Prepare table data
	headers := []string{"Name", "Description"}
	rows := make([][]string, 0, len(servers))

	for _, server := range servers {
		// Use server name as the primary identifier
		name := server.Name
		if name == "" {
			name = server.ID // fallback to ID if no name
		}

		// Truncate long descriptions for table display
		description := server.Description
		if len(description) > 80 {
			description = description[:77] + "..."
		}
		if description == "" {
			description = "-"
		}

		rows = append(rows, []string{
			name,
			description,
		})
	}

	// Create and render table
	tableConfig := console.TableConfig{
		Title:     "Available MCP Servers",
		Headers:   headers,
		Rows:      rows,
		ShowTotal: true,
		TotalRow:  []string{fmt.Sprintf("Total: %d servers", len(servers)), "", ""},
	}

	fmt.Print(console.RenderTable(tableConfig))
	fmt.Println(console.FormatInfoMessage("Usage: gh aw mcp add <workflow-file> <server-name>"))

	return nil
}

// AddMCPTool adds an MCP tool to an agentic workflow
func AddMCPTool(workflowFile string, mcpServerID string, registryURL string, transportType string, customToolID string, verbose bool) error {
	// Resolve the workflow file path
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Adding MCP tool '%s' to workflow: %s", mcpServerID, console.ToRelativePath(workflowPath))))
	}

	// Create registry client
	registryClient := NewMCPRegistryClient(registryURL)

	// Search for the MCP server in the registry
	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Searching for MCP server '%s' in registry: %s", mcpServerID, registryClient.registryURL)))
	}

	servers, err := registryClient.SearchServers(mcpServerID)
	if err != nil {
		return fmt.Errorf("failed to search MCP registry: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no MCP servers found matching '%s'", mcpServerID)
	}

	// Find exact match by name first, then by ID
	var selectedServer *MCPRegistryServer
	for i, server := range servers {
		// Prioritize name matches over ID matches
		if server.Name == mcpServerID {
			selectedServer = &servers[i]
			break
		}
	}

	// If no name match, try ID match
	if selectedServer == nil {
		for i, server := range servers {
			if server.ID == mcpServerID {
				selectedServer = &servers[i]
				break
			}
		}
	}

	// If still no exact match, use the first result if it looks like a partial match
	if selectedServer == nil && len(servers) > 0 {
		selectedServer = &servers[0]
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("No exact match for '%s', using closest match: %s", mcpServerID, selectedServer.Name)))
		}
	}

	if selectedServer == nil {
		return fmt.Errorf("no MCP servers found matching '%s'", mcpServerID)
	}

	// Determine tool ID (use custom if provided, otherwise use cleaned server name)
	toolID := cleanMCPToolID(selectedServer.Name)
	if customToolID != "" {
		toolID = customToolID
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Selected server: %s (ID: %s, Transport: %s)", selectedServer.Name, selectedServer.ID, selectedServer.Transport)))
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Will add as tool ID: %s", toolID)))
	}

	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse the workflow file
	workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse workflow file: %w", err)
	}

	// Check if tool already exists
	if workflowData.Frontmatter["tools"] != nil {
		if tools, ok := workflowData.Frontmatter["tools"].(map[string]any); ok {
			if _, exists := tools[toolID]; exists {
				return fmt.Errorf("tool '%s' already exists in workflow", toolID)
			}
		}
	}

	// Create MCP tool configuration based on server info and preferences
	mcpConfig, err := createMCPToolConfig(selectedServer, transportType, registryClient.registryURL, verbose)
	if err != nil {
		return fmt.Errorf("failed to create MCP tool configuration: %w", err)
	}

	// Add the tool to the workflow
	if err := addToolToWorkflow(workflowPath, toolID, mcpConfig, verbose); err != nil {
		return fmt.Errorf("failed to add tool to workflow: %w", err)
	}

	fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Added MCP tool '%s' to workflow %s", toolID, console.ToRelativePath(workflowPath))))

	// Check for required secrets and provide CLI commands if missing
	if err := checkAndSuggestSecrets(mcpConfig, verbose); err != nil {
		// Don't fail the command if secret checking fails, just log a warning
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Could not check repository secrets: %v", err)))
		}
	}

	// Compile the workflow
	if verbose {
		fmt.Println(console.FormatInfoMessage("Compiling workflow..."))
	}

	compiler := workflow.NewCompiler(verbose, "", "")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Workflow compilation failed: %v", err)))
		fmt.Println(console.FormatInfoMessage("You can fix the issues and run 'gh aw compile' manually"))
	} else {
		fmt.Println(console.FormatSuccessMessage("Workflow compiled successfully"))
	}

	return nil
}

// cleanMCPToolID removes common MCP prefixes and suffixes from tool IDs
// Examples: "notion-mcp" -> "notion", "mcp-notion" -> "notion", "some-mcp-server" -> "some-server"
func cleanMCPToolID(toolID string) string {
	cleaned := toolID

	// Remove "mcp-" prefix
	cleaned = strings.TrimPrefix(cleaned, "mcp-")

	// Remove "-mcp" suffix
	cleaned = strings.TrimSuffix(cleaned, "-mcp")

	// If the result is empty, use the original
	if cleaned == "" {
		return toolID
	}

	return cleaned
}

// createMCPToolConfig creates the MCP tool configuration based on registry server info
func createMCPToolConfig(server *MCPRegistryServer, preferredTransport string, registryURL string, verbose bool) (map[string]any, error) {
	config := make(map[string]any)

	// Determine transport type (use preference if provided and supported)
	transport := server.Transport
	if preferredTransport != "" {
		switch preferredTransport {
		case "stdio", "http", "docker":
			transport = preferredTransport
			if verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Using preferred transport: %s", transport)))
			}
		default:
			return nil, fmt.Errorf("unsupported transport type: %s (supported: stdio, http, docker)", preferredTransport)
		}
	}

	// Create MCP configuration based on transport type
	mcpSection := map[string]any{
		"type":     transport,
		"registry": fmt.Sprintf("%s/servers/%s", registryURL, server.ID),
	}

	switch transport {
	case "stdio":
		// Handle container field (simplified Docker run)
		if server.Config != nil {
			if container, hasContainer := server.Config["container"]; hasContainer {
				if containerStr, ok := container.(string); ok {
					mcpSection["container"] = containerStr

					// Add environment variables for Docker container
					if env, hasEnv := server.Config["env"]; hasEnv {
						mcpSection["env"] = convertToGitHubActionsEnv(env)
					}
				}
			} else {
				// Handle regular command and args
				if server.Command != "" {
					mcpSection["command"] = server.Command
				}
				if len(server.Args) > 0 {
					mcpSection["args"] = server.Args
				}

				// Add environment variables if present
				if env, hasEnv := server.Config["env"]; hasEnv {
					mcpSection["env"] = convertToGitHubActionsEnv(env)
				}
			}
		} else {
			// Handle command and args when no config
			if server.Command != "" {
				mcpSection["command"] = server.Command
			}
			if len(server.Args) > 0 {
				mcpSection["args"] = server.Args
			}
		}

	case "http":
		// For HTTP transport, we need a URL
		if server.Config != nil {
			if url, hasURL := server.Config["url"]; hasURL {
				mcpSection["url"] = url
			} else {
				return nil, fmt.Errorf("HTTP transport requires URL configuration")
			}

			// Add headers if present
			if headers, hasHeaders := server.Config["headers"]; hasHeaders {
				mcpSection["headers"] = headers
			}
		} else {
			return nil, fmt.Errorf("HTTP transport requires configuration")
		}

	case "docker":
		// For Docker transport, use container configuration
		if server.Config != nil {
			if container, hasContainer := server.Config["container"]; hasContainer {
				mcpSection["container"] = container
			} else {
				return nil, fmt.Errorf("docker transport requires container configuration")
			}

			// Add environment variables if present
			if env, hasEnv := server.Config["env"]; hasEnv {
				mcpSection["env"] = convertToGitHubActionsEnv(env)
			}
		} else {
			return nil, fmt.Errorf("docker transport requires configuration")
		}

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transport)
	}

	config["mcp"] = mcpSection

	return config, nil
}

// addToolToWorkflow adds a tool configuration to the workflow file
func addToolToWorkflow(workflowPath string, toolID string, toolConfig map[string]any, verbose bool) error {
	// Use frontmatter helper to update the workflow file
	return parser.UpdateWorkflowFrontmatter(workflowPath, func(frontmatter map[string]any) error {
		// Ensure tools section exists
		tools := parser.EnsureToolsSection(frontmatter)

		// Check if tool already exists
		if _, exists := tools[toolID]; exists {
			return fmt.Errorf("tool '%s' already exists in workflow", toolID)
		}

		// Add the new tool
		tools[toolID] = toolConfig
		return nil
	}, verbose)
}

// NewMCPAddSubcommand creates the mcp add subcommand
func NewMCPAddSubcommand() *cobra.Command {
	var registryURL string
	var transportType string
	var customToolID string

	cmd := &cobra.Command{
		Use:   "add [workflow-file] [mcp-server-name]",
		Short: "Add an MCP tool to an agentic workflow",
		Long: `Add an MCP tool to an agentic workflow by searching the MCP registry.

This command searches the MCP registry for the specified server, adds it to the workflow's tools section,
and automatically compiles the workflow. If the tool already exists, the command will fail.

When called with no arguments, it will show a list of available MCP servers from the registry.

Examples:
  gh aw mcp add                                          # List available MCP servers
  gh aw mcp add weekly-research makenotion/notion-mcp-server  # Add Notion MCP server to weekly-research.md
  gh aw mcp add weekly-research makenotion/notion-mcp-server --transport stdio  # Prefer stdio transport
  gh aw mcp add weekly-research makenotion/notion-mcp-server --registry https://custom.registry.com/v1  # Use custom registry
  gh aw mcp add weekly-research makenotion/notion-mcp-server --tool-id my-notion  # Use custom tool ID

The command will:
- Search the MCP registry for the specified server
- Check that the tool doesn't already exist in the workflow
- Add the MCP tool configuration to the workflow's frontmatter
- Automatically compile the workflow to generate the .lock.yml file

Registry URL defaults to: https://api.mcp.github.com/v0`,
		Args: cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Inherit verbose from parent commands
			if !verbose {
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
			}

			// If no arguments provided, show list of available servers
			if len(args) == 0 {
				// Use default registry URL if not provided
				if registryURL == "" {
					registryURL = "https://api.mcp.github.com/v0"
				}
				return listAvailableServers(registryURL, verbose)
			}

			// If only workflow file is provided, show error (need both workflow and server)
			if len(args) == 1 {
				return fmt.Errorf("both workflow file and server name are required to add an MCP tool\nUse 'gh aw mcp add' to list available servers")
			}

			// If both arguments are provided, add the MCP tool
			workflowFile := args[0]
			mcpServerID := args[1]

			return AddMCPTool(workflowFile, mcpServerID, registryURL, transportType, customToolID, verbose)
		},
	}

	cmd.Flags().StringVar(&registryURL, "registry", "", "MCP registry URL (default: https://api.mcp.github.com/v0)")
	cmd.Flags().StringVar(&transportType, "transport", "", "Preferred transport type (stdio, http, docker)")
	cmd.Flags().StringVar(&customToolID, "tool-id", "", "Custom tool ID to use in the workflow (default: uses server ID)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}
