package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestExtractMissingToolsFromRun tests extracting missing tools from safe output artifact files
func TestExtractMissingToolsFromRun(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	testRun := WorkflowRun{
		DatabaseID:   67890,
		WorkflowName: "Integration Test",
	}

	tests := []struct {
		name               string
		safeOutputContent  string
		expected           int
		expectTool         string
		expectReason       string
		expectAlternatives string
	}{
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
			expected: 0, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the safe output artifact file
			safeOutputFile := filepath.Join(tmpDir, constants.AgentOutputArtifactName)
			err := os.WriteFile(safeOutputFile, []byte(tt.safeOutputContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test safe output file: %v", err)
			}

			// Extract missing tools
			tools, err := extractMissingToolsFromRun(tmpDir, testRun, false)
			if err != nil {
				t.Fatalf("Error extracting missing tools: %v", err)
			}

			if len(tools) != tt.expected {
				t.Errorf("Expected %d tools, got %d", tt.expected, len(tools))
				return
			}

			if tt.expected > 0 && len(tools) > 0 {
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

				// Check that run information was populated
				if tool.WorkflowName != testRun.WorkflowName {
					t.Errorf("Expected workflow name '%s', got '%s'", testRun.WorkflowName, tool.WorkflowName)
				}

				if tool.RunID != testRun.DatabaseID {
					t.Errorf("Expected run ID %d, got %d", testRun.DatabaseID, tool.RunID)
				}
			}

			// Clean up for next test
			os.Remove(safeOutputFile)
		})
	}
}

// TestDisplayMissingToolsAnalysis tests the display functionality
func TestDisplayMissingToolsAnalysis(t *testing.T) {
	// This is a smoke test to ensure the function doesn't panic
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   1001,
				WorkflowName: "Workflow A",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "docker",
					Reason:       "Containerization needed",
					Alternatives: "VM setup",
					WorkflowName: "Workflow A",
					RunID:        1001,
				},
				{
					Tool:         "kubectl",
					Reason:       "K8s management",
					WorkflowName: "Workflow A",
					RunID:        1001,
				},
			},
		},
		{
			Run: WorkflowRun{
				DatabaseID:   1002,
				WorkflowName: "Workflow B",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "docker",
					Reason:       "Need containers for deployment",
					WorkflowName: "Workflow B",
					RunID:        1002,
				},
			},
		},
	}

	// Test non-verbose mode (should not panic)
	displayMissingToolsAnalysis(processedRuns, false)

	// Test verbose mode (should not panic)
	displayMissingToolsAnalysis(processedRuns, true)
}

// TestDisplayMissingToolsAnalysisEmpty tests display with no missing tools
func TestDisplayMissingToolsAnalysisEmpty(t *testing.T) {
	// Test with empty processed runs (should not display anything)
	emptyRuns := []ProcessedRun{}
	displayMissingToolsAnalysis(emptyRuns, false)
	displayMissingToolsAnalysis(emptyRuns, true)

	// Test with runs that have no missing tools (should not display anything)
	runsWithoutMissingTools := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   2001,
				WorkflowName: "Clean Workflow",
			},
			MissingTools: []MissingToolReport{}, // Empty slice
		},
	}
	displayMissingToolsAnalysis(runsWithoutMissingTools, false)
	displayMissingToolsAnalysis(runsWithoutMissingTools, true)
}
