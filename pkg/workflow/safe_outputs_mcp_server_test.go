package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPTestClient wraps the MCP Go SDK client for testing
type MCPTestClient struct {
	client  *mcp.Client
	session *mcp.ClientSession
	cmd     *exec.Cmd
}

// NewMCPTestClient creates a new MCP client using the Go SDK
func NewMCPTestClient(t *testing.T, outputFile string, config map[string]any) *MCPTestClient {
	t.Helper()

	// Set up environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("GITHUB_AW_SAFE_OUTPUTS=%s", outputFile))

	// Add required environment variables for upload_asset tool
	env = append(env, "GITHUB_AW_ASSETS_BRANCH=test-assets")
	env = append(env, "GITHUB_AW_ASSETS_MAX_SIZE_KB=10240")
	env = append(env, "GITHUB_AW_ASSETS_ALLOWED_EXTS=.png,.jpg,.jpeg")
	env = append(env, "GITHUB_SERVER_URL=https://github.com")
	env = append(env, "GITHUB_REPOSITORY=test/repo")

	if config != nil {
		configJSON, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}
		env = append(env, fmt.Sprintf("GITHUB_AW_SAFE_OUTPUTS_CONFIG=%s", string(configJSON)))
	}

	// Create command for the MCP server
	cmd := exec.Command("node", "js/safe_outputs_mcp_server.cjs")
	cmd.Dir = filepath.Dir("") // Use current working directory context
	cmd.Env = env

	// Create MCP client with command transport
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcp.CommandTransport{Command: cmd}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}

	return &MCPTestClient{
		client:  client,
		session: session,
		cmd:     cmd,
	}
}

// CallTool calls a tool using the MCP Go SDK
func (c *MCPTestClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error) {
	params := &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	}
	return c.session.CallTool(ctx, params)
}

// ListTools lists available tools using the MCP Go SDK
func (c *MCPTestClient) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	return c.session.ListTools(ctx, &mcp.ListToolsParams{})
}

// Close closes the MCP client and cleans up resources
func (c *MCPTestClient) Close() {
	if c.session != nil {
		c.session.Close()
	}
}

func TestSafeOutputsMCPServer_Initialize(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"create-issue": map[string]any{
			"enabled": true,
			"max":     5,
		},
		"missing-tool": map[string]any{
			"enabled": true,
		},
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// If we got here, initialization worked (handled by Connect in the SDK)
	t.Log("MCP server initialized successfully using Go MCP SDK")
}

func TestSafeOutputsMCPServer_ListTools(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"create-issue":      true,
		"create-discussion": true,
		"missing-tool":      true,
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to get tools list: %v", err)
	}

	// Verify enabled tools are present
	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}

	expectedTools := []string{"create_issue", "create_discussion", "missing_tool"}
	for _, expected := range expectedTools {
		found := false
		for _, actual := range toolNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool '%s' not found in tools list: %v", expected, toolNames)
		}
	}

	t.Logf("Found tools: %v", toolNames)
}

func TestSafeOutputsMCPServer_CreateIssue(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"create-issue": map[string]any{
			"enabled": true,
			"max":     5,
		},
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// Call create-issue tool
	ctx := context.Background()
	result, err := client.CallTool(ctx, "create-issue", map[string]any{
		"title":  "Test Issue",
		"body":   "This is a test issue created by MCP server",
		"labels": []string{"bug", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to call create-issue: %v", err)
	}

	// Check response structure
	if len(result.Content) == 0 {
		t.Fatalf("Expected at least one content item")
	}

	// Type assert to text content (should be safe since we're generating text content)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Expected first content item to be text content, got %T", result.Content[0])
	}

	if !strings.Contains(textContent.Text, "success") {
		t.Errorf("Expected response to mention issue creation, got: %s", textContent.Text)
	}

	// Verify output file was written (expect underscore format after normalization)
	if err := verifyOutputFile(t, tempFile, "create_issue", map[string]any{
		"title":  "Test Issue",
		"body":   "This is a test issue created by MCP server",
		"labels": []any{"bug", "test"},
	}); err != nil {
		t.Fatalf("Output file verification failed: %v", err)
	}

	t.Log("create-issue tool executed successfully using Go MCP SDK")
}

func TestSafeOutputsMCPServer_MissingTool(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"missing-tool": map[string]any{
			"enabled": true,
		},
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// Call missing-tool
	ctx := context.Background()
	_, err := client.CallTool(ctx, "missing-tool", map[string]any{
		"tool":         "advanced-analyzer",
		"reason":       "Need to analyze complex data structures",
		"alternatives": "Could use basic analysis tools with manual processing",
	})
	if err != nil {
		t.Fatalf("Failed to call missing-tool: %v", err)
	}

	// Verify output file was written (expect underscore format after normalization)
	if err := verifyOutputFile(t, tempFile, "missing_tool", map[string]any{
		"tool":         "advanced-analyzer",
		"reason":       "Need to analyze complex data structures",
		"alternatives": "Could use basic analysis tools with manual processing",
	}); err != nil {
		t.Fatalf("Output file verification failed: %v", err)
	}

	t.Log("missing-tool executed successfully using Go MCP SDK")
}

func TestSafeOutputsMCPServer_UnknownTool(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"create-issue": true,
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// Try to call unknown tool
	ctx := context.Background()
	_, err := client.CallTool(ctx, "nonexistent_tool", map[string]any{})

	// Should get a "Tool not found" error
	if err == nil {
		t.Fatalf("Expected error for unknown tool, got success")
	}

	if !strings.Contains(err.Error(), "Tool not found") {
		t.Errorf("Expected 'Tool not found' error, got: %s", err.Error())
	}

	t.Log("Unknown tool correctly rejected using Go MCP SDK")
}

func TestSafeOutputsMCPServer_MultipleTools(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"create-issue": true,
		"add-comment":  true,
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// Call multiple tools in sequence
	tools := []struct {
		name         string
		args         map[string]any
		expectedType string
	}{
		{
			name: "create-issue",
			args: map[string]any{
				"title": "First Issue",
				"body":  "First test issue",
			},
			expectedType: "create_issue",
		},
		{
			name: "add-comment",
			args: map[string]any{
				"body":        "This is a comment",
				"item_number": 1,
			},
			expectedType: "add_comment",
		},
	}

	ctx := context.Background()
	for _, tool := range tools {
		_, err := client.CallTool(ctx, tool.name, tool.args)
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool.name, err)
		}
	}

	// Verify multiple entries in output file
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != len(tools) {
		t.Fatalf("Expected %d output lines, got %d", len(tools), len(lines))
	}

	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Failed to parse output line %d: %v", i, err)
		}

		if entry["type"] != tools[i].expectedType {
			t.Errorf("Expected type %s for line %d, got %s", tools[i].expectedType, i, entry["type"])
		}
	}

	t.Log("Multiple tools executed successfully using Go MCP SDK")
}

// Helper functions

func createTempOutputFile(t *testing.T) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "safe_outputs_test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	return tmpFile.Name()
}

func verifyOutputFile(t *testing.T, filename string, expectedType string, expectedFields map[string]any) error {
	t.Helper()

	// Wait a bit for file to be written
	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	if len(content) == 0 {
		return fmt.Errorf("output file is empty")
	}

	// Parse the JSON line
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	lastLine := lines[len(lines)-1]

	var entry map[string]any
	if err := json.Unmarshal([]byte(lastLine), &entry); err != nil {
		return fmt.Errorf("failed to parse output entry: %w", err)
	}

	// Check type
	if entry["type"] != expectedType {
		return fmt.Errorf("expected type %s, got %s", expectedType, entry["type"])
	}

	// Check expected fields
	for key, expectedValue := range expectedFields {
		actualValue, exists := entry[key]
		if !exists {
			return fmt.Errorf("expected field %s not found", key)
		}

		// Handle different types appropriately
		if !deepEqual(actualValue, expectedValue) {
			return fmt.Errorf("field %s: expected %v, got %v", key, expectedValue, actualValue)
		}
	}

	return nil
}

// Simple deep equality check for test purposes
func deepEqual(a, b any) bool {
	aBytes, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bBytes, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(aBytes, bBytes)
}

func TestSafeOutputsMCPServer_PublishAsset(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"upload_asset": true,
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// Create a test file to publish (using allowed extension)
	testFilePath := "/tmp/gh-aw/test-asset.png"
	testContent := "This is a test asset file."

	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	// Call upload_asset tool
	ctx := context.Background()
	result, err := client.CallTool(ctx, "upload_asset", map[string]any{
		"path": testFilePath,
	})
	if err != nil {
		t.Fatalf("Failed to call upload_asset: %v", err)
	}

	// Check response structure
	if len(result.Content) == 0 {
		t.Fatalf("Expected at least one content item")
	}

	// Type assert to text content (should be safe since we're generating text content)
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Expected first content item to be text content, got %T", result.Content[0])
	}

	if !strings.Contains(textContent.Text, "raw.githubusercontent.com") {
		t.Errorf("Expected response to contain URL with raw.githubusercontent.com, got: %s", textContent.Text)
	}

	// Verify the output file contains the expected entry (expect underscore format after normalization)
	if err := verifyOutputFile(t, tempFile, "upload_asset", map[string]any{
		"type": "upload_asset",
	}); err != nil {
		t.Fatalf("Output file verification failed: %v", err)
	}
}

func TestSafeOutputsMCPServer_PublishAsset_PathValidation(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]any{
		"upload_asset": true,
	}

	client := NewMCPTestClient(t, tempFile, config)
	defer client.Close()

	// Test valid paths first - /tmp should be allowed
	testFilePath := "/tmp/gh-aw/test-asset-validation.png"
	testContent := "This is a test file for path validation."

	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	ctx := context.Background()

	// This should succeed (file in /tmp)
	_, err := client.CallTool(ctx, "upload_asset", map[string]any{
		"path": testFilePath,
	})
	if err != nil {
		t.Fatalf("Expected /tmp file to be allowed, but got error: %v", err)
	}

	// Test invalid path - should be rejected
	invalidPath := "/etc/passwd"
	_, err = client.CallTool(ctx, "upload_asset", map[string]any{
		"path": invalidPath,
	})
	if err == nil {
		t.Fatalf("Expected file outside workspace/tmp to be rejected, but it was allowed")
	}

	// Check that the error mentions it's an error (could be wrapped)
	t.Logf("Got expected error for invalid path: %v", err)
	// Just verify that an error occurred - the exact message might be wrapped
}
