package cli

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The capability is captured at startup, never supplied by a tool caller.
func mcpWriteToolsMiddleware(allowWriteTools bool, actor string, validateActor bool) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}

			tool := extractMCPToolName(req)
			switch tool {
			case "compile":
				if allowWriteTools {
					if err := checkActorPermission(ctx, actor, validateActor, tool); err != nil {
						return nil, err
					}
				}
			case "add", "update", "fix":
				if !allowWriteTools {
					mcpLog.Printf("Tool %s: access denied (workflow-writing tools disabled)", tool)
					return nil, newMCPError(jsonrpc.CodeInvalidRequest, "permission denied: workflow-writing tools disabled; restart gh aw mcp-server with --allow-write-tools to enable them for trusted clients and workflow sources", map[string]any{
						"tool":   tool,
						"reason": "The operator must start gh aw mcp-server with --allow-write-tools to allow workflow modifications. Enable this only for trusted clients and workflow sources.",
					})
				}
				if err := checkActorPermission(ctx, actor, validateActor, tool); err != nil {
					return nil, err
				}
			}

			return next(ctx, method, req)
		}
	}
}
