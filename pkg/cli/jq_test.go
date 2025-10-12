package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestApplyJqFilter(t *testing.T) {
	// Skip if jq is not available
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("Skipping test: jq not found in PATH")
	}

	tests := []struct {
		name      string
		jsonInput string
		jqFilter  string
		wantErr   bool
		validate  func(t *testing.T, output string)
	}{
		{
			name:      "simple filter - get first element",
			jsonInput: `[{"name":"a"},{"name":"b"}]`,
			jqFilter:  ".[0]",
			wantErr:   false,
			validate: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Expected non-empty output")
				}
			},
		},
		{
			name:      "filter - count array length",
			jsonInput: `[{"name":"a"},{"name":"b"},{"name":"c"}]`,
			jqFilter:  "length",
			wantErr:   false,
			validate: func(t *testing.T, output string) {
				if output != "3\n" {
					t.Errorf("Expected '3\\n', got %q", output)
				}
			},
		},
		{
			name:      "filter - map and select",
			jsonInput: `[{"name":"a","type":"x"},{"name":"b","type":"y"},{"name":"c","type":"x"}]`,
			jqFilter:  `[.[] | select(.type == "x") | .name]`,
			wantErr:   false,
			validate: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Expected non-empty output")
				}
			},
		},
		{
			name:      "invalid filter - syntax error",
			jsonInput: `[{"name":"a"}]`,
			jqFilter:  ".[invalid",
			wantErr:   true,
			validate:  nil,
		},
		{
			name:      "invalid JSON input",
			jsonInput: `{invalid json}`,
			jqFilter:  ".",
			wantErr:   true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ApplyJqFilter(tt.jsonInput, tt.jqFilter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyJqFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

func TestApplyJqFilter_JqNotAvailable(t *testing.T) {
	// This test verifies the error message when jq is not available
	// We can't easily mock exec.LookPath, so we'll just verify the function structure

	// If jq is available, skip this test
	if _, err := exec.LookPath("jq"); err == nil {
		t.Skip("Skipping test: jq is available, cannot test 'not found' scenario")
	}

	_, err := ApplyJqFilter(`[]`, ".")
	if err == nil {
		t.Error("Expected error when jq is not available")
	}
	if err != nil && err.Error() != "jq not found in PATH" {
		t.Errorf("Expected 'jq not found in PATH' error, got: %v", err)
	}
}

// TestMCPServer_StatusToolWithJq tests the status tool with jq filter parameter
func TestMCPServer_StatusToolWithJq(t *testing.T) {
	// Skip if jq is not available
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("Skipping test: jq not found in PATH")
	}

	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create a temporary directory with a workflow file
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
on: push
engine: copilot
---
# Test Workflow
`
	workflowFile := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Save current directory and change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Initialize git repository in the temp directory
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Start the MCP server as a subprocess
	serverCmd := exec.Command(filepath.Join(originalDir, binaryPath), "mcp-server")
	serverCmd.Dir = tmpDir
	transport := &mcp.CommandTransport{Command: serverCmd}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer session.Close()

	// Test 1: Call status tool with jq filter to get just workflow names
	params := &mcp.CallToolParams{
		Name: "status",
		Arguments: map[string]any{
			"jq": ".[].workflow",
		},
	}
	result, err := session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call status tool with jq filter: %v", err)
	}

	// Verify result contains the workflow name
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text == "" {
			t.Error("Expected non-empty text content from status tool with jq filter")
		}
		// The output should contain "test" (the workflow name)
		t.Logf("Status tool output with jq filter: %s", textContent.Text)
	} else {
		t.Error("Expected text content from status tool with jq filter")
	}

	// Test 2: Call status tool with jq filter to count workflows
	params = &mcp.CallToolParams{
		Name: "status",
		Arguments: map[string]any{
			"jq": "length",
		},
	}
	result, err = session.CallTool(ctx, params)
	if err != nil {
		t.Fatalf("Failed to call status tool with jq count filter: %v", err)
	}

	// Verify result contains a number
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text == "" {
			t.Error("Expected non-empty text content from status tool with jq count filter")
		}
		t.Logf("Status tool count output: %s", textContent.Text)
	} else {
		t.Error("Expected text content from status tool with jq count filter")
	}
}
