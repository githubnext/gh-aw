package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// NewMCPCommand creates the mcp command with subcommands
func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP tools in agentic workflows",
		Long: `Manage Model Context Protocol (MCP) tools in agentic workflows.

This command provides subcommands to add, list, and inspect MCP servers
configured in workflow files. You can manage tool configurations, allowed
tool lists, and inspect available capabilities from MCP servers.

Examples:
  gh aw mcp list weekly-research              # List MCP tools in workflow
  gh aw mcp add weekly-research github        # Add GitHub MCP server
  gh aw mcp inspect weekly-research           # Inspect MCP servers in workflow
  gh aw mcp inspect weekly-research --server github  # Inspect specific server`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewMCPAddCommand())
	cmd.AddCommand(NewMCPListCommand())
	cmd.AddCommand(NewMCPInspectCommand())

	return cmd
}

// NewMCPAddCommand creates the mcp add command
func NewMCPAddCommand() *cobra.Command {
	var forceFlag bool
	var allowedTools []string

	cmd := &cobra.Command{
		Use:   "add <workflow-id> <tool-name> [tool-config-args...]",
		Short: "Add MCP tools to an agentic workflow",
		Long: `Add Model Context Protocol (MCP) tools to an agentic workflow.

This command adds MCP tool configurations to the frontmatter of a workflow file.
You can add built-in tools (github, playwright) or custom MCP servers.

Built-in tools:
  github       - GitHub MCP server for repository operations
  playwright   - Playwright MCP server for browser automation

Custom tools require additional configuration parameters.

Examples:
  gh aw mcp add weekly-research github              # Add GitHub MCP server
  gh aw mcp add weekly-research playwright          # Add Playwright MCP server
  gh aw mcp add weekly-research github --allowed create_issue,list_repos
  gh aw mcp add weekly-research custom-tool --type stdio --command python --args="-m,server"
  gh aw mcp add weekly-research api-server --type http --url https://api.example.com/mcp`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]
			toolName := args[1]
			toolConfigArgs := args[2:]

			verbose, _ := cmd.Flags().GetBool("verbose")
			if cmd.Parent() != nil && cmd.Parent().Parent() != nil {
				parentVerbose, _ := cmd.Parent().Parent().PersistentFlags().GetBool("verbose")
				verbose = verbose || parentVerbose
			}

			return AddMCPToolWithFlags(workflowID, toolName, toolConfigArgs, cmd, verbose)
		},
	}

	cmd.Flags().BoolVar(&forceFlag, "force", false, "Overwrite existing tool configuration")
	cmd.Flags().StringSliceVar(&allowedTools, "allowed", nil, "Comma-separated list of allowed tools for this MCP server")
	cmd.Flags().StringP("type", "t", "", "MCP server type (stdio, http) for custom tools")
	cmd.Flags().StringP("command", "c", "", "Command to run for stdio MCP servers")
	cmd.Flags().StringSliceP("args", "a", nil, "Command arguments for stdio MCP servers")
	cmd.Flags().StringP("url", "u", "", "URL for HTTP MCP servers")
	cmd.Flags().StringP("container", "", "", "Docker container for stdio MCP servers")
	cmd.Flags().StringToStringP("env", "e", nil, "Environment variables (key=value)")
	cmd.Flags().StringToStringP("headers", "", nil, "HTTP headers for HTTP MCP servers (key=value)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}

// NewMCPListCommand creates the mcp list command
func NewMCPListCommand() *cobra.Command {
	var showAllowedTools bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list <workflow-id>",
		Short: "List MCP tools in an agentic workflow",
		Long: `List Model Context Protocol (MCP) tools configured in an agentic workflow.

This command displays all MCP servers configured in the specified workflow file,
including their types, configurations, and allowed tools.

Examples:
  gh aw mcp list weekly-research                    # List all MCP tools
  gh aw mcp list weekly-research --show-allowed     # Include allowed tools list
  gh aw mcp list weekly-research --format json      # Output in JSON format`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]

			verbose, _ := cmd.Flags().GetBool("verbose")
			if cmd.Parent() != nil && cmd.Parent().Parent() != nil {
				parentVerbose, _ := cmd.Parent().Parent().PersistentFlags().GetBool("verbose")
				verbose = verbose || parentVerbose
			}

			return ListMCPTools(workflowID, showAllowedTools, outputFormat, verbose)
		},
	}

	cmd.Flags().BoolVar(&showAllowedTools, "show-allowed", false, "Show allowed tools for each MCP server")
	cmd.Flags().StringVar(&outputFormat, "format", "table", "Output format (table, json)")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}

// AddMCPToolWithFlags adds an MCP tool using flags from the cobra command
func AddMCPToolWithFlags(workflowID, toolName string, toolConfigArgs []string, cmd *cobra.Command, verbose bool) error {
	workflowPath, err := getWorkflowPath(workflowID)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Adding MCP tool '%s' to workflow: %s", toolName, workflowPath)))
	}

	// Extract flags
	allowedTools, _ := cmd.Flags().GetStringSlice("allowed")
	forceFlag, _ := cmd.Flags().GetBool("force")
	mcpType, _ := cmd.Flags().GetString("type")
	command, _ := cmd.Flags().GetString("command")
	args, _ := cmd.Flags().GetStringSlice("args")
	url, _ := cmd.Flags().GetString("url")
	container, _ := cmd.Flags().GetString("container")
	env, _ := cmd.Flags().GetStringToString("env")
	headers, _ := cmd.Flags().GetStringToString("headers")

	// Read existing workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse frontmatter
	workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse workflow frontmatter: %w", err)
	}

	// Get or create tools section
	toolsSection, hasTools := workflowData.Frontmatter["tools"]
	var tools map[string]any
	if hasTools {
		if toolsMap, ok := toolsSection.(map[string]any); ok {
			tools = toolsMap
		} else {
			return fmt.Errorf("tools section is not a valid map")
		}
	} else {
		tools = make(map[string]any)
		workflowData.Frontmatter["tools"] = tools
	}

	// Check if tool already exists
	if _, exists := tools[toolName]; exists && !forceFlag {
		return fmt.Errorf("tool '%s' already exists in workflow (use --force to overwrite)", toolName)
	}

	// Create tool configuration
	toolConfig, err := createToolConfigWithFlags(toolName, mcpType, command, args, url, container, env, headers, allowedTools)
	if err != nil {
		return fmt.Errorf("failed to create tool configuration: %w", err)
	}

	// Add tool to configuration
	tools[toolName] = toolConfig

	if verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Successfully added MCP tool '%s' to workflow", toolName)))
	}

	// Write updated frontmatter back to file
	return writeFrontmatterToFile(workflowPath, workflowData)
}

// AddMCPTool adds an MCP tool to a workflow file (legacy function for compatibility)
func AddMCPTool(workflowID, toolName string, toolConfigArgs, allowedTools []string, force, verbose bool) error {
	workflowPath, err := getWorkflowPath(workflowID)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Adding MCP tool '%s' to workflow: %s", toolName, workflowPath)))
	}

	// Read existing workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse frontmatter
	workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse workflow frontmatter: %w", err)
	}

	// Get or create tools section
	toolsSection, hasTools := workflowData.Frontmatter["tools"]
	var tools map[string]any
	if hasTools {
		if toolsMap, ok := toolsSection.(map[string]any); ok {
			tools = toolsMap
		} else {
			return fmt.Errorf("tools section is not a valid map")
		}
	} else {
		tools = make(map[string]any)
		workflowData.Frontmatter["tools"] = tools
	}

	// Check if tool already exists
	if _, exists := tools[toolName]; exists && !force {
		return fmt.Errorf("tool '%s' already exists in workflow (use --force to overwrite)", toolName)
	}

	// Create tool configuration based on tool type
	toolConfig, err := createToolConfig(toolName, toolConfigArgs, allowedTools)
	if err != nil {
		return fmt.Errorf("failed to create tool configuration: %w", err)
	}

	// Add tool to configuration
	tools[toolName] = toolConfig

	if verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Successfully added MCP tool '%s' to workflow", toolName)))
	}

	// Write updated frontmatter back to file
	return writeFrontmatterToFile(workflowPath, workflowData)
}

// ListMCPTools lists MCP tools in a workflow
func ListMCPTools(workflowID string, showAllowedTools bool, outputFormat string, verbose bool) error {
	workflowPath, err := getWorkflowPath(workflowID)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Listing MCP tools in workflow: %s", workflowPath)))
	}

	// Read and parse workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	workflowData, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse workflow frontmatter: %w", err)
	}

	// Extract MCP configurations
	mcpConfigs, err := parser.ExtractMCPConfigurations(workflowData.Frontmatter, "")
	if err != nil {
		return fmt.Errorf("failed to extract MCP configurations: %w", err)
	}

	if len(mcpConfigs) == 0 {
		fmt.Println(console.FormatInfoMessage("No MCP tools found in workflow"))
		return nil
	}

	// Display results based on format
	switch outputFormat {
	case "json":
		return displayMCPToolsJSON(mcpConfigs)
	case "table":
		return displayMCPToolsTable(mcpConfigs, showAllowedTools)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

// Helper functions

func getWorkflowPath(workflowID string) (string, error) {
	workflowsDir := getWorkflowsDir()

	// Handle .md extension
	if !strings.HasSuffix(workflowID, ".md") {
		workflowID += ".md"
	}

	workflowPath := filepath.Join(workflowsDir, workflowID)

	// Check if file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		return "", fmt.Errorf("workflow file not found: %s", workflowPath)
	}

	return workflowPath, nil
}

func createToolConfigWithFlags(toolName, mcpType, command string, args []string, url, container string, env, headers map[string]string, allowedTools []string) (map[string]any, error) {
	// Handle built-in tools
	switch toolName {
	case "github":
		config := map[string]any{}
		if len(allowedTools) > 0 {
			config["allowed"] = allowedTools
		}
		return config, nil

	case "playwright":
		config := map[string]any{}
		if len(allowedTools) > 0 {
			config["allowed"] = allowedTools
		}
		return config, nil

	default:
		// Handle custom MCP tools
		if mcpType == "" {
			return nil, fmt.Errorf("custom MCP tool '%s' requires --type flag (stdio or http)", toolName)
		}

		config := map[string]any{}

		// Add allowed tools if specified
		if len(allowedTools) > 0 {
			config["allowed"] = allowedTools
		}

		// Create MCP configuration
		mcpConfig := map[string]any{
			"type": mcpType,
		}

		switch mcpType {
		case "stdio":
			if container != "" {
				mcpConfig["container"] = container
				if len(env) > 0 {
					mcpConfig["env"] = env
				}
			} else {
				if command == "" {
					return nil, fmt.Errorf("stdio MCP tool requires --command or --container flag")
				}
				mcpConfig["command"] = command
				if len(args) > 0 {
					mcpConfig["args"] = args
				}
				if len(env) > 0 {
					mcpConfig["env"] = env
				}
			}

		case "http":
			if url == "" {
				return nil, fmt.Errorf("http MCP tool requires --url flag")
			}
			mcpConfig["url"] = url
			if len(headers) > 0 {
				mcpConfig["headers"] = headers
			}

		default:
			return nil, fmt.Errorf("unsupported MCP type: %s (supported: stdio, http)", mcpType)
		}

		config["mcp"] = mcpConfig
		return config, nil
	}
}

func createToolConfig(toolName string, toolConfigArgs, allowedTools []string) (map[string]any, error) {
	// Handle built-in tools
	switch toolName {
	case "github":
		config := map[string]any{}
		if len(allowedTools) > 0 {
			config["allowed"] = allowedTools
		}
		return config, nil

	case "playwright":
		config := map[string]any{}
		if len(allowedTools) > 0 {
			config["allowed"] = allowedTools
		}
		return config, nil

	default:
		// Custom MCP tools require additional configuration
		return nil, fmt.Errorf("custom MCP tool '%s' requires additional configuration (use --type, --command, --url, etc.)", toolName)
	}
}

func displayMCPToolsJSON(configs []parser.MCPServerConfig) error {
	// Convert configs to a JSON-serializable format
	type JsonMCPTool struct {
		Name      string            `json:"name"`
		Type      string            `json:"type"`
		Command   string            `json:"command,omitempty"`
		Args      []string          `json:"args,omitempty"`
		Container string            `json:"container,omitempty"`
		URL       string            `json:"url,omitempty"`
		Headers   map[string]string `json:"headers,omitempty"`
		Env       map[string]string `json:"env,omitempty"`
		Allowed   []string          `json:"allowed,omitempty"`
	}

	var jsonTools []JsonMCPTool
	for _, config := range configs {
		tool := JsonMCPTool{
			Name:      config.Name,
			Type:      config.Type,
			Command:   config.Command,
			Args:      config.Args,
			Container: config.Container,
			URL:       config.URL,
			Headers:   config.Headers,
			Env:       config.Env,
			Allowed:   config.Allowed,
		}
		jsonTools = append(jsonTools, tool)
	}

	jsonBytes, err := json.MarshalIndent(jsonTools, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func displayMCPToolsTable(configs []parser.MCPServerConfig, showAllowed bool) error {
	fmt.Println(console.FormatListHeader("MCP Tools"))
	fmt.Println(console.FormatListHeader("========="))

	for _, config := range configs {
		fmt.Printf("• %s (%s)\n", config.Name, config.Type)

		if config.Type == "stdio" {
			if config.Container != "" {
				fmt.Printf("  Container: %s\n", config.Container)
			} else {
				fmt.Printf("  Command: %s\n", config.Command)
				if len(config.Args) > 0 {
					fmt.Printf("  Args: %s\n", strings.Join(config.Args, " "))
				}
			}
		} else if config.Type == "http" {
			fmt.Printf("  URL: %s\n", config.URL)
		}

		if showAllowed && len(config.Allowed) > 0 {
			fmt.Printf("  Allowed: %s\n", strings.Join(config.Allowed, ", "))
		}
		fmt.Println()
	}

	return nil
}

func writeFrontmatterToFile(workflowPath string, workflowData *parser.FrontmatterResult) error {
	// Convert frontmatter back to YAML
	yamlBytes, err := yaml.Marshal(workflowData.Frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Construct the full file content
	var content strings.Builder
	content.WriteString("---\n")
	content.WriteString(string(yamlBytes))
	content.WriteString("---\n\n")
	content.WriteString(workflowData.Markdown)

	// Write to file
	if err := os.WriteFile(workflowPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}
