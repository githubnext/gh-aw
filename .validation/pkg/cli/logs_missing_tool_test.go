//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
)

type missingToolsTestCase struct {
	name               string
	safeOutputContent  string
	expected           int
	expectTool         string
	expectReason       string
	expectAlternatives string
}

func missingToolsTestCases() []missingToolsTestCase {
	return []missingToolsTestCase{
		{
			name: "single_missing_tool_in_safe_output",
			safeOutputContent: `{
"items": [
{
"type": "missing_tool",
"tool": "terraform",
"reason": "Infrastructure automation needed",
"alternatives": "Manual setup",
"timestamp": "2024-01-01T12:00:00Z"
}
],
"errors": []
}`,
			expected:           1,
			expectTool:         "terraform",
			expectReason:       "Infrastructure automation needed",
			expectAlternatives: "Manual setup",
		},
		{
			name: "multiple_missing_tools_in_safe_output",
			safeOutputContent: `{
"items": [
{
"type": "missing_tool",
"tool": "docker",
"reason": "Need containerization",
"alternatives": "VM setup",
"timestamp": "2024-01-01T10:00:00Z"
},
{
"type": "missing_tool",
"tool": "kubectl",
"reason": "K8s management",
"timestamp": "2024-01-01T10:01:00Z"
},
{
"type": "create-issue",
"title": "Test Issue",
"body": "This should be ignored"
}
],
"errors": []
}`,
			expected:   2,
			expectTool: "docker",
		},
		{
			name: "no_missing_tools_in_safe_output",
			safeOutputContent: `{
"items": [
{
"type": "create-issue",
"title": "Test Issue",
"body": "No missing tools here"
}
],
"errors": []
}`,
			expected: 0,
		},
		{
			name: "empty_safe_output",
			safeOutputContent: `{
"items": [],
"errors": []
}`,
			expected: 0,
		},
		{
			name: "malformed_json",
			safeOutputContent: `{
"items": [
{
"type": "missing_tool"
"tool": "docker"
}
]
}`,
			expected: 0,
		},
	}
}

func writeMissingToolSafeOutput(t *testing.T, dir, content string) {
	t.Helper()
	safeOutputFile := filepath.Join(dir, constants.AgentOutputArtifactName)
	if err := os.WriteFile(safeOutputFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test safe output file: %v", err)
	}
}

func assertMissingToolExtraction(t *testing.T, tools []MissingToolReport, tt missingToolsTestCase, testRun WorkflowRun) {
	t.Helper()
	if len(tools) != tt.expected {
		t.Fatalf("Expected %d tools, got %d", tt.expected, len(tools))
	}
	if tt.expected == 0 {
		return
	}
	tool := tools[0]
	if tool.Tool != tt.expectTool {
		t.Errorf("Expected tool '%s', got '%s'", tt.expectTool, tool.Tool)
	}
	if tt.expectReason != "" && tool.Reason != tt.expectReason {
		t.Errorf("Expected reason '%s', got '%s'", tt.expectReason, tool.Reason)
	}
	if tt.expectAlternatives != "" && tool.Alternatives != tt.expectAlternatives {
		t.Errorf("Expected alternatives '%s', got '%s'", tt.expectAlternatives, tool.Alternatives)
	}
	if tool.WorkflowName != testRun.WorkflowName {
		t.Errorf("Expected workflow name '%s', got '%s'", testRun.WorkflowName, tool.WorkflowName)
	}
	if tool.RunID != testRun.DatabaseID {
		t.Errorf("Expected run ID %d, got %d", testRun.DatabaseID, tool.RunID)
	}
}

// TestExtractMissingToolsFromRun tests extracting missing tools from safe output artifact files.
func TestExtractMissingToolsFromRun(t *testing.T) {
	testRun := WorkflowRun{DatabaseID: 67890, WorkflowName: "Integration Test"}
	for _, tt := range missingToolsTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")
			writeMissingToolSafeOutput(t, tmpDir, tt.safeOutputContent)
			tools, err := extractMissingToolsFromRun(tmpDir, testRun, false)
			if err != nil {
				t.Fatalf("Error extracting missing tools: %v", err)
			}
			assertMissingToolExtraction(t, tools, tt, testRun)
		})
	}
}
