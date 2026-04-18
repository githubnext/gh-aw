//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBuildLogsFileResponse_WritesFile(t *testing.T) {
	// buildLogsFileResponse always writes to a file and returns file_path
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

	// Verify the file is in the cache directory (not the artifact directory)
	if !strings.HasPrefix(response.FilePath, mcpLogsCacheDir) {
		t.Errorf("File should be in cache dir %q, got %q", mcpLogsCacheDir, response.FilePath)
	}

	// Verify the file was actually created and contains the output
	data, err := os.ReadFile(response.FilePath)
	if err != nil {
		t.Fatalf("File should exist at file_path: %v", err)
	}
	if string(data) != output {
		t.Errorf("File content should match input: got %q, want %q", string(data), output)
	}

	// Cleanup
	_ = os.Remove(response.FilePath)
}

func TestBuildLogsFileResponse_SetsReadablePermissions(t *testing.T) {
	output := `{"summary":{"total_runs":1}}`
	result := buildLogsFileResponse(output)

	var response MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}
	if response.FilePath == "" {
		t.Fatal("Response should include file_path")
	}

	cacheInfo, err := os.Stat(mcpLogsCacheDir)
	if err != nil {
		t.Fatalf("Cache directory should exist: %v", err)
	}
	cachePerm := cacheInfo.Mode().Perm()
	if cachePerm&0o005 != 0o005 {
		t.Errorf("Cache directory should be accessible to other users (expected other r-x bits set), got mode %o", cachePerm)
	}

	fileInfo, err := os.Stat(response.FilePath)
	if err != nil {
		t.Fatalf("Cache file should exist: %v", err)
	}
	filePerm := fileInfo.Mode().Perm()
	if filePerm&0o004 != 0o004 {
		t.Errorf("Cache file should be world-readable (expected other read bit set), got mode %o", filePerm)
	}

	_ = os.Remove(response.FilePath)
}

func TestBuildLogsFileResponse_ContentDeduplication(t *testing.T) {
	// Same content should yield the same file path (content-addressed)
	output := `{"summary": {"total_runs": 5}, "runs": []}`

	result1 := buildLogsFileResponse(output)
	result2 := buildLogsFileResponse(output)

	var r1, r2 MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result1), &r1); err != nil {
		t.Fatalf("First response should be valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(result2), &r2); err != nil {
		t.Fatalf("Second response should be valid JSON: %v", err)
	}

	if r1.FilePath != r2.FilePath {
		t.Errorf("Identical content should produce the same file path: got %q and %q", r1.FilePath, r2.FilePath)
	}

	// Verify the file exists only once (not duplicated)
	if _, err := os.Stat(r1.FilePath); os.IsNotExist(err) {
		t.Errorf("Cached file should exist at %q", r1.FilePath)
	}

	// Cleanup
	_ = os.Remove(r1.FilePath)
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

	// Cleanup
	_ = os.Remove(response.FilePath)
}
