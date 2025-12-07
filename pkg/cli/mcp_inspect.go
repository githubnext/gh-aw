package cli

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var mcpInspectLog = logger.New("cli:mcp_inspect")

const (
	// Port range for safe-inputs HTTP server
	safeInputsStartPort = 3000
	safeInputsPortRange = 10
	// Port range for MCP inspector
	inspectorStartPort = 5173
	inspectorPortRange = 10
)

// filterOutSafeOutputs removes safe-outputs MCP servers from the list since they are
// handled by the workflow compiler and not actual MCP servers that can be inspected
func filterOutSafeOutputs(configs []parser.MCPServerConfig) []parser.MCPServerConfig {
	var filteredConfigs []parser.MCPServerConfig
	for _, config := range configs {
		if config.Name != constants.SafeOutputsMCPServerID {
			filteredConfigs = append(filteredConfigs, config)
		}
	}
	return filteredConfigs
}

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
func InspectWorkflowMCP(workflowFile string, serverFilter string, toolFilter string, verbose bool, useActionsSecrets bool) error {
	mcpInspectLog.Printf("Inspecting workflow MCP: workflow=%s, serverFilter=%s, toolFilter=%s",
		workflowFile, serverFilter, toolFilter)

	workflowsDir := getWorkflowsDir()

	// If no workflow file specified, show available workflow files with MCP configs
	if workflowFile == "" {
		return listWorkflowsWithMCP(workflowsDir, verbose)
	}

	// Resolve the workflow file path
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		return err
	}

	// Convert to absolute path if needed
	if !filepath.IsAbs(workflowPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		workflowPath = filepath.Join(cwd, workflowPath)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Inspecting MCP servers in: %s", workflowPath)))
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
	importsResult, err := parser.ProcessImportsFromFrontmatterWithManifest(workflowData.Frontmatter, markdownDir, nil)
	if err != nil {
		return fmt.Errorf("failed to process imports from frontmatter: %w", err)
	}

	// Apply imported MCP servers to frontmatter
	frontmatterWithImports, err := applyImportsToFrontmatter(workflowData.Frontmatter, importsResult)
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
		if err := inspectMCPServer(config, toolFilter, verbose, useActionsSecrets); err != nil {
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
	// Scan workflows for MCP configurations
	results, err := ScanWorkflowsForMCP(workflowsDir, "", verbose)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no .github/workflows directory found")
		}
		return err
	}

	// Filter out safe-outputs MCP servers for inspection
	var workflowsWithMCP []string
	for _, result := range results {
		filteredConfigs := filterOutSafeOutputs(result.MCPConfigs)
		if len(filteredConfigs) > 0 {
			workflowsWithMCP = append(workflowsWithMCP, result.FileName)
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

// writeSafeInputsFiles writes all safe-inputs MCP server files to the specified directory
func writeSafeInputsFiles(dir string, safeInputsConfig *workflow.SafeInputsConfig, verbose bool) error {
	mcpInspectLog.Printf("Writing safe-inputs files to: %s", dir)

	// Create logs directory
	logsDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Write JavaScript dependencies that are needed
	jsFiles := []struct {
		name    string
		content string
	}{
		{"read_buffer.cjs", workflow.GetReadBufferScript()},
		{"mcp_server.cjs", workflow.GetMCPServerScript()},
		{"mcp_http_transport.cjs", workflow.GetMCPHTTPTransportScript()},
		{"safe_inputs_config_loader.cjs", workflow.GetSafeInputsConfigLoaderScript()},
		{"mcp_server_core.cjs", workflow.GetMCPServerCoreScript()},
		{"safe_inputs_validation.cjs", workflow.GetSafeInputsValidationScript()},
		{"mcp_logger.cjs", workflow.GetMCPLoggerScript()},
		{"mcp_handler_shell.cjs", workflow.GetMCPHandlerShellScript()},
		{"mcp_handler_python.cjs", workflow.GetMCPHandlerPythonScript()},
		{"safe_inputs_mcp_server_http.cjs", workflow.GetSafeInputsMCPServerHTTPScript()},
	}

	for _, jsFile := range jsFiles {
		filePath := filepath.Join(dir, jsFile.name)
		if err := os.WriteFile(filePath, []byte(jsFile.content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", jsFile.name, err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Wrote %s", jsFile.name)))
		}
	}

	// Generate and write tools.json
	toolsJSON := workflow.GenerateSafeInputsToolsConfigForInspector(safeInputsConfig)
	toolsPath := filepath.Join(dir, "tools.json")
	if err := os.WriteFile(toolsPath, []byte(toolsJSON), 0644); err != nil {
		return fmt.Errorf("failed to write tools.json: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Wrote tools.json"))
	}

	// Generate and write mcp-server.cjs entry point
	mcpServerScript := workflow.GenerateSafeInputsMCPServerScriptForInspector(safeInputsConfig)
	mcpServerPath := filepath.Join(dir, "mcp-server.cjs")
	if err := os.WriteFile(mcpServerPath, []byte(mcpServerScript), 0755); err != nil {
		return fmt.Errorf("failed to write mcp-server.cjs: %w", err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Wrote mcp-server.cjs"))
	}

	// Generate and write tool handler files
	for toolName, toolConfig := range safeInputsConfig.Tools {
		var content string
		var extension string

		if toolConfig.Script != "" {
			content = workflow.GenerateSafeInputJavaScriptToolScriptForInspector(toolConfig)
			extension = ".cjs"
		} else if toolConfig.Run != "" {
			content = workflow.GenerateSafeInputShellToolScriptForInspector(toolConfig)
			extension = ".sh"
		} else if toolConfig.Py != "" {
			content = workflow.GenerateSafeInputPythonToolScriptForInspector(toolConfig)
			extension = ".py"
		} else {
			continue
		}

		toolPath := filepath.Join(dir, toolName+extension)
		mode := os.FileMode(0644)
		if extension == ".sh" || extension == ".py" {
			mode = 0755
		}
		if err := os.WriteFile(toolPath, []byte(content), mode); err != nil {
			return fmt.Errorf("failed to write tool %s: %w", toolName, err)
		}
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Wrote tool handler: %s%s", toolName, extension)))
		}
	}

	mcpInspectLog.Printf("Successfully wrote all safe-inputs files")
	return nil
}

// startSafeInputsHTTPServer starts the safe-inputs HTTP MCP server
func startSafeInputsHTTPServer(dir string, port int, verbose bool) (*exec.Cmd, error) {
	mcpInspectLog.Printf("Starting safe-inputs HTTP server on port %d", port)

	mcpServerPath := filepath.Join(dir, "mcp-server.cjs")

	cmd := exec.Command("node", mcpServerPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GH_AW_SAFE_INPUTS_PORT=%d", port),
	)

	// Capture output for debugging
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Started safe-inputs server (PID: %d)", cmd.Process.Pid)))
	}

	return cmd, nil
}

// findAvailablePort finds an available port starting from the given port within the specified range
func findAvailablePort(startPort int, portRange int, verbose bool) int {
	for port := startPort; port < startPort+portRange; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			listener.Close()
			if verbose {
				mcpInspectLog.Printf("Found available port: %d", port)
			}
			return port
		}
	}
	return 0
}

// waitForServerReady waits for the HTTP server to be ready by polling the endpoint
func waitForServerReady(port int, timeout time.Duration, verbose bool) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	url := fmt.Sprintf("http://localhost:%d/", port)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if verbose {
				mcpInspectLog.Printf("Server is ready on port %d", port)
			}
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}

	mcpInspectLog.Printf("Server did not become ready within timeout")
	return false
}

// spawnSafeInputsInspector generates safe-inputs MCP server files, starts the HTTP server,
// and launches the inspector to inspect it
func spawnSafeInputsInspector(workflowFile string, verbose bool) error {
	mcpInspectLog.Printf("Spawning safe-inputs inspector for workflow: %s", workflowFile)

	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found. Please install Node.js and npm to use the MCP inspector: %w", err)
	}

	// Check if node is available
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("node not found. Please install Node.js to run the safe-inputs MCP server: %w", err)
	}

	// Resolve the workflow file path
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		return err
	}

	// Convert to absolute path if needed
	if !filepath.IsAbs(workflowPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		workflowPath = filepath.Join(cwd, workflowPath)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Inspecting safe-inputs from: %s", workflowPath)))
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

	// Extract safe-inputs configuration
	safeInputsConfig := workflow.ParseSafeInputs(workflowData.Frontmatter)
	if safeInputsConfig == nil || len(safeInputsConfig.Tools) == 0 {
		return fmt.Errorf("no safe-inputs configuration found in workflow")
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d safe-input tool(s) to configure", len(safeInputsConfig.Tools))))

	// Create temporary directory for safe-inputs files
	tmpDir, err := os.MkdirTemp("", "gh-aw-safe-inputs-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup temporary directory: %v", err)))
		}
	}()

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Created temporary directory: %s", tmpDir)))
	}

	// Write safe-inputs files to temporary directory
	if err := writeSafeInputsFiles(tmpDir, safeInputsConfig, verbose); err != nil {
		return fmt.Errorf("failed to write safe-inputs files: %w", err)
	}

	// Find an available port for the HTTP server
	port := findAvailablePort(safeInputsStartPort, safeInputsPortRange, verbose)
	if port == 0 {
		return fmt.Errorf("failed to find an available port for the HTTP server")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using port %d for safe-inputs HTTP server", port)))
	}

	// Start the HTTP server
	serverCmd, err := startSafeInputsHTTPServer(tmpDir, port, verbose)
	if err != nil {
		return fmt.Errorf("failed to start safe-inputs HTTP server: %w", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			// Try graceful shutdown first
			if err := serverCmd.Process.Signal(os.Interrupt); err != nil && verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to send interrupt signal: %v", err)))
			}
			// Wait a moment for graceful shutdown
			time.Sleep(500 * time.Millisecond)
			// Check if process is still running before force kill
			// On Unix, sending signal 0 checks if process exists without killing it
			if err := serverCmd.Process.Signal(os.Signal(syscall.Signal(0))); err == nil {
				// Process still running, force kill
				if err := serverCmd.Process.Kill(); err != nil && verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to kill server process: %v", err)))
				}
			}
		}
	}()

	// Wait for the server to start up
	if !waitForServerReady(port, 5*time.Second, verbose) {
		return fmt.Errorf("safe-inputs HTTP server failed to start within timeout")
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Safe-inputs HTTP server started successfully"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Server running on: http://localhost:%d", port)))
	fmt.Println()
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Configure the MCP inspector with the following settings:"))
	fmt.Fprintf(os.Stderr, "  Type: HTTP\n")
	fmt.Fprintf(os.Stderr, "  URL: http://localhost:%d\n", port)
	fmt.Println()

	// Find an available port for the inspector
	inspectorPort := findAvailablePort(inspectorStartPort, inspectorPortRange, verbose)
	if inspectorPort == 0 {
		return fmt.Errorf("failed to find an available port for the MCP inspector")
	}

	// Launch the inspector
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Launching @modelcontextprotocol/inspector..."))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Visit http://localhost:%d after the inspector starts", inspectorPort)))

	inspectorCmd := exec.Command("npx", "@modelcontextprotocol/inspector")
	inspectorCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", inspectorPort))
	inspectorCmd.Stdout = os.Stdout
	inspectorCmd.Stderr = os.Stderr
	inspectorCmd.Stdin = os.Stdin

	return inspectorCmd.Run()
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
		// Resolve the workflow file path (supports shared workflows)
		workflowPath, err := ResolveWorkflowPath(workflowFile)
		if err != nil {
			return err
		}

		// Convert to absolute path if needed
		if !filepath.IsAbs(workflowPath) {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			workflowPath = filepath.Join(cwd, workflowPath)
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
		importsResult, err := parser.ProcessImportsFromFrontmatterWithManifest(workflowData.Frontmatter, markdownDir, nil)
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

	// Find an available port for the inspector
	inspectorPort := findAvailablePort(inspectorStartPort, inspectorPortRange, verbose)
	if inspectorPort == 0 {
		return fmt.Errorf("failed to find an available port for the MCP inspector")
	}

	fmt.Println(console.FormatInfoMessage("Launching @modelcontextprotocol/inspector..."))
	fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Visit http://localhost:%d after the inspector starts", inspectorPort)))
	if len(serverProcesses) > 0 {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("%d stdio MCP server(s) are running in the background", len(serverProcesses))))
		fmt.Println(console.FormatInfoMessage("Configure them in the inspector using the details shown above"))
	}

	cmd := exec.Command("npx", "@modelcontextprotocol/inspector")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", inspectorPort))
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
	var checkSecrets bool
	var safeInputs bool

	cmd := &cobra.Command{
		Use:   "inspect [workflow-id-or-file]",
		Short: "Inspect MCP servers and list available tools, resources, and roots",
		Long: `Inspect MCP servers used by a workflow and display available tools, resources, and roots.

This command starts each MCP server configured in the workflow, queries its capabilities,
and displays the results in a formatted table. It supports stdio, Docker, and HTTP MCP servers.

The workflow-id-or-file can be:
- A workflow ID (basename without .md extension, e.g., "weekly-research")
- A file path (e.g., "weekly-research.md" or ".github/workflows/weekly-research.md")

Examples:
  gh aw mcp inspect                    # List workflows with MCP servers
  gh aw mcp inspect weekly-research    # Inspect MCP servers in weekly-research.md
  gh aw mcp inspect daily-news --server tavily  # Inspect only the tavily server
  gh aw mcp inspect weekly-research --server github --tool create_issue  # Show details for a specific tool
  gh aw mcp inspect weekly-research -v # Verbose output with detailed connection info
  gh aw mcp inspect weekly-research --inspector  # Launch @modelcontextprotocol/inspector
  gh aw mcp inspect weekly-research --check-secrets  # Check GitHub Actions secrets
  gh aw mcp inspect weekly-research --safe-inputs  # Inspect safe-inputs MCP server with inspector

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

			// Validate that safe-inputs and inspector flags are mutually exclusive with other flags
			if safeInputs {
				if workflowFile == "" {
					return fmt.Errorf("--safe-inputs flag requires a workflow file to be specified")
				}
				if spawnInspector {
					return fmt.Errorf("--safe-inputs already includes inspector functionality; do not use --inspector flag with --safe-inputs")
				}
				if serverFilter != "" || toolFilter != "" {
					return fmt.Errorf("--safe-inputs cannot be used with --server or --tool flags")
				}
				return spawnSafeInputsInspector(workflowFile, verbose)
			}

			// Handle spawn inspector flag
			if spawnInspector {
				return spawnMCPInspector(workflowFile, serverFilter, verbose)
			}

			return InspectWorkflowMCP(workflowFile, serverFilter, toolFilter, verbose, checkSecrets)
		},
	}

	cmd.Flags().StringVar(&serverFilter, "server", "", "Filter to inspect only the specified MCP server")
	cmd.Flags().StringVar(&toolFilter, "tool", "", "Show detailed information about a specific tool (requires --server)")
	cmd.Flags().BoolVar(&spawnInspector, "inspector", false, "Launch the official @modelcontextprotocol/inspector tool")
	cmd.Flags().BoolVar(&checkSecrets, "check-secrets", false, "Check GitHub Actions repository secrets for missing secrets")
	cmd.Flags().BoolVar(&safeInputs, "safe-inputs", false, "Launch safe-inputs MCP server and inspect it")

	// Register completions for mcp inspect command
	cmd.ValidArgsFunction = CompleteWorkflowNames

	return cmd
}
