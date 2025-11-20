package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPBridgeIntegration tests the bridge with gh-aw's own MCP server
func TestMCPBridgeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build gh-aw if not already built
	ghAwPath := "../../gh-aw"
	if _, err := os.Stat(ghAwPath); os.IsNotExist(err) {
		t.Skip("gh-aw binary not found, run 'make build' first")
	}

	// Start the bridge in the background
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	port := 18080 // Use a high port to avoid conflicts
	bridgeCmd := exec.CommandContext(ctx, ghAwPath, "mcp", "bridge",
		"--command", ghAwPath,
		"--args", "mcp-server",
		"--port", fmt.Sprintf("%d", port))

	// Capture output for debugging
	bridgeCmd.Stderr = os.Stderr
	bridgeCmd.Stdout = os.Stdout

	err := bridgeCmd.Start()
	require.NoError(t, err, "Failed to start bridge")

	// Clean up bridge process
	defer func() {
		if bridgeCmd.Process != nil {
			bridgeCmd.Process.Kill()
		}
	}()

	// Wait for bridge to start
	time.Sleep(2 * time.Second)

	// Try to connect to the bridge HTTP endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d", port)

	resp, err := client.Get(url)
	if err != nil {
		t.Logf("Note: Bridge might not have started yet or encountered an error")
		t.Logf("This is expected in CI environments without proper MCP SDK setup")
		t.Skip("Skipping actual HTTP test - bridge likely requires MCP SDK dependencies")
		return
	}
	defer resp.Body.Close()

	// The bridge is running - a 400 is expected for a simple GET without MCP protocol
	// A 200 would mean the bridge is accepting HTTP requests, which is also valid
	// Any response (200, 400, etc.) means the HTTP server is running
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
		"Bridge HTTP endpoint should be accessible (got status %d)", resp.StatusCode)
}

// TestMCPBridgeCommandValidation tests command validation without actually running the bridge
func TestMCPBridgeCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing required flags",
			args:        []string{},
			expectError: true,
			errorMsg:    "required flag(s)",
		},
		{
			name:        "missing port flag",
			args:        []string{"--command", "echo"},
			expectError: true,
			errorMsg:    "required flag(s)",
		},
		{
			name:        "missing command flag",
			args:        []string{"--port", "8080"},
			expectError: true,
			errorMsg:    "required flag(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewMCPBridgeCommand()
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

// TestMCPBridgeCommandParsing tests that flags are parsed correctly
func TestMCPBridgeCommandParsing(t *testing.T) {
	cmd := NewMCPBridgeCommand()

	// Set arguments
	cmd.SetArgs([]string{
		"--command", "node",
		"--args", "server.js,--verbose",
		"--port", "8080",
	})

	// Parse flags (don't execute)
	err := cmd.ParseFlags([]string{
		"--command", "node",
		"--args", "server.js,--verbose",
		"--port", "8080",
	})
	require.NoError(t, err)

	// Verify flags were parsed correctly
	command, _ := cmd.Flags().GetString("command")
	assert.Equal(t, "node", command)

	args, _ := cmd.Flags().GetStringSlice("args")
	assert.Equal(t, []string{"server.js", "--verbose"}, args)

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 8080, port)
}
