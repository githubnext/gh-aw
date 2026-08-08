//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	if len(info.Roots) != 1 || info.Roots[0].URI != "test://" {
		t.Errorf("expected inferred test:// root, got %+v", info.Roots)
	}
}

func TestQueryServerCapabilities_PartialResultsError(t *testing.T) {
	ctx := context.Background()
	var listToolsCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if r.Method == http.MethodPost && r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		switch req.Method {
		case "server/discover":
			writeJSONRPCError(t, w, req.ID, -32601, `method not found: "server/discover"`)
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "test-session")
			writeJSONRPCResult(t, w, req.ID, map[string]any{
				"capabilities": map[string]any{
					"tools":     map[string]any{"listChanged": true},
					"resources": map[string]any{"listChanged": true},
					"prompts":   map[string]any{"listChanged": true},
				},
				"protocolVersion": "2025-11-25",
				"serverInfo":      map[string]any{"name": "test-server", "version": "1.0.0"},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			if listToolsCalls.Add(1) == 1 {
				writeJSONRPCResult(t, w, req.ID, map[string]any{
					"tools":      []map[string]any{{"name": "tool-0", "description": "tool-0"}},
					"nextCursor": "page-2",
				})
				return
			}
			http.Error(w, "upstream tools page failed", http.StatusBadGateway)
		case "resources/list":
			writeJSONRPCResult(t, w, req.ID, map[string]any{
				"resources": []map[string]any{{"uri": "file:///tmp/resource-0", "name": "resource-0"}},
			})
		case "prompts/list":
			writeJSONRPCResult(t, w, req.ID, map[string]any{
				"prompts": []map[string]any{{"name": "prompt-0", "description": "prompt-0"}},
			})
		default:
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             server.URL,
		DisableStandaloneSSE: true,
	}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer session.Close()

	config := parser.RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{Type: "http", URL: server.URL},
		Name:                "test-server",
	}

	info := queryServerCapabilities(ctx, config, session, false)

	if len(info.Tools) != 1 {
		t.Fatalf("expected one partial tool result before pagination error, got %d", len(info.Tools))
	}
	if info.Error == nil {
		t.Fatal("expected partial-results error to be recorded")
	}
	if !strings.Contains(info.Error.Error(), "listing tools") {
		t.Fatalf("expected tools error context, got %v", info.Error)
	}
	if len(info.Resources) != 1 {
		t.Fatalf("expected resources to still be listed, got %d", len(info.Resources))
	}
	if len(info.Roots) != 1 || info.Roots[0].URI != "file://" {
		t.Fatalf("expected inferred file:// root after partial error, got %+v", info.Roots)
	}
	if len(info.Prompts) != 1 {
		t.Fatalf("expected prompts to still be listed, got %d", len(info.Prompts))
	}
}

func TestDisplayServerCapabilities_PartialResultsWarning(t *testing.T) {
	info := &parser.MCPServerInfo{
		Error: errors.New("listing tools: upstream tools page failed"),
		Tools: []*mcp.Tool{
			{Name: "tool-0", Description: "tool-0"},
		},
	}

	_, stderr := captureOutput(t, func() error {
		displayServerCapabilities(info, "")
		return nil
	})

	if !strings.Contains(stderr, "MCP inspection returned partial results") {
		t.Fatalf("expected partial-results warning on stderr, got %q", stderr)
	}
}

func writeJSONRPCResult(t *testing.T, w http.ResponseWriter, id, result any) {
	t.Helper()
	writeJSONRPCMessage(t, w, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeJSONRPCError(t *testing.T, w http.ResponseWriter, id any, code int, message string) {
	t.Helper()
	writeJSONRPCMessage(t, w, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSONRPCMessage(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("failed to encode JSON-RPC body: %v", err)
	}
}
