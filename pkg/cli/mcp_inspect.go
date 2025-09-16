package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

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

	// Parse and spawn MCP servers from the generated configuration
	if err := spawnMCPServersFromConfig(mcpConfigBuilder.String(), verbose); err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to spawn MCP servers: %v", err)))
		}
	}

	return nil
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// MCPConfig represents the complete MCP configuration
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// spawnMCPServersFromConfig parses the generated MCP configuration and spawns servers
func spawnMCPServersFromConfig(configScript string, verbose bool) error {
	// Extract JSON from the generated shell script
	jsonConfig, err := extractJSONFromScript(configScript)
	if err != nil {
		return fmt.Errorf("failed to extract JSON from configuration script: %w", err)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage("Extracted MCP JSON configuration:"))
		fmt.Println(jsonConfig)
		fmt.Println()
	}

	// Replace GitHub Actions template variables with actual environment values
	resolvedConfig := resolveTemplateVariables(jsonConfig, verbose)

	if verbose {
		fmt.Println(console.FormatInfoMessage("Resolved MCP JSON configuration:"))
		fmt.Println(resolvedConfig)
		fmt.Println()
	}

	// Parse the JSON configuration
	var config MCPConfig
	if err := json.Unmarshal([]byte(resolvedConfig), &config); err != nil {
		return fmt.Errorf("failed to parse MCP configuration JSON: %w", err)
	}

	if len(config.MCPServers) == 0 {
		if verbose {
			fmt.Println(console.FormatInfoMessage("No MCP servers found in configuration to spawn"))
		}
		return nil
	}

	fmt.Println()
	fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Spawning %d MCP server(s) from generated configuration...", len(config.MCPServers))))

	var wg sync.WaitGroup
	var serverProcesses []*exec.Cmd

	// Start each server
	for serverName, serverConfig := range config.MCPServers {
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Starting MCP server: %s", serverName)))
		}

		// Create the command
		cmd := exec.Command(serverConfig.Command, serverConfig.Args...)
		
		// Set environment variables
		cmd.Env = os.Environ()
		for key, value := range serverConfig.Env {
			// Resolve environment variable references (simple implementation)
			resolvedValue := os.ExpandEnv(value)
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, resolvedValue))
		}

		// Start the server process
		if err := cmd.Start(); err != nil {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to start server %s: %v", serverName, err)))
			continue
		}

		serverProcesses = append(serverProcesses, cmd)

		// Monitor the process in the background
		wg.Add(1)
		go func(serverCmd *exec.Cmd, name string) {
			defer wg.Done()
			if err := serverCmd.Wait(); err != nil && verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Server %s exited with error: %v", name, err)))
			}
		}(cmd, serverName)

		if verbose {
			fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Started server: %s (PID: %d)", serverName, cmd.Process.Pid)))
		}
	}

	if len(serverProcesses) > 0 {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Successfully started %d MCP server(s)", len(serverProcesses))))
		fmt.Println(console.FormatInfoMessage("Servers are running in the background"))
		fmt.Println(console.FormatInfoMessage("Press Ctrl+C to stop the inspection and cleanup servers"))

		// Set up cleanup on interrupt
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		
		go func() {
			<-c
			fmt.Println()
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
			
			os.Exit(0)
		}()

		// Keep the main process alive to maintain servers
		select {}
	}

	return nil
}

// resolveTemplateVariables replaces GitHub Actions template variables with local environment values
func resolveTemplateVariables(jsonConfig string, verbose bool) string {
	// Replace common GitHub Actions template variables with environment values or defaults
	resolved := jsonConfig
	
	// Replace ${{ env.GITHUB_AW_SAFE_OUTPUTS }} with environment value or default
	if safeOutputs := os.Getenv("GITHUB_AW_SAFE_OUTPUTS"); safeOutputs != "" {
		resolved = strings.ReplaceAll(resolved, `"${{ env.GITHUB_AW_SAFE_OUTPUTS }}"`, fmt.Sprintf(`"%s"`, safeOutputs))
	} else {
		// Default to a temporary file for local testing
		resolved = strings.ReplaceAll(resolved, `"${{ env.GITHUB_AW_SAFE_OUTPUTS }}"`, `"/tmp/safe-outputs.jsonl"`)
	}
	
	// Replace ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }} with environment value or default
	if safeOutputsConfig := os.Getenv("GITHUB_AW_SAFE_OUTPUTS_CONFIG"); safeOutputsConfig != "" {
		resolved = strings.ReplaceAll(resolved, `${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}`, safeOutputsConfig)
	} else {
		// Default to empty config for local testing
		resolved = strings.ReplaceAll(resolved, `${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}`, `"{}"`)
	}
	
	// Replace ${{ secrets.GITHUB_TOKEN }} with environment value or default
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		resolved = strings.ReplaceAll(resolved, `"${{ secrets.GITHUB_TOKEN }}"`, fmt.Sprintf(`"%s"`, ghToken))
	} else if ghToken := os.Getenv("GH_TOKEN"); ghToken != "" {
		resolved = strings.ReplaceAll(resolved, `"${{ secrets.GITHUB_TOKEN }}"`, fmt.Sprintf(`"%s"`, ghToken))
	} else {
		if verbose {
			fmt.Println(console.FormatWarningMessage("GitHub token not found in environment (set GITHUB_TOKEN or GH_TOKEN)"))
		}
		resolved = strings.ReplaceAll(resolved, `"${{ secrets.GITHUB_TOKEN }}"`, `"your-github-token"`)
	}
	
	return resolved
}

// extractJSONFromScript extracts the JSON configuration from the generated shell script
func extractJSONFromScript(script string) (string, error) {
	// Look for the JSON content between << 'EOF' and EOF (multiline with DOTALL flag)
	re := regexp.MustCompile(`(?s)cat > [^<]+ << 'EOF'\s*\n(.*?)\n\s*EOF`)
	matches := re.FindStringSubmatch(script)
	
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find JSON configuration in script")
	}
	
	return strings.TrimSpace(matches[1]), nil
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

This command generates MCP configurations using the Claude agentic engine and automatically
spawns configured servers including github, playwright, and safe-outputs.

Examples:
  gh aw mcp inspect                    # List workflows with MCP servers
  gh aw mcp inspect weekly-research    # Inspect MCP servers in weekly-research.md  
  gh aw mcp inspect repomind --server repo-mind  # Inspect only the repo-mind server
  gh aw mcp inspect weekly-research --server github --tool create_issue  # Show details for a specific tool
  gh aw mcp inspect weekly-research -v # Verbose output with detailed connection info

The command will:
- Parse the workflow file to extract MCP server configurations
- Generate MCP configuration using the Claude agentic engine
- Spawn MCP servers from the generated configuration
- Query available tools, resources, and roots
- Validate required secrets are available  
- Display results in formatted tables with error details
- Keep servers running until interrupted (Ctrl+C)`,
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

