//go:build !integration

package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestAuditDataJSONIncludesInputSizes verifies that JSON output includes input sizes
func TestAuditDataJSONIncludesInputSizes(t *testing.T) {
	run := WorkflowRun{
		DatabaseID:   888999,
		WorkflowName: "JSON Test",
		Status:       "completed",
		Conclusion:   "success",
		CreatedAt:    time.Now(),
		Event:        "push",
		HeadBranch:   "main",
		URL:          "https://github.com/test/repo/actions/runs/888999",
		LogsPath:     testutil.TempDir(t, "test-*"),
	}

	metrics := LogMetrics{
		ToolCalls: []workflow.ToolCallInfo{
			{
				Name:          "github_issue_read",
				CallCount:     2,
				MaxInputSize:  256,
				MaxOutputSize: 1024,
				MaxDuration:   1 * time.Second,
			},
		},
	}

	processedRun := ProcessedRun{
		Run: run,
	}

	// Build audit data
	auditData := buildAuditData(processedRun, metrics, nil)

	// Verify tool usage data includes input sizes
	if len(auditData.ToolUsage) == 0 {
		t.Fatal("Expected tool usage data, got none")
	}

	toolUsage := auditData.ToolUsage[0]
	if toolUsage.MaxInputSize != 256 {
		t.Errorf("Expected MaxInputSize 256, got %d", toolUsage.MaxInputSize)
	}
	if toolUsage.MaxOutputSize != 1024 {
		t.Errorf("Expected MaxOutputSize 1024, got %d", toolUsage.MaxOutputSize)
	}

	// Verify JSON serialization includes input sizes
	jsonData, err := json.Marshal(auditData)
	if err != nil {
		t.Fatalf("Failed to marshal audit data: %v", err)
	}

	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "max_input_size") {
		t.Error("JSON should contain max_input_size field")
	}
	if !strings.Contains(jsonStr, "\"max_input_size\":256") {
		t.Error("JSON should contain max_input_size value of 256")
	}
}

// TestToolUsageInfoStructure verifies the ToolUsageInfo structure has correct fields
func TestToolUsageInfoStructure(t *testing.T) {
	toolInfo := ToolUsageInfo{
		Name:          "test_tool",
		CallCount:     5,
		MaxInputSize:  128,
		MaxOutputSize: 512,
		MaxDuration:   "1s",
	}

	// Verify all fields are accessible
	if toolInfo.Name != "test_tool" {
		t.Error("Name field should be accessible")
	}
	if toolInfo.CallCount != 5 {
		t.Error("CallCount field should be accessible")
	}
	if toolInfo.MaxInputSize != 128 {
		t.Error("MaxInputSize field should be accessible")
	}
	if toolInfo.MaxOutputSize != 512 {
		t.Error("MaxOutputSize field should be accessible")
	}
	if toolInfo.MaxDuration != "1s" {
		t.Error("MaxDuration field should be accessible")
	}

	// Verify JSON tags are correct
	jsonData, err := json.Marshal(toolInfo)
	if err != nil {
		t.Fatalf("Failed to marshal tool info: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal tool info: %v", err)
	}

	if _, exists := parsed["max_input_size"]; !exists {
		t.Error("JSON should have max_input_size field")
	}
	if _, exists := parsed["max_output_size"]; !exists {
		t.Error("JSON should have max_output_size field")
	}
}
