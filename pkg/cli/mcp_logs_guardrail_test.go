//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBuildLogsFileResponse_WritesFile(t *testing.T) {
	// buildLogsFileResponse always writes to a file and returns schema + file_path
	output := `{"summary": {"total_runs": 1}, "runs": []}`

	result := buildLogsFileResponse(output)

	// Verify the result is valid JSON
	var response MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	// Verify message is set
	if response.Message == "" {
		t.Error("Response should have a message")
	}

	// Verify file_path is set
	if response.FilePath == "" {
		t.Error("Response should have a file_path")
	}

	// Verify the file was actually created and contains the output
	data, err := os.ReadFile(response.FilePath)
	if err != nil {
		t.Fatalf("File should exist at file_path: %v", err)
	}
	if string(data) != output {
		t.Errorf("File content should match input: got %q, want %q", string(data), output)
	}

	// Verify schema is set
	if len(response.Schema.Fields) == 0 {
		t.Error("Response should include schema fields")
	}

	// Cleanup
	_ = os.Remove(response.FilePath)
}

func TestBuildLogsFileResponse_LargeOutput(t *testing.T) {
	// buildLogsFileResponse should always write to file regardless of output size
	largeOutput := strings.Repeat("x", 50000)

	result := buildLogsFileResponse(largeOutput)

	var response MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	if response.FilePath == "" {
		t.Error("Large output should also produce a file_path")
	}

	// Verify the file contains the large output
	data, err := os.ReadFile(response.FilePath)
	if err != nil {
		t.Fatalf("File should exist at file_path: %v", err)
	}
	if len(data) != len(largeOutput) {
		t.Errorf("File size mismatch: got %d, want %d", len(data), len(largeOutput))
	}

	// Cleanup
	_ = os.Remove(response.FilePath)
}

func TestBuildLogsFileResponse_ResponseStructure(t *testing.T) {
	output := `{"summary": {"total_runs": 2}}`

	result := buildLogsFileResponse(output)

	var response MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Should return valid JSON: %v", err)
	}

	if response.Message == "" {
		t.Error("JSON should have message field")
	}

	if response.FilePath == "" {
		t.Error("JSON should have file_path field")
	}

	if response.Schema.Type == "" {
		t.Error("JSON should have schema.type field")
	}

	if len(response.Schema.Fields) == 0 {
		t.Error("JSON should have schema.fields")
	}

	// Cleanup
	_ = os.Remove(response.FilePath)
}

func TestGetLogsDataSchema(t *testing.T) {
	schema := getLogsDataSchema()

	// Verify basic schema structure
	if schema.Type != "object" {
		t.Errorf("Schema type should be 'object', got '%s'", schema.Type)
	}

	if schema.Description == "" {
		t.Error("Schema should have a description")
	}

	// Verify expected fields are present
	expectedFields := []string{
		"summary",
		"runs",
		"tool_usage",
		"errors_and_warnings",
		"missing_tools",
		"mcp_failures",
		"access_log",
		"firewall_log",
		"continuation",
		"logs_location",
	}

	for _, field := range expectedFields {
		if _, ok := schema.Fields[field]; !ok {
			t.Errorf("Schema should have field '%s'", field)
		}
	}

	// Verify each field has type and description
	for fieldName, field := range schema.Fields {
		if field.Type == "" {
			t.Errorf("Field '%s' should have a type", fieldName)
		}
		if field.Description == "" {
			t.Errorf("Field '%s' should have a description", fieldName)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	// Test token estimation
	tests := []struct {
		text           string
		expectedTokens int
	}{
		{"", 0},
		{"x", 0},                        // 1 char / 4 = 0
		{"xxxx", 1},                     // 4 chars / 4 = 1
		{"xxxxxxxx", 2},                 // 8 chars / 4 = 2
		{strings.Repeat("x", 400), 100}, // 400 chars / 4 = 100
	}

	for _, tt := range tests {
		got := estimateTokens(tt.text)
		if got != tt.expectedTokens {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.text, got, tt.expectedTokens)
		}
	}
}
