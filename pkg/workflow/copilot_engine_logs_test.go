//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestCopilotEngineRenderGitHubMCPConfig(t *testing.T) {
	tests := []struct {
		name         string
		githubTool   map[string]any
		isLast       bool
		expectedStrs []string
	}{
		{
			name:       "GitHub MCP with default version",
			githubTool: nil,
			isLast:     false,
			expectedStrs: []string{
				`"github": {`,
				`"type": "stdio",`,
				`"container": "ghcr.io/github/github-mcp-server:` + string(constants.DefaultGitHubMCPServerVersion) + `"`,
				`"env": {`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_MCP_SERVER_TOKEN}"`,
				`},`,
			},
		},
		{
			name: "GitHub MCP with custom version",
			githubTool: map[string]any{
				"version": "v1.2.3",
			},
			isLast: true,
			expectedStrs: []string{
				`"github": {`,
				`"type": "stdio",`,
				`"container": "ghcr.io/github/github-mcp-server:v1.2.3"`,
				`"env": {`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_MCP_SERVER_TOKEN}"`,
				`}`,
			},
		},
		{
			name: "GitHub MCP with allowed tools",
			githubTool: map[string]any{
				"allowed": []string{"actions_list", "get_file_contents"},
			},
			isLast: true,
			expectedStrs: []string{
				`"github": {`,
				`"type": "stdio",`,
				`"container": "ghcr.io/github/github-mcp-server:` + string(constants.DefaultGitHubMCPServerVersion) + `"`,
				`"env": {`,
				`}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{}
			// Use unified renderer instead of direct method call
			renderer := NewMCPConfigRenderer(MCPRendererOptions{
				IncludeCopilotFields: true,
				InlineArgs:           true,
				Format:               "json",
				IsLast:               tt.isLast,
			})
			renderer.RenderGitHubMCP(&yaml, tt.githubTool, workflowData)
			output := yaml.String()

			for _, expected := range tt.expectedStrs {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nFull output:\n%s", expected, output)
				}
			}

			// Verify proper ending based on isLast
			if tt.isLast {
				if !strings.HasSuffix(strings.TrimSpace(output), "}") {
					t.Errorf("Expected output to end with '}' when isLast=true, got:\n%s", output)
				}
			} else {
				if !strings.HasSuffix(strings.TrimSpace(output), "},") {
					t.Errorf("Expected output to end with '},' when isLast=false, got:\n%s", output)
				}
			}
		})
	}
}

func TestCopilotEngineLogParsingUsesCorrectLogFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "copilot-log-parsing-test")

	// Create a test workflow with Copilot engine
	testContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    allowed: [list_issues]
---

# Test Copilot Log Parsing

This workflow tests that Copilot log parsing uses the correct log file path.
`

	testFile := filepath.Join(tmpDir, "test-copilot-log-parsing.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify that the log parsing step uses /tmp/gh-aw/sandbox/agent/logs/ instead of agent-stdio.log
	if !strings.Contains(lockStr, "GH_AW_AGENT_OUTPUT: /tmp/gh-aw/sandbox/agent/logs/") {
		t.Error("Expected GH_AW_AGENT_OUTPUT to be set to '/tmp/gh-aw/sandbox/agent/logs/' for Copilot engine")
	}

	// Verify that it's NOT using the agent-stdio.log path for parsing
	if strings.Contains(lockStr, "GH_AW_AGENT_OUTPUT: /tmp/gh-aw/agent-stdio.log") {
		t.Error("Expected GH_AW_AGENT_OUTPUT to NOT use '/tmp/gh-aw/agent-stdio.log' for Copilot engine")
	}

	t.Log("Successfully verified that Copilot log parsing uses /tmp/gh-aw/sandbox/agent/logs/")
}

func TestCopilotEngineParseLogMetrics_MultilineJSON(t *testing.T) {
	engine := NewCopilotEngine()

	// Read the test data file with multi-line JSON format
	testDataPath := filepath.Join("test_data", "copilot_debug_log.txt")
	logContent, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to read test data file: %v", err)
	}

	// Parse the log metrics
	metrics := engine.ParseLogMetrics(string(logContent), false)

	// Verify token usage is extracted from multi-line JSON blocks
	// Expected: 1524 + 89 + 1689 + 23 = 3325 tokens total
	expectedTokens := 3325
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}

	// Verify the parser handles multiple JSON response blocks
	// Log contains 2 responses with usage information
	if metrics.TokenUsage == 0 {
		t.Error("Token usage should not be zero when parsing Copilot debug logs with usage data")
	}
}

func TestCopilotEngineParseLogMetrics_SingleLineJSON(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with single-line JSON wrapped in debug log format (realistic format)
	// This tests backward compatibility with compact JSON logging
	singleLineLog := `2025-09-26T11:13:17.989Z [DEBUG] data:
2025-09-26T11:13:17.990Z [DEBUG] {"usage": {"prompt_tokens": 100, "completion_tokens": 50}}
2025-09-26T11:13:17.990Z [DEBUG] Workflow completed`
	metrics := engine.ParseLogMetrics(singleLineLog, false)

	// Should extract tokens from single-line JSON in data block
	expectedTokens := 150
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d from single-line JSON, got %d", expectedTokens, metrics.TokenUsage)
	}
}

func TestCopilotEngineParseLogMetrics_NoTokenData(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with log content that has no token data
	noTokenLog := `2025-09-26T11:13:11.798Z [DEBUG] Using model: claude-sonnet-4
2025-09-26T11:13:12.575Z [DEBUG] Starting workflow
2025-09-26T11:13:18.502Z [DEBUG] Workflow completed`

	metrics := engine.ParseLogMetrics(noTokenLog, false)

	// Token usage should be 0 when no usage data is present
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 when no usage data present, got %d", metrics.TokenUsage)
	}
}

func TestCopilotEngineExtractToolSizes(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name          string
		jsonStr       string
		expectedTools map[string]struct{ inputSize, outputSize int }
		expectError   bool
	}{
		{
			name: "tool call with arguments",
			jsonStr: `{
				"choices": [{
					"message": {
						"role": "assistant",
						"tool_calls": [{
							"id": "call_abc123",
							"type": "function",
							"function": {
								"name": "bash",
								"arguments": "{\"command\":\"echo 'test'\",\"description\":\"Test command\"}"
							}
						}]
					}
				}]
			}`,
			expectedTools: map[string]struct{ inputSize, outputSize int }{
				"bash": {inputSize: 54, outputSize: 0},
			},
		},
		{
			name: "multiple tool calls",
			jsonStr: `{
				"choices": [{
					"message": {
						"tool_calls": [{
							"function": {
								"name": "github",
								"arguments": "{\"owner\":\"githubnext\",\"repo\":\"gh-aw\"}"
							}
						}, {
							"function": {
								"name": "playwright",
								"arguments": "{\"url\":\"https://example.com\",\"action\":\"screenshot\"}"
							}
						}]
					}
				}]
			}`,
			expectedTools: map[string]struct{ inputSize, outputSize int }{
				"github":     {inputSize: 37, outputSize: 0},
				"playwright": {inputSize: 51, outputSize: 0},
			},
		},
		{
			name: "tool call without arguments",
			jsonStr: `{
				"choices": [{
					"message": {
						"tool_calls": [{
							"function": {
								"name": "bash"
							}
						}]
					}
				}]
			}`,
			expectedTools: map[string]struct{ inputSize, outputSize int }{
				"bash": {inputSize: 0, outputSize: 0},
			},
		},
		{
			name: "empty tool_calls array",
			jsonStr: `{
				"choices": [{
					"message": {
						"tool_calls": []
					}
				}]
			}`,
			expectedTools: map[string]struct{ inputSize, outputSize int }{},
		},
		{
			name:          "invalid JSON",
			jsonStr:       `{invalid json}`,
			expectedTools: map[string]struct{ inputSize, outputSize int }{},
			expectError:   true,
		},
		{
			name: "tool call in alternative message format",
			jsonStr: `{
				"message": {
					"tool_calls": [{
						"function": {
							"name": "edit",
							"arguments": "{\"path\":\"/test/file.txt\",\"content\":\"test content\"}"
						}
					}]
				}
			}`,
			expectedTools: map[string]struct{ inputSize, outputSize int }{
				"edit": {inputSize: 50, outputSize: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCallMap := make(map[string]*ToolCallInfo)
			engine.extractToolCallSizes(tt.jsonStr, toolCallMap, true)

			// Verify tool count
			if len(toolCallMap) != len(tt.expectedTools) {
				t.Errorf("Expected %d tools, got %d: %v", len(tt.expectedTools), len(toolCallMap), toolCallMap)
			}

			// Verify each tool's sizes
			for toolName, expectedSizes := range tt.expectedTools {
				toolInfo, exists := toolCallMap[toolName]
				if !exists {
					t.Errorf("Expected tool '%s' not found in tool call map", toolName)
					continue
				}

				if toolInfo.MaxInputSize != expectedSizes.inputSize {
					t.Errorf("Tool '%s': expected MaxInputSize %d, got %d",
						toolName, expectedSizes.inputSize, toolInfo.MaxInputSize)
				}

				if toolInfo.MaxOutputSize != expectedSizes.outputSize {
					t.Errorf("Tool '%s': expected MaxOutputSize %d, got %d",
						toolName, expectedSizes.outputSize, toolInfo.MaxOutputSize)
				}
			}
		})
	}
}

func TestCopilotEngineExtractToolSizes_MaxTracking(t *testing.T) {
	engine := NewCopilotEngine()
	toolCallMap := make(map[string]*ToolCallInfo)

	// First call with smaller arguments
	json1 := `{
		"choices": [{
			"message": {
				"tool_calls": [{
					"function": {
						"name": "bash",
						"arguments": "{\"cmd\":\"ls\"}"
					}
				}]
			}
		}]
	}`
	engine.extractToolCallSizes(json1, toolCallMap, false)

	// Second call with larger arguments
	json2 := `{
		"choices": [{
			"message": {
				"tool_calls": [{
					"function": {
						"name": "bash",
						"arguments": "{\"command\":\"echo 'This is a much longer command with more content'\"}"
					}
				}]
			}
		}]
	}`
	engine.extractToolCallSizes(json2, toolCallMap, false)

	// Third call with smaller arguments again
	json3 := `{
		"choices": [{
			"message": {
				"tool_calls": [{
					"function": {
						"name": "bash",
						"arguments": "{\"cmd\":\"pwd\"}"
					}
				}]
			}
		}]
	}`
	engine.extractToolCallSizes(json3, toolCallMap, false)

	// Verify that MaxInputSize tracks the maximum
	bashInfo, exists := toolCallMap["bash"]
	if !exists {
		t.Fatal("bash tool not found in tool call map")
	}

	// Should have tracked the largest input size (from json2)
	expectedMaxInput := len("{\"command\":\"echo 'This is a much longer command with more content'\"}")
	if bashInfo.MaxInputSize != expectedMaxInput {
		t.Errorf("Expected MaxInputSize %d (from largest call), got %d", expectedMaxInput, bashInfo.MaxInputSize)
	}

	// Call count should be 3
	if bashInfo.CallCount != 3 {
		t.Errorf("Expected CallCount 3, got %d", bashInfo.CallCount)
	}
}

func TestCopilotEngineParseLogMetrics_WithToolSizes(t *testing.T) {
	engine := NewCopilotEngine()

	// Log with tool calls containing size information
	logWithTools := `2025-09-26T11:13:17.989Z [DEBUG] response (Request-ID 00000-4ceedfde):
2025-09-26T11:13:17.989Z [DEBUG] data:
2025-09-26T11:13:17.990Z [DEBUG] {
2025-09-26T11:13:17.990Z [DEBUG]   "choices": [
2025-09-26T11:13:17.990Z [DEBUG]     {
2025-09-26T11:13:17.990Z [DEBUG]       "message": {
2025-09-26T11:13:17.990Z [DEBUG]         "tool_calls": [
2025-09-26T11:13:17.990Z [DEBUG]           {
2025-09-26T11:13:17.990Z [DEBUG]             "function": {
2025-09-26T11:13:17.990Z [DEBUG]               "name": "github",
2025-09-26T11:13:17.990Z [DEBUG]               "arguments": "{\"owner\":\"githubnext\",\"repo\":\"gh-aw\",\"method\":\"list_issues\"}"
2025-09-26T11:13:17.990Z [DEBUG]             }
2025-09-26T11:13:17.990Z [DEBUG]           }
2025-09-26T11:13:17.990Z [DEBUG]         ]
2025-09-26T11:13:17.990Z [DEBUG]       }
2025-09-26T11:13:17.990Z [DEBUG]     }
2025-09-26T11:13:17.990Z [DEBUG]   ],
2025-09-26T11:13:17.990Z [DEBUG]   "usage": {
2025-09-26T11:13:17.990Z [DEBUG]     "prompt_tokens": 100,
2025-09-26T11:13:17.990Z [DEBUG]     "completion_tokens": 50
2025-09-26T11:13:17.990Z [DEBUG]   }
2025-09-26T11:13:17.990Z [DEBUG] }
2025-09-26T11:13:18.000Z [DEBUG] Executing tool: github`

	metrics := engine.ParseLogMetrics(logWithTools, false)

	// Verify token usage
	if metrics.TokenUsage != 150 {
		t.Errorf("Expected token usage 150, got %d", metrics.TokenUsage)
	}

	// Verify tool info was extracted
	if len(metrics.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(metrics.ToolCalls))
	}

	githubTool := metrics.ToolCalls[0]
	if githubTool.Name != "github" {
		t.Errorf("Expected tool name 'github', got '%s'", githubTool.Name)
	}

	// Verify input size was extracted
	expectedInputSize := len("{\"owner\":\"githubnext\",\"repo\":\"gh-aw\",\"method\":\"list_issues\"}")
	if githubTool.MaxInputSize != expectedInputSize {
		t.Errorf("Expected MaxInputSize %d, got %d", expectedInputSize, githubTool.MaxInputSize)
	}

	// Output size should be 0 (not extracted from current log format)
	if githubTool.MaxOutputSize != 0 {
		t.Errorf("Expected MaxOutputSize 0, got %d", githubTool.MaxOutputSize)
	}
}
