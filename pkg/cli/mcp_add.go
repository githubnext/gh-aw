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
	"gopkg.in/yaml.v3"
)

// AddMCPTool adds an MCP tool to an agentic workflow
func AddMCPTool(workflowFile string, mcpServerID string, registryURL string, transportType string, customToolID string, verbose bool) error {
	// Determine the workflow directory and file path
	workflowsDir := ".github/workflows"
	var workflowPath string

	if strings.HasSuffix(workflowFile, ".md") {
		// If it's already a .md file, use it directly if it exists
		if _, err := os.Stat(workflowFile); err == nil {
			workflowPath = workflowFile
		} else {
			// Try in workflows directory
			workflowPath = filepath.Join(workflowsDir, workflowFile)
		}
	} else {
		// Add .md extension and look in workflows directory
		workflowPath = filepath.Join(workflowsDir, workflowFile+".md")
	}

	// Verify the workflow file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		return fmt.Errorf("workflow file not found: %s", workflowPath)
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

	// Find exact match or best match
	var selectedServer *MCPRegistryServer
	for i, server := range servers {
		if server.ID == mcpServerID || server.Name == mcpServerID {
			selectedServer = &servers[i]
			break
		}
	}

	// If no exact match, use the first result
	if selectedServer == nil {
		selectedServer = &servers[0]
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("No exact match for '%s', using closest match: %s", mcpServerID, selectedServer.Name)))
		}
	}

	// Determine tool ID (use custom if provided, otherwise use server ID)
	toolID := selectedServer.ID
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
	mcpConfig, err := createMCPToolConfig(selectedServer, transportType, verbose)
	if err != nil {
		return fmt.Errorf("failed to create MCP tool configuration: %w", err)
	}

	// Add the tool to the workflow
	if err := addToolToWorkflow(workflowPath, toolID, mcpConfig, verbose); err != nil {
		return fmt.Errorf("failed to add tool to workflow: %w", err)
	}

	fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Added MCP tool '%s' to workflow %s", toolID, console.ToRelativePath(workflowPath))))

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

// convertToGitHubActionsEnv converts environment variables from shell syntax to GitHub Actions syntax
// Converts "${TOKEN_NAME}" to "${{ secrets.TOKEN_NAME }}"
// Leaves existing GitHub Actions syntax unchanged
func convertToGitHubActionsEnv(env interface{}) map[string]string {
	result := make(map[string]string)

	if envMap, ok := env.(map[string]interface{}); ok {
		for key, value := range envMap {
			if valueStr, ok := value.(string); ok {
				// Only convert shell syntax ${TOKEN_NAME}, leave GitHub Actions syntax unchanged
				if strings.HasPrefix(valueStr, "${") && strings.HasSuffix(valueStr, "}") && !strings.Contains(valueStr, "{{") {
					tokenName := valueStr[2 : len(valueStr)-1] // Remove ${ and }
					result[key] = fmt.Sprintf("${{ secrets.%s }}", tokenName)
				} else {
					// Keep as-is if not shell syntax or already GitHub Actions syntax
					result[key] = valueStr
				}
			}
		}
	}

	return result
}

// createMCPToolConfig creates the MCP tool configuration based on registry server info
func createMCPToolConfig(server *MCPRegistryServer, preferredTransport string, verbose bool) (map[string]any, error) {
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
		"type": transport,
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
				return nil, fmt.Errorf("Docker transport requires container configuration")
			}

			// Add environment variables if present
			if env, hasEnv := server.Config["env"]; hasEnv {
				mcpSection["env"] = convertToGitHubActionsEnv(env)
			}
		} else {
			return nil, fmt.Errorf("Docker transport requires configuration")
		}

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transport)
	}

	config["mcp"] = mcpSection

	return config, nil
}

// addToolToWorkflow adds a tool configuration to the workflow file
func addToolToWorkflow(workflowPath string, toolID string, toolConfig map[string]any, verbose bool) error {
	// Read the file content
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse YAML frontmatter and markdown content
	fileContent := string(content)

	// Find frontmatter boundaries
	lines := strings.Split(fileContent, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return fmt.Errorf("workflow file does not have valid YAML frontmatter")
	}

	// Find the end of frontmatter
	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			frontmatterEnd = i
			break
		}
	}

	if frontmatterEnd == -1 {
		return fmt.Errorf("workflow file frontmatter is not properly closed")
	}

	// Extract frontmatter YAML
	frontmatterLines := lines[1:frontmatterEnd]
	frontmatterYAML := strings.Join(frontmatterLines, "\n")

	// Parse the frontmatter
	var frontmatter map[string]any
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &frontmatter); err != nil {
		return fmt.Errorf("failed to parse frontmatter YAML: %w", err)
	}

	// Ensure tools section exists
	if frontmatter["tools"] == nil {
		frontmatter["tools"] = make(map[string]any)
	}

	tools, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return fmt.Errorf("tools section is not a valid map")
	}

	// Add the new tool
	tools[toolID] = toolConfig

	// Convert back to YAML
	updatedFrontmatter, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}

	// Reconstruct the file
	var newLines []string
	newLines = append(newLines, "---")

	// Add the updated frontmatter (trim the trailing newline from Marshal)
	frontmatterStr := strings.TrimSuffix(string(updatedFrontmatter), "\n")
	newLines = append(newLines, strings.Split(frontmatterStr, "\n")...)

	newLines = append(newLines, "---")

	// Add the remaining content (markdown)
	if frontmatterEnd+1 < len(lines) {
		newLines = append(newLines, lines[frontmatterEnd+1:]...)
	}

	// Write the updated content back to the file
	updatedContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(workflowPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow file: %w", err)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Updated workflow file: %s", console.ToRelativePath(workflowPath))))
	}

	return nil
}

// NewMCPAddSubcommand creates the mcp add subcommand
func NewMCPAddSubcommand() *cobra.Command {
	var registryURL string
	var transportType string
	var customToolID string

	cmd := &cobra.Command{
		Use:   "add <workflow-file> <mcp-server-id>",
		Short: "Add an MCP tool to an agentic workflow",
		Long: `Add an MCP tool to an agentic workflow by searching the MCP registry.

This command searches the MCP registry for the specified server, adds it to the workflow's tools section,
and automatically compiles the workflow. If the tool already exists, the command will fail.

Examples:
  gh aw mcp add weekly-research notion        # Add Notion MCP server to weekly-research.md
  gh aw mcp add weekly-research notion --transport stdio  # Prefer stdio transport
  gh aw mcp add weekly-research notion --registry https://custom.registry.com/v1  # Use custom registry
  gh aw mcp add weekly-research notion --tool-id my-notion  # Use custom tool ID

The command will:
- Search the MCP registry for the specified server
- Check that the tool doesn't already exist in the workflow
- Add the MCP tool configuration to the workflow's frontmatter
- Automatically compile the workflow to generate the .lock.yml file

Registry URL defaults to: https://api.mcp.github.com/v0`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowFile := args[0]
			mcpServerID := args[1]

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

			return AddMCPTool(workflowFile, mcpServerID, registryURL, transportType, customToolID, verbose)
		},
	}

	cmd.Flags().StringVar(&registryURL, "registry", "", "MCP registry URL (default: https://api.mcp.github.com/v0)")
	cmd.Flags().StringVar(&transportType, "transport", "", "Preferred transport type (stdio, http, docker)")
	cmd.Flags().StringVar(&customToolID, "tool-id", "", "Custom tool ID to use in the workflow (default: uses server ID)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}
