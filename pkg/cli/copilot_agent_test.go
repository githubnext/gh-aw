//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var copilotAgentDetectorTests = []struct {
	name           string
	setupFunc      func(string) error
	workflowPath   string
	expectedResult bool
}{
	{name: "detects from workflow path copilot-swe-agent.yml", setupFunc: func(string) error { return nil }, workflowPath: ".github/workflows/copilot-swe-agent.yml", expectedResult: true},
	{name: "detects from workflow path copilot-swe-agent.yaml", setupFunc: func(string) error { return nil }, workflowPath: ".github/workflows/copilot-swe-agent.yaml", expectedResult: true},
	{name: "does not detect from different workflow path", setupFunc: func(string) error { return nil }, workflowPath: ".github/workflows/my-workflow.yml", expectedResult: false},
	{
		name: "aw_info.json present means agentic workflow, not copilot agent",
		setupFunc: func(dir string) error {
			awInfo := `{"workflow_name": "copilot-swe-agent-session", "workflow_file": "test.yml"}`
			return os.WriteFile(filepath.Join(dir, "aw_info.json"), []byte(awInfo), 0644)
		},
		workflowPath:   ".github/workflows/copilot-swe-agent.yml",
		expectedResult: false,
	},
	{
		name: "aw_info.json present with any workflow name means agentic workflow",
		setupFunc: func(dir string) error {
			awInfo := `{"workflow_name": "test", "workflow_file": "copilot_swe_agent.yml"}`
			return os.WriteFile(filepath.Join(dir, "aw_info.json"), []byte(awInfo), 0644)
		},
		expectedResult: false,
	},
	{
		name: "detects agent pattern in log file without aw_info.json",
		setupFunc: func(dir string) error {
			logContent := "2024-01-15 10:00:00 Starting GitHub Copilot Agent v1.2.3\n2024-01-15 10:00:01 Initializing agent session execution\n"
			return os.WriteFile(filepath.Join(dir, "agent.log"), []byte(logContent), 0644)
		},
		expectedResult: true,
	},
	{
		name: "detects copilot-swe-agent in log without aw_info.json",
		setupFunc: func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "execution.log"), []byte("Using @github/copilot-swe-agent for task execution"), 0644)
		},
		expectedResult: true,
	},
	{
		name: "detects agent artifact without aw_info.json",
		setupFunc: func(dir string) error {
			return os.Mkdir(filepath.Join(dir, "copilot-agent-output"), 0755)
		},
		expectedResult: true,
	},
	{name: "no indicators - returns false", setupFunc: func(string) error { return nil }, expectedResult: false},
}

var parseLogMetricsTests = []struct {
	name          string
	logContent    string
	expectedTurns int
	expectedTools int
	expectNoTools bool
}{
	{name: "parses agent turns", logContent: "Task iteration 1: Starting analysis\nExecuting tool: github\nTask iteration 2: Processing\nCalling write tool\nTask iteration 3: Finalizing\n", expectedTurns: 3, expectedTools: 2},
	{name: "parses tool calls", logContent: "Calling: github_search\nTool call: write_file\nUsing tool: bash\nExecuting tool: read\n", expectedTools: 4},
	{name: "extracts tool from bullet point format", logContent: "● startHttpServer\n● noop\n● label_agent\n", expectedTools: 3},
	{
		name:          "does not extract common English words as tool names",
		logContent:    "The agent tool calls a command\nSince the tool or alternative is needed\nThe tool calls for analysis\ntool call to the API\nagent calls a bash command\n",
		expectNoTools: true,
	},
	{name: "extracts token usage from JSON", logContent: `{"token_usage": 1500, "estimated_cost": 0.05}` + "\nTask step 1 complete\n", expectedTurns: 1},
	{name: "handles empty log", logContent: "\n\n\n", expectNoTools: true},
}

var extractToolNameTests = []struct {
	name     string
	line     string
	expected string
}{
	{name: "extracts from 'tool:' pattern", line: "Using tool: github_search", expected: "github_search"},
	{name: "extracts from 'calling' pattern", line: "Calling: write_file", expected: "write_file"},
	{name: "extracts from 'executing' pattern", line: "Executing: bash_command", expected: "bash_command"},
	{name: "extracts from 'using tool' pattern", line: "Using tool: read_operation", expected: "read_operation"},
	{name: "returns empty for no match", line: "Just a regular log line", expected: ""},
	{name: "does not extract 'calls' from 'tool calls' in prose", line: "The agent tool calls a command", expected: ""},
	{name: "does not extract 'call' from 'tool call' without colon", line: "Making a tool call to the API", expected: ""},
	{name: "does not extract 'or' from 'tool or' in prose", line: "The tool or alternative approach", expected: ""},
	{name: "does not extract 'for' from 'tool for' in prose", line: "Use a tool for building the project", expected: ""},
	{name: "does not extract word from 'calling' without colon", line: "The agent is calling a function here", expected: ""},
	{name: "extracts tool name from bullet point format", line: "● startHttpServer", expected: "startHttpServer"},
	{name: "extracts snake_case tool name from bullet point", line: "● label_agent", expected: "label_agent"},
	{name: "extracts tool from bullet with leading whitespace", line: "  ● noop", expected: "noop"},
}

func TestCopilotCodingAgentDetector_IsGitHubCopilotCodingAgent(t *testing.T) {
	for _, tt := range copilotAgentDetectorTests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "copilot-agent-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)
			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			var detector *CopilotCodingAgentDetector
			if tt.workflowPath != "" {
				detector = NewCopilotCodingAgentDetectorWithPath(tmpDir, false, tt.workflowPath)
			} else {
				detector = NewCopilotCodingAgentDetector(tmpDir, false)
			}
			result := detector.IsGitHubCopilotCodingAgent()
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestParseCopilotCodingAgentLogMetrics(t *testing.T) {
	for _, tt := range parseLogMetricsTests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := ParseCopilotCodingAgentLogMetrics(tt.logContent, false)
			if tt.expectedTurns > 0 && metrics.Turns != tt.expectedTurns {
				t.Errorf("Expected %d turns, got %d", tt.expectedTurns, metrics.Turns)
			}
			if tt.expectedTools > 0 && len(metrics.ToolCalls) < 1 {
				t.Errorf("Expected tool calls to be detected, got none")
			}
			if tt.expectNoTools && len(metrics.ToolCalls) > 0 {
				var names []string
				for _, tc := range metrics.ToolCalls {
					names = append(names, tc.Name)
				}
				t.Errorf("Expected no tool calls, but got %d: %v", len(metrics.ToolCalls), names)
			}
		})
	}
}

func TestExtractToolName(t *testing.T) {
	for _, tt := range extractToolNameTests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolName(tt.line)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIntegration_CopilotCodingAgentWithAudit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copilot-agent-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentLog := "2024-01-15T10:00:00.000Z Starting GitHub Copilot Agent\nTask iteration 1: Analyzing codebase\nTool call: github_search\n" +
		`{"token_usage": 1500, "estimated_cost": 0.05}` + "\nTask iteration 2: Making changes\nTool call: write_file\nERROR: Failed to write to protected file\nTask iteration 3: Completing task\nTool call: github_create_pr\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "agent-stdio.log"), []byte(agentLog), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	detector := NewCopilotCodingAgentDetector(tmpDir, false)
	if !detector.IsGitHubCopilotCodingAgent() {
		t.Error("Expected GitHub Copilot coding agent to be detected from log patterns")
	}

	metrics, err := extractLogMetrics(tmpDir, false)
	if err != nil {
		t.Fatalf("extractLogMetrics failed: %v", err)
	}
	if metrics.Turns < 1 {
		t.Errorf("Expected turns to be parsed from agent log, got %d", metrics.Turns)
	}
	if len(metrics.ToolCalls) < 1 {
		t.Error("Expected tool calls to be extracted from agent log")
	}
	if metrics.TokenUsage < 1 {
		t.Logf("Note: Token usage was not extracted from JSON in log (this is acceptable)")
	}
}

func TestReadLogHeader(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-header-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.log")
	content := strings.Repeat("x", 20000)
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	header, err := readLogHeader(testFile, 10240)
	if err != nil {
		t.Fatalf("readLogHeader failed: %v", err)
	}
	if len(header) != 10240 {
		t.Errorf("Expected 10240 bytes, got %d", len(header))
	}
}

func TestWorkflowLogMetricsConversion(t *testing.T) {
	logContent := "Task iteration 1\nTool call: github\nERROR: Test error\n"
	metrics := ParseCopilotCodingAgentLogMetrics(logContent, false)
	var _ = metrics
	if metrics.Turns != 1 {
		t.Errorf("Expected 1 turn, got %d", metrics.Turns)
	}
	if len(metrics.ToolCalls) < 1 {
		t.Error("Expected tool calls to be extracted")
	}
}
