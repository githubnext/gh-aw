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

// MCP tool handlers in this package return (*mcp.CallToolResult, any, error).
// The second return value is a reserved SDK extension slot and must be nil for now.

// createMCPServer creates and configures the MCP server with all tools
func createMCPServer(cmdPath string, actor string, validateActor bool, manifestCacheFile string, env []string) *mcp.Server {
	execCmd := createMCPServerExecCmd(cmdPath, env)
	createMCPServerLogActor(actor, validateActor)
	server := createMCPServerInstance()

	if !createMCPServerRegisterTools(server, execCmd, actor, validateActor, manifestCacheFile) {
		return server
	}

	// Add receiving middleware to transform raw JSON-schema "additional properties"
	// validation errors into helpful messages with "Did you mean?" suggestions.
	server.AddReceivingMiddleware(argumentValidationMiddleware(mcpToolParams()))

	return server
}

func createMCPServerExecCmd(cmdPath string, env []string) execCmdFunc {
	return func(ctx context.Context, args ...string) *exec.Cmd {
		var cmd *exec.Cmd
		if cmdPath != "" {
			// Use custom command path
			cmd = exec.CommandContext(ctx, cmdPath, args...)
		} else {
			// Use default gh aw command with proper token handling
			cmd = workflow.ExecGHContext(ctx, append([]string{"aw"}, args...)...)
		}
		if env != nil {
			cmd.Env = append([]string(nil), env...)
		}
		return cmd
	}
}

func createMCPServerLogActor(actor string, validateActor bool) {
	// Log actor and validation settings
	if validateActor && actor != "" {
		mcpLog.Printf("Actor validation enabled: actor=%s (logs/audit tools will check permissions)", actor)
	} else if validateActor {
		mcpLog.Print("Actor validation enabled: no actor specified (logs/audit tools will deny access)")
	} else if actor != "" {
		mcpLog.Printf("Actor validation disabled: actor=%s (logs/audit tools will allow access)", actor)
	} else {
		mcpLog.Print("Actor validation disabled: no actor specified (logs/audit tools will allow access)")
	}
}

func createMCPServerInstance() *mcp.Server {
	// Create MCP server with capabilities and logging
	// Note: Schema caching is automatic in go-sdk v1.3.0+ (eliminates repeated reflection overhead)
	return mcp.NewServer(&mcp.Implementation{
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
}

func createMCPServerRegisterTools(server *mcp.Server, execCmd execCmdFunc, actor string, validateActor bool, manifestCacheFile string) bool {
	// Register read-only tools
	registerStatusTool(server)

	if err := registerCompileTool(server, execCmd, manifestCacheFile); err != nil {
		return false
	}

	// Register privileged tools (require write+ access)
	if err := registerLogsTool(server, execCmd, actor, validateActor); err != nil {
		return false
	}

	if err := registerAuditTool(server, execCmd, actor, validateActor); err != nil {
		return false
	}

	if err := registerAuditDiffTool(server, execCmd, actor, validateActor); err != nil {
		return false
	}

	// Register remaining read-only tools
	registerChecksTool(server)
	registerMCPInspectTool(server, execCmd)

	// Register workflow management tools
	registerAddTool(server, execCmd)
	registerUpdateTool(server, execCmd)
	registerFixTool(server, execCmd)
	return true
}
