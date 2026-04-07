//go:build integration

package cli

import (
	"encoding/json"
	"os"
	"testing"
)

// TestMCPServer_LogsAlwaysWritesFile tests that the logs tool always writes data to a file
func TestMCPServer_LogsAlwaysWritesFile(t *testing.T) {
	t.Run("buildLogsFileResponse always produces file_path and schema", func(t *testing.T) {
		// Verify that buildLogsFileResponse always writes to a file and returns schema
		output := `{"summary": {"total_runs": 1}, "runs": []}`
		result := buildLogsFileResponse(output)

		var response MCPLogsGuardrailResponse
		if err := json.Unmarshal([]byte(result), &response); err != nil {
			t.Fatalf("Response should be valid JSON: %v", err)
		}

		if response.Message == "" {
			t.Error("Response should have a message")
		}

		if response.FilePath == "" {
			t.Error("Response should always have a file_path")
		}

		// Verify the file was actually created
		if _, err := os.Stat(response.FilePath); os.IsNotExist(err) {
			t.Errorf("File should exist at file_path %q", response.FilePath)
		}

		if response.Schema.Type != "object" {
			t.Error("Schema should be object type")
		}

		if len(response.Schema.Fields) == 0 {
			t.Error("Schema should have fields")
		}

		// Verify expected fields in schema
		expectedFields := []string{"summary", "runs", "tool_usage", "errors_and_warnings"}
		for _, field := range expectedFields {
			if _, ok := response.Schema.Fields[field]; !ok {
				t.Errorf("Schema should include field '%s'", field)
			}
		}

		// Cleanup
		_ = os.Remove(response.FilePath)
	})

	t.Run("small output also gets written to file", func(t *testing.T) {
		// Even small outputs should be written to a file (no conditional logic)
		smallOutput := `{"summary": {"total_runs": 1}, "runs": []}`
		result := buildLogsFileResponse(smallOutput)

		var response MCPLogsGuardrailResponse
		if err := json.Unmarshal([]byte(result), &response); err != nil {
			t.Fatalf("Response should be valid JSON: %v", err)
		}

		if response.FilePath == "" {
			t.Error("Even small output should be written to a file")
		}

		// Cleanup
		_ = os.Remove(response.FilePath)
	})
}
