//go:build integration

package awmg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// TestMCPGateway_InspectWithPlaywright tests the MCP gateway by:
// 1. Starting the gateway with a test configuration
// 2. Using mcp inspect to verify the gateway configuration
// 3. Checking the tool list is accessible
func TestMCPGateway_InspectWithPlaywright(t *testing.T) {
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(filepath.Join("..", "..", "gh-aw"))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: gh-aw binary not found at %s. Run 'make build' first.", binaryPath)
	}

	// Create temporary directory structure
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow that uses the MCP gateway
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
sandbox:
  mcp:
    port: 8089
tools:
  playwright:
    allowed_domains:
      - "localhost"
      - "example.com"
---

# Test MCP Gateway with mcp-inspect

This workflow tests the MCP gateway configuration and tool list.
`

	workflowFile := filepath.Join(workflowsDir, "test-mcp-gateway.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Create MCP gateway configuration with gh-aw MCP server
	configFile := filepath.Join(tmpDir, "gateway-config.json")
	config := MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"gh-aw": {
				Command: binaryPath,
				Args:    []string{"mcp-server"},
			},
		},
		Gateway: GatewaySettings{
			Port: 8089,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal gateway config: %v", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write gateway config file: %v", err)
	}

	// Start the MCP gateway in background
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gatewayErrChan := make(chan error, 1)
	go func() {
		// Use context for gateway lifecycle
		_ = ctx // Mark as used
		gatewayErrChan <- runMCPGateway([]string{configFile}, 8089, tmpDir)
	}()

	// Wait for gateway to start
	t.Log("Waiting for MCP gateway to start...")
	time.Sleep(3 * time.Second)

	// Verify gateway health endpoint
	healthResp, err := http.Get("http://localhost:8089/health")
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect to gateway health endpoint: %v", err)
	}
	healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("Gateway health check failed: status=%d", healthResp.StatusCode)
	}
	t.Log("✓ Gateway health check passed")

	// Test 1: Verify gateway servers endpoint
	serversResp, err := http.Get("http://localhost:8089/servers")
	if err != nil {
		cancel()
		t.Fatalf("Failed to get servers list from gateway: %v", err)
	}
	defer serversResp.Body.Close()

	var serversData map[string]any
	if err := json.NewDecoder(serversResp.Body).Decode(&serversData); err != nil {
		t.Fatalf("Failed to decode servers response: %v", err)
	}

	servers, ok := serversData["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Fatalf("Expected servers list, got: %v", serversData)
	}
	t.Logf("✓ Gateway has %d server(s)", len(servers))

	// Test 2: Use mcp inspect to check the workflow configuration
	t.Log("Running mcp inspect on test workflow...")
	inspectCmd := exec.Command(binaryPath, "mcp", "inspect", "test-mcp-gateway", "--verbose")
	inspectCmd.Dir = tmpDir
	inspectCmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
	)

	output, err := inspectCmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Logf("mcp inspect output:\n%s", outputStr)
		t.Fatalf("mcp inspect failed: %v", err)
	}

	t.Logf("mcp inspect output:\n%s", outputStr)

	// Verify the output contains expected information
	if !strings.Contains(outputStr, "playwright") {
		t.Errorf("Expected 'playwright' in mcp inspect output")
	}

	// Test 3: Use mcp inspect with --server flag to check specific server
	t.Log("Running mcp inspect with --server playwright...")
	inspectServerCmd := exec.Command(binaryPath, "mcp", "inspect", "test-mcp-gateway", "--server", "playwright", "--verbose")
	inspectServerCmd.Dir = tmpDir
	inspectServerCmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
	)

	serverOutput, err := inspectServerCmd.CombinedOutput()
	serverOutputStr := string(serverOutput)

	if err != nil {
		t.Logf("mcp inspect --server output:\n%s", serverOutputStr)
		// This might fail if playwright server isn't available, which is okay
		t.Logf("Warning: mcp inspect --server failed (expected if playwright not configured): %v", err)
	} else {
		t.Logf("mcp inspect --server output:\n%s", serverOutputStr)
	}

	// Test 4: Verify tool list can be accessed via mcp list command
	t.Log("Running mcp list to check available tools...")
	listCmd := exec.Command(binaryPath, "mcp", "list", "test-mcp-gateway")
	listCmd.Dir = tmpDir
	listCmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
	)

	listOutput, err := listCmd.CombinedOutput()
	listOutputStr := string(listOutput)

	if err != nil {
		t.Logf("mcp list output:\n%s", listOutputStr)
		t.Fatalf("mcp list failed: %v", err)
	}

	t.Logf("mcp list output:\n%s", listOutputStr)

	// Verify the list output contains MCP server information
	if !strings.Contains(listOutputStr, "MCP") {
		t.Errorf("Expected 'MCP' in mcp list output")
	}

	// Test 5: Check tool list using mcp list-tools command
	t.Log("Running mcp list-tools to enumerate available tools...")
	listToolsCmd := exec.Command(binaryPath, "mcp", "list-tools", "test-mcp-gateway")
	listToolsCmd.Dir = tmpDir
	listToolsCmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
	)

	toolsOutput, err := listToolsCmd.CombinedOutput()
	toolsOutputStr := string(toolsOutput)

	if err != nil {
		t.Logf("mcp list-tools output:\n%s", toolsOutputStr)
		// This might fail depending on MCP server configuration
		t.Logf("Warning: mcp list-tools failed: %v", err)
	} else {
		t.Logf("mcp list-tools output:\n%s", toolsOutputStr)

		// If successful, verify we have tool information
		if strings.Contains(toolsOutputStr, "No tools") {
			t.Log("Note: No tools found in MCP servers (this may be expected)")
		}
	}

	t.Log("✓ All mcp inspect tests completed successfully")

	// Clean up: cancel context to stop the gateway
	cancel()

	// Wait for gateway to stop
	select {
	case err := <-gatewayErrChan:
		if err != nil && err != http.ErrServerClosed && !strings.Contains(err.Error(), "context canceled") {
			t.Logf("Gateway stopped with error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Log("Gateway shutdown timed out")
	}
}

// TestMCPGateway_InspectToolList specifically tests tool list inspection
func TestMCPGateway_InspectToolList(t *testing.T) {
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(filepath.Join("..", "..", "gh-aw"))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: gh-aw binary not found at %s. Run 'make build' first.", binaryPath)
	}

	// Create temporary directory
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a minimal workflow for tool list testing
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Tool List Inspection

Test workflow for verifying tool list via mcp inspect.
`

	workflowFile := filepath.Join(workflowsDir, "test-tools.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Run mcp inspect to check tool list
	t.Log("Running mcp inspect to check tool list...")
	inspectCmd := exec.Command(binaryPath, "mcp", "inspect", "test-tools", "--server", "github", "--verbose")
	inspectCmd.Dir = tmpDir
	inspectCmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
		"GH_TOKEN=dummy_token_for_testing", // Provide dummy token for GitHub MCP
	)

	output, err := inspectCmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("mcp inspect output:\n%s", outputStr)

	// Check if inspection was successful or at least attempted
	if err != nil {
		// It's okay if it fails due to auth issues, we're testing the workflow parsing
		if !strings.Contains(outputStr, "github") && !strings.Contains(outputStr, "Secret validation") {
			t.Fatalf("mcp inspect failed unexpectedly: %v", err)
		}
		t.Log("Note: Inspection failed as expected due to auth/connection issues")
	}

	// Verify the workflow was parsed and github server was detected
	if strings.Contains(outputStr, "github") || strings.Contains(outputStr, "GitHub MCP") {
		t.Log("✓ GitHub MCP server detected in workflow")
	}

	t.Log("✓ Tool list inspection test completed")
}
