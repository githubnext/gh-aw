package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/types"
)

// TestQueryServerCapabilities_Pagination verifies that queryServerCapabilities
// retrieves all tools, resources, and prompts from a server, even when the
// server splits the results across multiple pages. This exercises the SDK's
// iterator-based pagination helpers (session.Tools/Resources/Prompts), which
// automatically follow cursors instead of requiring a single manual request.
func TestQueryServerCapabilities_Pagination(t *testing.T) {
	ctx := context.Background()

	// Force pagination by using a page size smaller than the number of items
	// registered below.
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "1.0.0"}, &mcp.ServerOptions{
		PageSize: 1,
	})

	const itemCount = 3
	for i := range itemCount {
		name := fmt.Sprintf("tool-%d", i)
		mcp.AddTool(server, &mcp.Tool{Name: name, Description: name}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{}, nil, nil
		})
		server.AddResource(&mcp.Resource{URI: fmt.Sprintf("test://resource-%d", i), Name: name}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{}, nil
		})
		server.AddPrompt(&mcp.Prompt{Name: name}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{}, nil
		})
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer clientSession.Close()

	config := parser.RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "stdio"},
		Name:                "test-server",
	}

	info := queryServerCapabilities(ctx, config, clientSession, false)

	if len(info.Tools) != itemCount {
		t.Errorf("expected %d tools, got %d", itemCount, len(info.Tools))
	}
	if len(info.Resources) != itemCount {
		t.Errorf("expected %d resources, got %d", itemCount, len(info.Resources))
	}
	if len(info.Prompts) != itemCount {
		t.Errorf("expected %d prompts, got %d", itemCount, len(info.Prompts))
	}
}
