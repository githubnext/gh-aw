//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestBuildLogsDataEmptyRuns tests that buildLogsData works correctly with zero runs
func TestBuildLogsDataEmptyRuns(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	// Build logs data with no runs
	logsData := buildLogsData([]ProcessedRun{}, tmpDir, nil)

	// Verify summary has zero values but all fields present
	if logsData.Summary.TotalRuns != 0 {
		t.Errorf("Expected TotalRuns to be 0, got %d", logsData.Summary.TotalRuns)
	}

	// Verify runs array is empty
	if len(logsData.Runs) != 0 {
		t.Errorf("Expected empty runs array, got %d runs", len(logsData.Runs))
	}
}

// TestRenderLogsJSONEmptyRuns tests that JSON rendering works correctly with zero runs
func TestRenderLogsJSONEmptyRuns(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	// Create logs data with no runs
	logsData := buildLogsData([]ProcessedRun{}, tmpDir, nil)

	// Redirect stdout to capture JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Render JSON
	err := renderLogsJSON(logsData, true)
	if err != nil {
		t.Fatalf("Failed to render JSON: %v", err)
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var parsedData LogsData
	if err := json.Unmarshal([]byte(output), &parsedData); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify key fields exist and have correct zero values
	if parsedData.Summary.TotalRuns != 0 {
		t.Errorf("Expected TotalRuns 0, got %d", parsedData.Summary.TotalRuns)
	}

	// Verify the JSON summary is valid
	var jsonMap map[string]any
	if err := json.Unmarshal([]byte(output), &jsonMap); err != nil {
		t.Fatalf("Failed to parse JSON as map: %v", err)
	}

	summary, ok := jsonMap["summary"].(map[string]any)
	if !ok {
		t.Fatalf("Expected summary to be a map, got %T", jsonMap["summary"])
	}
	if _, exists := summary["total_runs"]; !exists {
		t.Fatalf("Expected total_runs field in summary. Summary: %+v", summary)
	}
	if _, exists := summary["total_tokens"]; exists {
		t.Fatalf("Expected total_tokens to be omitted when token data is unavailable. Summary: %+v", summary)
	}
}
