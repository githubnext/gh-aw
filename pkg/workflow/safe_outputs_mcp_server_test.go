package workflow

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// MCPRequest represents an MCP JSON-RPC 2.0 request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC 2.0 response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP JSON-RPC 2.0 error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPClient wraps communication with the MCP server
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Reader
	stderr *bufio.Reader
}

// NewMCPClient creates a new MCP client for testing
func NewMCPClient(t *testing.T, outputFile string, config map[string]interface{}) *MCPClient {
	t.Helper()

	// Set up environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("GITHUB_AW_SAFE_OUTPUTS=%s", outputFile))

	if config != nil {
		configJSON, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}
		env = append(env, fmt.Sprintf("GITHUB_AW_SAFE_OUTPUTS_CONFIG=%s", string(configJSON)))
	}

	// Start the MCP server
	cmd := exec.Command("node", "js/safe_outputs_mcp_server.cjs")
	cmd.Dir = filepath.Dir("") // Use current working directory context
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start MCP server: %v", err)
	}

	client := &MCPClient{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewReader(stdout),
		stderr: bufio.NewReader(stderr),
	}

	// Initialize the server
	initReq := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	_, err = client.SendRequest(initReq)
	if err != nil {
		client.Close()
		t.Fatalf("Failed to initialize MCP server: %v", err)
	}

	return client
}

// SendRequest sends a request to the MCP server and returns the response
func (c *MCPClient) SendRequest(req MCPRequest) (*MCPResponse, error) {
	// Serialize request
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send Content-Length header and body
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(reqJSON))
	if _, err := c.stdin.WriteString(header); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := c.stdin.Write(reqJSON); err != nil {
		return nil, fmt.Errorf("failed to write body: %w", err)
	}
	if err := c.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush: %w", err)
	}

	// Read response
	return c.ReadResponse()
}

// ReadResponse reads a response from the MCP server
func (c *MCPClient) ReadResponse() (*MCPResponse, error) {
	// Read Content-Length header
	line, err := c.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read header line: %w", err)
	}

	var contentLength int
	if _, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength); err != nil {
		return nil, fmt.Errorf("failed to parse content length from '%s': %w", strings.TrimSpace(line), err)
	}

	// Read empty line
	if _, err := c.stdout.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("failed to read empty line: %w", err)
	}

	// Read body
	body := make([]byte, contentLength)
	if _, err := c.stdout.Read(body); err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// Parse response
	var resp MCPResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// Close closes the MCP client
func (c *MCPClient) Close() {
	c.stdin.Flush()
	c.cmd.Process.Kill()
	c.cmd.Wait()
}

func TestSafeOutputsMCPServer_Initialize(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]interface{}{
		"create-issue": map[string]interface{}{
			"enabled": true,
			"max":     5,
		},
		"missing-tool": map[string]interface{}{
			"enabled": true,
		},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Server was already initialized in NewMCPClient, so if we got here, initialization worked
	t.Log("MCP server initialized successfully")
}

func TestSafeOutputsMCPServer_ListTools(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]interface{}{
		"create-issue":      map[string]interface{}{"enabled": true},
		"create-discussion": map[string]interface{}{"enabled": true},
		"missing-tool":      map[string]interface{}{"enabled": true},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Request tools list
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	resp, err := client.SendRequest(req)
	if err != nil {
		t.Fatalf("Failed to get tools list: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %+v", resp.Error)
	}

	// Check result structure
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be an object, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools to be an array, got %T", result["tools"])
	}

	// Verify enabled tools are present
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolObj, ok := tool.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected tool to be an object, got %T", tool)
		}

		name, ok := toolObj["name"].(string)
		if !ok {
			t.Fatalf("Expected tool name to be a string, got %T", toolObj["name"])
		}

		toolNames[i] = name
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

	config := map[string]interface{}{
		"create-issue": map[string]interface{}{
			"enabled": true,
			"max":     5,
		},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Call create_issue tool
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "create_issue",
			"arguments": map[string]interface{}{
				"title":  "Test Issue",
				"body":   "This is a test issue created by MCP server",
				"labels": []string{"bug", "test"},
			},
		},
	}

	resp, err := client.SendRequest(req)
	if err != nil {
		t.Fatalf("Failed to call create_issue: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %+v", resp.Error)
	}

	// Check response structure
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be an object, got %T", resp.Result)
	}

	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatalf("Expected content to be an array, got %T", result["content"])
	}

	if len(content) == 0 {
		t.Fatalf("Expected at least one content item")
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected content item to be an object, got %T", content[0])
	}

	text, ok := contentItem["text"].(string)
	if !ok {
		t.Fatalf("Expected text to be a string, got %T", contentItem["text"])
	}

	if !strings.Contains(text, "Issue creation queued") {
		t.Errorf("Expected response to mention issue creation, got: %s", text)
	}

	// Verify output file was written
	if err := verifyOutputFile(t, tempFile, "create-issue", map[string]interface{}{
		"title":  "Test Issue",
		"body":   "This is a test issue created by MCP server",
		"labels": []interface{}{"bug", "test"},
	}); err != nil {
		t.Fatalf("Output file verification failed: %v", err)
	}

	t.Log("create_issue tool executed successfully")
}

func TestSafeOutputsMCPServer_MissingTool(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]interface{}{
		"missing-tool": map[string]interface{}{
			"enabled": true,
		},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Call missing_tool
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "missing_tool",
			"arguments": map[string]interface{}{
				"tool":         "advanced-analyzer",
				"reason":       "Need to analyze complex data structures",
				"alternatives": "Could use basic analysis tools with manual processing",
			},
		},
	}

	resp, err := client.SendRequest(req)
	if err != nil {
		t.Fatalf("Failed to call missing_tool: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %+v", resp.Error)
	}

	// Verify output file was written
	if err := verifyOutputFile(t, tempFile, "missing-tool", map[string]interface{}{
		"tool":         "advanced-analyzer",
		"reason":       "Need to analyze complex data structures",
		"alternatives": "Could use basic analysis tools with manual processing",
	}); err != nil {
		t.Fatalf("Output file verification failed: %v", err)
	}

	t.Log("missing_tool executed successfully")
}

func TestSafeOutputsMCPServer_DisabledTool(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]interface{}{
		"create-issue": map[string]interface{}{
			"enabled": false, // Explicitly disabled
		},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Try to call disabled tool
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "create_issue",
			"arguments": map[string]interface{}{
				"title": "This should fail",
				"body":  "Tool is disabled",
			},
		},
	}

	resp, err := client.SendRequest(req)
	if err != nil {
		t.Fatalf("Failed to call disabled tool: %v", err)
	}

	// Should get an error
	if resp.Error == nil {
		t.Fatalf("Expected error for disabled tool, got success")
	}

	if !strings.Contains(resp.Error.Message, "create-issue safe-output is not enabled") && !strings.Contains(resp.Error.Message, "Tool 'create_issue' failed") {
		t.Errorf("Expected error about disabled tool, got: %s", resp.Error.Message)
	}

	t.Log("Disabled tool correctly rejected")
}

func TestSafeOutputsMCPServer_UnknownTool(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]interface{}{
		"create-issue": map[string]interface{}{"enabled": true},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Try to call unknown tool
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "nonexistent_tool",
			"arguments": map[string]interface{}{},
		},
	}

	resp, err := client.SendRequest(req)
	if err != nil {
		t.Fatalf("Failed to call unknown tool: %v", err)
	}

	// Should get a "Tool not found" error
	if resp.Error == nil {
		t.Fatalf("Expected error for unknown tool, got success")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601 (Method not found), got %d", resp.Error.Code)
	}

	if !strings.Contains(resp.Error.Message, "Tool not found") {
		t.Errorf("Expected 'Tool not found' error, got: %s", resp.Error.Message)
	}

	t.Log("Unknown tool correctly rejected")
}

func TestSafeOutputsMCPServer_MultipleTools(t *testing.T) {
	tempFile := createTempOutputFile(t)
	defer os.Remove(tempFile)

	config := map[string]interface{}{
		"create-issue":      map[string]interface{}{"enabled": true},
		"add-issue-comment": map[string]interface{}{"enabled": true},
	}

	client := NewMCPClient(t, tempFile, config)
	defer client.Close()

	// Call multiple tools in sequence
	tools := []struct {
		name         string
		args         map[string]interface{}
		expectedType string
	}{
		{
			name: "create_issue",
			args: map[string]interface{}{
				"title": "First Issue",
				"body":  "First test issue",
			},
			expectedType: "create-issue",
		},
		{
			name: "add_issue_comment",
			args: map[string]interface{}{
				"body": "This is a comment",
			},
			expectedType: "add-issue-comment",
		},
	}

	for i, tool := range tools {
		req := MCPRequest{
			JSONRPC: "2.0",
			ID:      10 + i,
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name":      tool.name,
				"arguments": tool.args,
			},
		}

		resp, err := client.SendRequest(req)
		if err != nil {
			t.Fatalf("Failed to call tool %s: %v", tool.name, err)
		}

		if resp.Error != nil {
			t.Fatalf("MCP error for tool %s: %+v", tool.name, resp.Error)
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
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Failed to parse output line %d: %v", i, err)
		}

		if entry["type"] != tools[i].expectedType {
			t.Errorf("Expected type %s for line %d, got %s", tools[i].expectedType, i, entry["type"])
		}
	}

	t.Log("Multiple tools executed successfully")
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

func verifyOutputFile(t *testing.T, filename string, expectedType string, expectedFields map[string]interface{}) error {
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

	var entry map[string]interface{}
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
func deepEqual(a, b interface{}) bool {
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
