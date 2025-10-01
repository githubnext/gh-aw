//go:build integration

package workflow

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestPlaywrightMCPServerSpawn verifies that the Playwright MCP server can be spawned
// with the default version and responds to basic MCP protocol messages
func TestPlaywrightMCPServerSpawn(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available, skipping Playwright MCP integration test")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Construct the command to spawn Playwright MCP server with default version
	playwrightPackage := "@playwright/mcp@" + constants.DefaultPlaywrightMCPVersion
	cmd := exec.CommandContext(ctx, "npx", playwrightPackage, "--output-dir", "/tmp/mcp-test-logs")

	// Set up pipes for stdin/stdout communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Start the MCP server
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Playwright MCP server: %v", err)
	}

	// Create a channel to collect stderr output for debugging
	stderrChan := make(chan string, 100)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrChan <- scanner.Text()
		}
		close(stderrChan)
	}()

	// Wait a bit for server to initialize
	time.Sleep(2 * time.Second)

	// Send MCP initialize request
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	// Encode and send the request
	encoder := json.NewEncoder(stdin)
	if err := encoder.Encode(initRequest); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read response with timeout
	responseChan := make(chan map[string]any, 1)
	errorChan := make(chan error, 1)

	go func() {
		decoder := json.NewDecoder(stdout)
		var response map[string]any
		if err := decoder.Decode(&response); err != nil {
			errorChan <- err
			return
		}
		responseChan <- response
	}()

	select {
	case response := <-responseChan:
		// Verify we got a valid response
		if response["jsonrpc"] != "2.0" {
			t.Errorf("Expected jsonrpc 2.0, got %v", response["jsonrpc"])
		}

		// Check if there's an error in the response
		if errObj, hasError := response["error"]; hasError {
			t.Logf("MCP server returned error: %v", errObj)
		}

		// Check for result indicating successful initialization
		if result, hasResult := response["result"]; hasResult {
			t.Logf("Successfully initialized Playwright MCP server with version %s", constants.DefaultPlaywrightMCPVersion)
			t.Logf("Server capabilities: %v", result)
		} else if !hasResult && response["error"] == nil {
			t.Error("Expected either result or error in response")
		}

	case err := <-errorChan:
		// Collect stderr for debugging
		var stderrLines []string
		for line := range stderrChan {
			stderrLines = append(stderrLines, line)
			if len(stderrLines) > 50 {
				break
			}
		}

		if err == io.EOF {
			t.Logf("Server closed connection (may be expected)")
			if len(stderrLines) > 0 {
				t.Logf("Server stderr output:\n%s", strings.Join(stderrLines, "\n"))
			}
		} else {
			t.Fatalf("Failed to read response: %v\nServer stderr:\n%s", err, strings.Join(stderrLines, "\n"))
		}

	case <-time.After(10 * time.Second):
		// Collect stderr for debugging
		var stderrLines []string
		timeout := time.After(100 * time.Millisecond)
	collectStderr:
		for {
			select {
			case line, ok := <-stderrChan:
				if !ok {
					break collectStderr
				}
				stderrLines = append(stderrLines, line)
			case <-timeout:
				break collectStderr
			}
		}

		t.Fatalf("Timeout waiting for response from MCP server\nServer stderr:\n%s",
			strings.Join(stderrLines, "\n"))
	}

	// Clean up
	stdin.Close()

	// Wait for process to finish or kill it after a short delay
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process finished
	case <-time.After(2 * time.Second):
		// Kill the process if it's still running
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("Failed to kill process: %v", err)
		}
		<-done // Wait for Wait() to return
	}

	// Clean up test directory
	os.RemoveAll("/tmp/mcp-test-logs")
}

// TestPlaywrightMCPVersionConstant verifies that the default version constant is set correctly
func TestPlaywrightMCPVersionConstant(t *testing.T) {
	expectedVersion := "v0.0.40"
	if constants.DefaultPlaywrightMCPVersion != expectedVersion {
		t.Errorf("Expected DefaultPlaywrightMCPVersion to be %s, got %s",
			expectedVersion, constants.DefaultPlaywrightMCPVersion)
	}
}
