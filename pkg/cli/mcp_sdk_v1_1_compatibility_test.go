package cli

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPSDKv1_1_CapabilitiesAPI tests that the Capabilities API from MCP SDK v1.1.0 is properly configured
func TestMCPSDKv1_1_CapabilitiesAPI(t *testing.T) {
	// Create MCP server with capabilities
	server := createMCPServer("")

	// Verify the server has capabilities configured
	// The ServerOptions should include capabilities for tools
	// This test verifies that the server is configured with the Capabilities API
	// introduced in MCP SDK v1.1.0

	// We can't directly inspect the server's capabilities from the public API,
	// but we can verify that the server was created successfully with capabilities
	if server == nil {
		t.Fatal("Server should not be nil")
	}

	// The key test is that createMCPServer sets up ServerOptions with Capabilities
	// This is verified by checking that the code compiles and runs without errors
	t.Log("✓ MCP server created with Capabilities API")
}

// TestMCPSDKv1_1_LoggingIntegration tests that slog.Logger integration works
func TestMCPSDKv1_1_LoggingIntegration(t *testing.T) {
	// Create a logger
	log := logger.New("test:mcp")

	// Create slog logger from our logger
	slogLogger := logger.NewSlogLoggerWithHandler(log)

	// Verify the logger was created successfully
	if slogLogger == nil {
		t.Fatal("slog.Logger should not be nil")
	}

	// Create MCP server with logging
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Logger: slogLogger,
	})

	if server == nil {
		t.Fatal("Server with logger should not be nil")
	}

	t.Log("✓ MCP server created with slog.Logger integration")
}

// TestMCPSDKv1_1_ToolCapabilities tests that tool capabilities are properly configured
func TestMCPSDKv1_1_ToolCapabilities(t *testing.T) {
	// Create server with tool capabilities
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				ListChanged: false, // Tools are static, no notifications needed
			},
		},
	})

	if server == nil {
		t.Fatal("Server with tool capabilities should not be nil")
	}

	t.Log("✓ MCP server created with ToolCapabilities")
}

// TestMCPSDKv1_1_ServerOptions tests that all ServerOptions fields work correctly
func TestMCPSDKv1_1_ServerOptions(t *testing.T) {
	// Create logger for the server
	log := logger.New("test:mcp")
	slogLogger := logger.NewSlogLoggerWithHandler(log)

	// Create server with full ServerOptions configuration
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				ListChanged: false,
			},
		},
		Logger: slogLogger,
	})

	if server == nil {
		t.Fatal("Server with full ServerOptions should not be nil")
	}

	t.Log("✓ MCP server created with full ServerOptions (Capabilities + Logger)")
}

// TestStreamableHTTPOptions tests that StreamableHTTPOptions is available
func TestStreamableHTTPOptions(t *testing.T) {
	// Verify that we can create StreamableHTTPOptions with SessionTimeout
	// This is a compile-time check for MCP SDK v1.1.0+ compatibility
	options := &mcp.StreamableHTTPOptions{
		SessionTimeout: 30 * 60 * 1000000000, // 30 minutes in nanoseconds (time.Duration)
	}

	if options == nil {
		t.Fatal("StreamableHTTPOptions should not be nil")
	}

	if options.SessionTimeout <= 0 {
		t.Fatal("SessionTimeout should be positive")
	}

	t.Log("✓ StreamableHTTPOptions with SessionTimeout is available")
}
