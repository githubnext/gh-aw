package cli

import (
	"context"
	"os/exec"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// execCmdFunc is the type for the command execution function passed to tool registrations.
type execCmdFunc func(ctx context.Context, args ...string) *exec.Cmd

// createMCPServer creates and configures the MCP server with all tools
func createMCPServer(cmdPath string, actor string, validateActor bool) *mcp.Server {
	// Helper function to execute command with proper path
	execCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		if cmdPath != "" {
			// Use custom command path
			return exec.CommandContext(ctx, cmdPath, args...)
		}
		// Use default gh aw command with proper token handling
		return workflow.ExecGHContext(ctx, append([]string{"aw"}, args...)...)
	}

	// Log actor and validation settings
	if validateActor {
		if actor != "" {
			mcpLog.Printf("Actor validation enabled: actor=%s (logs/audit tools will check permissions)", actor)
		} else {
			mcpLog.Print("Actor validation enabled: no actor specified (logs/audit tools will deny access)")
		}
	} else {
		if actor != "" {
			mcpLog.Printf("Actor validation disabled: actor=%s (logs/audit tools will allow access)", actor)
		} else {
			mcpLog.Print("Actor validation disabled: no actor specified (logs/audit tools will allow access)")
		}
	}

	// Create MCP server with capabilities and logging
	// Note: Schema caching is automatic in go-sdk v1.3.0+ (eliminates repeated reflection overhead)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gh-aw",
		Version: GetVersion(),
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				ListChanged: false, // Tools are static, no notifications needed
			},
		},
		Logger: logger.NewSlogLoggerWithHandler(mcpLog),
	})

	// Register read-only tools
	registerStatusTool(server)

	if err := registerCompileTool(server, execCmd); err != nil {
		return server
	}

	// Register privileged tools (require write+ access)
	if err := registerLogsTool(server, execCmd, actor, validateActor); err != nil {
		return server
	}

	if err := registerAuditTool(server, execCmd, actor, validateActor); err != nil {
		return server
	}

	// Register remaining read-only tools
	registerMCPInspectTool(server, execCmd)

	// Register workflow management tools
	registerAddTool(server, execCmd)
	registerUpdateTool(server, execCmd)
	registerFixTool(server, execCmd)

	return server
}
