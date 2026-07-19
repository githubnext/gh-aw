package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var mcpInspectorLog = logger.New("cli:mcp_inspect_inspector")

const (
	// mcpStdioServerStartupDelay gives stdio MCP servers time to start accepting connections.
	mcpStdioServerStartupDelay = 2 * time.Second
	// mcpProcessCleanupDelay spaces cleanup signals so each MCP process can exit cleanly.
	mcpProcessCleanupDelay = 100 * time.Millisecond
)

var (
	mcpInspectorLookPath       = exec.LookPath
	mcpInspectorCommandContext = exec.CommandContext
	mcpInspectorMonitorDone    = func(string) {}
)

// spawnMCPInspector launches the official @modelcontextprotocol/inspector tool
// and spawns any stdio MCP servers beforehand
func spawnMCPInspector(ctx context.Context, workflowFile string, serverFilter string, verbose bool) error {
	mcpInspectorLog.Printf("Spawning MCP inspector: workflow_file=%s, server_filter=%s", workflowFile, serverFilter)
	// Check if npx is available
	if _, err := mcpInspectorLookPath("npx"); err != nil {
		return fmt.Errorf("npx not found. Please install Node.js and npm to use the MCP inspector: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	mcpConfigs, serverProcesses, err := spawnMCPInspectorWorkflow(gctx, workflowFile, serverFilter, verbose, g)
	if err != nil {
		return err
	}
	if len(mcpConfigs) == 0 && workflowFile != "" {
		return nil
	}

	// Set up cleanup function for stdio servers
	defer spawnMCPInspectorCleanup(serverProcesses, verbose, g)

	return spawnMCPInspectorRun(gctx, serverProcesses)
}

func spawnMCPInspectorWorkflow(ctx context.Context, workflowFile, serverFilter string, verbose bool, g *errgroup.Group) ([]parser.RegistryMCPServerConfig, []*exec.Cmd, error) {
	if workflowFile == "" {
		return nil, nil, nil
	}
	mcpConfigs, err := spawnMCPInspectorLoadConfigs(workflowFile, serverFilter, verbose)
	if err != nil {
		return nil, nil, err
	}
	if len(mcpConfigs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No MCP servers found in workflow"))
		return mcpConfigs, nil, nil
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d MCP server(s) in workflow:", len(mcpConfigs))))
	for _, config := range mcpConfigs {
		fmt.Fprintf(os.Stderr, "  • %s (%s)\n", config.Name, config.Type)
	}
	fmt.Fprintln(os.Stderr)
	serverProcesses := spawnMCPInspectorStartStdioServers(ctx, mcpConfigs, verbose, g)
	spawnMCPInspectorPrintConfigDetails(mcpConfigs)
	return mcpConfigs, serverProcesses, nil
}

func spawnMCPInspectorLoadConfigs(workflowFile, serverFilter string, verbose bool) ([]parser.RegistryMCPServerConfig, error) {
	workflowPath, err := ResolveWorkflowPath(workflowFile)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(workflowPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		workflowPath = filepath.Join(cwd, workflowPath)
	}
	compiler := workflow.NewCompiler(workflow.WithVerbose(verbose))
	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		return nil, err
	}
	frontmatterForMCP := buildFrontmatterFromWorkflowData(workflowData)
	mcpConfigs, err := parser.ExtractMCPConfigurations(frontmatterForMCP, serverFilter)
	if err != nil {
		mcpInspectorLog.Printf("Failed to extract MCP configurations: %v", err)
		return nil, err
	}
	mcpInspectorLog.Printf("Extracted %d MCP server configurations from workflow", len(mcpConfigs))
	return mcpConfigs, nil
}

func spawnMCPInspectorStartStdioServers(ctx context.Context, mcpConfigs []parser.RegistryMCPServerConfig, verbose bool, g *errgroup.Group) []*exec.Cmd {
	stdioServers := []parser.RegistryMCPServerConfig{}
	for _, config := range mcpConfigs {
		if config.Type == "stdio" {
			stdioServers = append(stdioServers, config)
		}
	}
	if len(stdioServers) == 0 {
		return nil
	}
	mcpInspectorLog.Printf("Starting %d stdio MCP servers", len(stdioServers))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting stdio MCP servers..."))
	serverProcesses := make([]*exec.Cmd, 0, len(stdioServers))
	for _, config := range stdioServers {
		if cmd := spawnMCPInspectorStartServer(ctx, config, verbose, g); cmd != nil {
			serverProcesses = append(serverProcesses, cmd)
		}
	}
	time.Sleep(mcpStdioServerStartupDelay)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All stdio servers started successfully"))
	return serverProcesses
}

func spawnMCPInspectorStartServer(ctx context.Context, config parser.RegistryMCPServerConfig, verbose bool, g *errgroup.Group) *exec.Cmd {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting server: "+config.Name))
	}
	cmd := spawnMCPInspectorCommand(ctx, config)
	if cmd == nil {
		return nil
	}
	if err := cmd.Start(); err != nil {
		mcpInspectorLog.Printf("Failed to start MCP server %s: %v", config.Name, err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to start server %s: %v", config.Name, err)))
		return nil
	}
	mcpInspectorLog.Printf("Started MCP server %s (PID: %d, type: %s)", config.Name, cmd.Process.Pid, config.Type)
	spawnMCPInspectorMonitorServer(g, cmd, config.Name, verbose)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Started server: %s (PID: %d)", config.Name, cmd.Process.Pid)))
	}
	return cmd
}

func spawnMCPInspectorCommand(ctx context.Context, config parser.RegistryMCPServerConfig) *exec.Cmd {
	var cmd *exec.Cmd
	if config.Container != "" {
		args := append([]string{"run", "--rm", "-i"}, config.Args...)
		cmd = mcpInspectorCommandContext(ctx, "docker", args...)
	} else {
		if config.Command == "" {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping server %s: no command specified", config.Name)))
			return nil
		}
		if _, err := mcpInspectorLookPath(config.Command); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping server %s: command not found: %s", config.Name, config.Command)))
			return nil
		}
		cmd = mcpInspectorCommandContext(ctx, config.Command, config.Args...)
	}
	cmd.Env = os.Environ()
	for key, value := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, os.ExpandEnv(value)))
	}
	return cmd
}

func spawnMCPInspectorMonitorServer(g *errgroup.Group, cmd *exec.Cmd, name string, verbose bool) {
	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				mcpInspectorLog.Printf("Recovered panic while waiting for MCP server %s: %v", name, r)
			}
			mcpInspectorMonitorDone(name)
		}()
		if err := cmd.Wait(); err != nil {
			mcpInspectorLog.Printf("MCP server %s exited with error: %v", name, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Server %s exited with error: %v", name, err)))
			}
		}
		return nil
	})
}

func spawnMCPInspectorPrintConfigDetails(mcpConfigs []parser.RegistryMCPServerConfig) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Configuration details for MCP inspector:"))
	for _, config := range mcpConfigs {
		fmt.Fprintf(os.Stderr, "\n📡 %s (%s):\n", config.Name, config.Type)
		switch config.Type {
		case "stdio":
			if config.Container != "" {
				fmt.Fprintf(os.Stderr, "  Container: %s\n", config.Container)
			} else {
				fmt.Fprintf(os.Stderr, "  Command: %s\n", config.Command)
				if len(config.Args) > 0 {
					fmt.Fprintf(os.Stderr, "  Args: %s\n", strings.Join(config.Args, " "))
				}
			}
		case "http":
			fmt.Fprintf(os.Stderr, "  URL: %s\n", config.URL)
		}
		if len(config.Env) > 0 {
			fmt.Fprintf(os.Stderr, "  Environment Variables: %v\n", config.Env)
		}
	}
	fmt.Fprintln(os.Stderr)
}

func spawnMCPInspectorCleanup(serverProcesses []*exec.Cmd, verbose bool, g *errgroup.Group) {
	if len(serverProcesses) == 0 {
		return
	}
	mcpInspectorLog.Printf("Cleaning up %d MCP server processes", len(serverProcesses))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Cleaning up MCP servers..."))
	for i, cmd := range serverProcesses {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil && verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to kill server process %d: %v", cmd.Process.Pid, err)))
			}
		}
		if i < len(serverProcesses)-1 {
			time.Sleep(mcpProcessCleanupDelay)
		}
	}
	if err := g.Wait(); err != nil {
		mcpInspectorLog.Printf("Error from MCP server monitor goroutine: %v", err)
	}
}

func spawnMCPInspectorRun(ctx context.Context, serverProcesses []*exec.Cmd) error {
	mcpInspectorLog.Print("Launching @modelcontextprotocol/inspector")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Launching @modelcontextprotocol/inspector..."))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Visit http://localhost:5173 after the inspector starts"))
	if len(serverProcesses) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("%d stdio MCP server(s) are running in the background", len(serverProcesses))))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Configure them in the inspector using the details shown above"))
	}
	cmd := mcpInspectorCommandContext(ctx, "npx", "@modelcontextprotocol/inspector")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		mcpInspectorLog.Printf("MCP inspector exited with error: %v", err)
	} else {
		mcpInspectorLog.Print("MCP inspector exited successfully")
	}
	return err
}
