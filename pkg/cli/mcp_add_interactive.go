package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var mcpAddInteractiveLog = logger.New("cli:mcp_add_interactive")

// MCPAddInteractiveConfig holds the interactive form input values
type MCPAddInteractiveConfig struct {
	WorkflowFile    string
	ServerName      string
	TransportType   string
	CustomToolID    string
	AllowNetwork    bool
	NetworkDomains  []string
	EnvironmentVars map[string]string
}

// AddMCPToolInteractively prompts the user to add an MCP tool using interactive forms
func AddMCPToolInteractively(workflowFile string, registryURL string, verbose bool) error {
	mcpAddInteractiveLog.Printf("Starting interactive MCP tool addition: workflowFile=%s", workflowFile)

	// Assert this function is not running in automated unit tests
	if os.Getenv("GO_TEST_MODE") == "true" || os.Getenv("CI") != "" {
		return fmt.Errorf("interactive MCP configuration cannot be used in automated tests or CI environments")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting interactive MCP server configuration..."))
	}

	// Resolve the workflow file path
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		mcpAddInteractiveLog.Printf("Failed to resolve workflow path: %v", err)
		return err
	}
	mcpAddInteractiveLog.Printf("Resolved workflow path: %s", workflowPath)

	// Create registry client
	if registryURL == "" {
		registryURL = string(constants.DefaultMCPRegistryURL)
	}
	registryClient := NewMCPRegistryClient(registryURL)

	// Fetch available servers from registry
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching MCP servers from registry: %s", registryClient.registryURL)))
	}

	servers, err := registryClient.SearchServers("")
	if err != nil {
		return fmt.Errorf("failed to fetch MCP servers from registry: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no MCP servers found in registry")
	}

	mcpAddInteractiveLog.Printf("Fetched %d servers from registry", len(servers))

	// Prepare server options for selection
	serverOptions := make([]huh.Option[string], 0, len(servers))
	for _, server := range servers {
		label := fmt.Sprintf("%s - %s", server.Name, server.Description)
		serverOptions = append(serverOptions, huh.NewOption(label, server.Name))
	}

	// Initialize config
	config := &MCPAddInteractiveConfig{
		WorkflowFile:    workflowPath,
		AllowNetwork:    false,
		NetworkDomains:  []string{},
		EnvironmentVars: make(map[string]string),
	}

	// Run through the interactive prompts
	if err := promptForMCPConfiguration(config, serverOptions, verbose); err != nil {
		return fmt.Errorf("failed to get MCP configuration: %w", err)
	}

	// Find the selected server
	var selectedServer *MCPRegistryServerForProcessing
	for i, server := range servers {
		if server.Name == config.ServerName {
			selectedServer = &servers[i]
			break
		}
	}

	if selectedServer == nil {
		return fmt.Errorf("selected server '%s' not found in registry", config.ServerName)
	}

	// Determine tool ID
	toolID := cleanMCPToolID(selectedServer.Name)
	if config.CustomToolID != "" {
		toolID = config.CustomToolID
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Adding MCP server '%s' as tool '%s'", selectedServer.Name, toolID)))
	}

	// Create MCP tool configuration
	mcpConfig, err := createMCPToolConfig(selectedServer, config.TransportType, registryClient.registryURL, verbose)
	if err != nil {
		return fmt.Errorf("failed to create MCP tool configuration: %w", err)
	}

	// Add the tool to the workflow
	if err := addToolToWorkflow(config.WorkflowFile, toolID, mcpConfig, verbose); err != nil {
		return fmt.Errorf("failed to add tool to workflow: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Added MCP tool '%s' to workflow %s", toolID, console.ToRelativePath(config.WorkflowFile))))

	// Check for required secrets and provide CLI commands if missing
	if err := checkAndSuggestSecrets(mcpConfig, verbose); err != nil {
		// Don't fail the command if secret checking fails, just log a warning
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Could not check repository secrets: %v", err)))
		}
	}

	// Compile the workflow
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling workflow..."))
	}

	return compileWorkflowAfterMCPAdd(config.WorkflowFile, verbose)
}

// promptForMCPConfiguration organizes all prompts into logical groups
func promptForMCPConfiguration(config *MCPAddInteractiveConfig, serverOptions []huh.Option[string], verbose bool) error {
	mcpAddInteractiveLog.Printf("Starting interactive prompts with %d server options", len(serverOptions))

	// Prepare transport type options
	transportOptions := []huh.Option[string]{
		huh.NewOption("stdio - Standard input/output (local process)", "stdio"),
		huh.NewOption("http - HTTP/REST API connection", "http"),
		huh.NewOption("docker - Docker container execution", "docker"),
	}

	// Variables to hold form results
	var serverName string
	var transportType string
	var customToolID string
	var allowNetwork bool

	// Create multi-page form
	form := huh.NewForm(
		// Page 1: Server Selection
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select MCP server from registry").
				Description("Choose the MCP server you want to add to your workflow").
				Options(serverOptions...).
				Height(10).
				Value(&serverName).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("please select an MCP server")
					}
					return nil
				}),
		).
			Title("Server Selection").
			Description("Select the MCP server to add to your workflow"),

		// Page 2: Configuration
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select transport type").
				Description("How should the workflow connect to this MCP server?").
				Options(transportOptions...).
				Value(&transportType),
			huh.NewInput().
				Title("Custom tool ID (optional)").
				Description("Provide a custom identifier for this tool in the workflow (leave empty to use default)").
				Value(&customToolID).
				Placeholder("e.g., my-custom-tool-name"),
		).
			Title("Transport & Identity").
			Description("Configure how the workflow will connect to the MCP server"),

		// Page 3: Network & Security
		huh.NewGroup(
			huh.NewConfirm().
				Title("Allow network access?").
				Description("Does this MCP server need to access external networks?").
				Value(&allowNetwork),
		).
			Title("Network & Security").
			Description("Configure network access and security settings"),
	).WithAccessible(isAccessibleMode())

	if err := form.Run(); err != nil {
		return err
	}

	// Store the results
	config.ServerName = serverName
	config.TransportType = transportType
	config.CustomToolID = customToolID
	config.AllowNetwork = allowNetwork

	mcpAddInteractiveLog.Printf("Interactive prompts completed: server=%s, transport=%s", serverName, transportType)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Configuration complete: server=%s, transport=%s", serverName, transportType)))
	}

	return nil
}

// compileWorkflowAfterMCPAdd compiles the workflow after adding an MCP tool
func compileWorkflowAfterMCPAdd(workflowPath string, verbose bool) error {
	mcpAddInteractiveLog.Print("Compiling workflow after adding MCP tool")

	// Create spinner for compilation progress
	spinner := console.NewSpinner("Compiling your workflow...")
	spinner.Start()

	// Use the existing compile functionality from workflow package
	// This uses the same approach as AddMCPTool in mcp_add.go
	compiler := workflow.NewCompiler(verbose, "", "")
	err := compiler.CompileWorkflow(workflowPath)

	if err != nil {
		spinner.Stop()
		// Security fix for CWE-312, CWE-315, CWE-359: Avoid logging detailed error messages
		// that could contain sensitive information from secret references
		mcpAddInteractiveLog.Print("Workflow compilation failed")
		fmt.Println(console.FormatWarningMessage("Workflow compilation failed. Please check your workflow configuration."))
		fmt.Println(console.FormatInfoMessage("You can fix the issues and run 'gh aw compile' manually"))
		return err
	}

	// Stop spinner with success message
	spinner.StopWithMessage("✓ Workflow compiled successfully!")
	mcpAddInteractiveLog.Print("Workflow compiled successfully")
	fmt.Println(console.FormatSuccessMessage("Workflow compiled successfully"))

	return nil
}
