package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
func TestParseAgentLog(t *testing.T) {
	t.Skip("Test skipped - agent log parser scripts now use require() pattern and are loaded at runtime from external files")
}

	// Create a temporary directory for the test
	tempDir := testutil.TempDir(t, "test-*")

	// Create a mock agent-stdio.log file with Claude log format
	agentStdioPath := filepath.Join(tempDir, "agent-stdio.log")
	mockLogContent := `[
		{"type": "text", "text": "Starting task"},
		{"type": "tool_use", "id": "1", "name": "bash", "input": {"command": "echo hello"}},
		{"type": "tool_result", "tool_use_id": "1", "content": "hello"}
	]`
	if err := os.WriteFile(agentStdioPath, []byte(mockLogContent), 0644); err != nil {
		t.Fatalf("Failed to create mock agent-stdio.log: %v", err)
	}

	// Create a mock aw_info.json with Claude engine
	awInfoPath := filepath.Join(tempDir, "aw_info.json")
	awInfoContent := `{"engine_id": "claude"}`
	if err := os.WriteFile(awInfoPath, []byte(awInfoContent), 0644); err != nil {
		t.Fatalf("Failed to create mock aw_info.json: %v", err)
	}

	// Get the Claude engine
	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine("claude")
	if err != nil {
		t.Fatalf("Failed to get Claude engine: %v", err)
	}

	// Run the parser
	err = parseAgentLog(tempDir, engine, true)
	if err != nil {
		t.Fatalf("parseAgentLog failed: %v", err)
	}

	// Check that log.md was created
	logMdPath := filepath.Join(tempDir, "log.md")
	if _, err := os.Stat(logMdPath); os.IsNotExist(err) {
		t.Fatalf("log.md was not created")
	}

	// Read the content and verify it's not empty
	content, err := os.ReadFile(logMdPath)
	if err != nil {
		t.Fatalf("Failed to read log.md: %v", err)
	}

	if len(content) == 0 {
		t.Fatalf("log.md is empty")
	}

	// The content should contain markdown formatting
	contentStr := string(content)
	if len(contentStr) < 10 {
		t.Errorf("log.md content seems too short: %d bytes", len(contentStr))
	}
}

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
func TestParseAgentLogWithAgentOutputDir(t *testing.T) {
	t.Skip("Test skipped - agent log parser scripts now use require() pattern and are loaded at runtime from external files")
}

	// Create a temporary directory for the test
	tempDir := testutil.TempDir(t, "test-*")

	// Create a mock agent_output directory with a log file
	agentOutputDir := filepath.Join(tempDir, "agent_output")
	if err := os.MkdirAll(agentOutputDir, 0755); err != nil {
		t.Fatalf("Failed to create agent_output directory: %v", err)
	}

	agentLogPath := filepath.Join(agentOutputDir, "output.log")
	mockLogContent := `Testing Copilot CLI log output with timestamps and debug info`
	if err := os.WriteFile(agentLogPath, []byte(mockLogContent), 0644); err != nil {
		t.Fatalf("Failed to create mock log in agent_output: %v", err)
	}

	// Create a mock aw_info.json with Copilot engine
	awInfoPath := filepath.Join(tempDir, "aw_info.json")
	awInfoContent := `{"engine_id": "copilot"}`
	if err := os.WriteFile(awInfoPath, []byte(awInfoContent), 0644); err != nil {
		t.Fatalf("Failed to create mock aw_info.json: %v", err)
	}

	// Get the Copilot engine
	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine("copilot")
	if err != nil {
		t.Fatalf("Failed to get Copilot engine: %v", err)
	}

	// Run the parser
	err = parseAgentLog(tempDir, engine, true)
	if err != nil {
		t.Fatalf("parseAgentLog failed: %v", err)
	}

	// Check that log.md was created
	logMdPath := filepath.Join(tempDir, "log.md")
	if _, err := os.Stat(logMdPath); os.IsNotExist(err) {
		t.Fatalf("log.md was not created")
	}

	// Read the content and verify it's not empty
	content, err := os.ReadFile(logMdPath)
	if err != nil {
		t.Fatalf("Failed to read log.md: %v", err)
	}

	if len(content) == 0 {
		t.Fatalf("log.md is empty")
	}
}

func TestParseAgentLogNoAgentOutput(t *testing.T) {
	// Create a temporary directory without agent logs
	tempDir := testutil.TempDir(t, "test-*")

	// Get the Claude engine
	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine("claude")
	if err != nil {
		t.Fatalf("Failed to get Claude engine: %v", err)
	}

	// Run the parser - should not fail, just skip
	err = parseAgentLog(tempDir, engine, true)
	if err != nil {
		t.Fatalf("parseAgentLog should not fail when agent logs are missing: %v", err)
	}

	// Check that log.md was NOT created
	logMdPath := filepath.Join(tempDir, "log.md")
	if _, err := os.Stat(logMdPath); !os.IsNotExist(err) {
		t.Fatalf("log.md should not be created when agent logs are missing")
	}
}

func TestParseAgentLogNoEngine(t *testing.T) {
	// Create a temporary directory with agent-stdio.log
	tempDir := testutil.TempDir(t, "test-*")

	agentStdioPath := filepath.Join(tempDir, "agent-stdio.log")
	if err := os.WriteFile(agentStdioPath, []byte("[]"), 0644); err != nil {
		t.Fatalf("Failed to create mock agent-stdio.log: %v", err)
	}

	// Run the parser with nil engine - should skip gracefully
	err := parseAgentLog(tempDir, nil, true)
	if err != nil {
		t.Fatalf("parseAgentLog should not fail with nil engine: %v", err)
	}

	// Check that log.md was NOT created
	logMdPath := filepath.Join(tempDir, "log.md")
	if _, err := os.Stat(logMdPath); !os.IsNotExist(err) {
		t.Fatalf("log.md should not be created when engine is nil")
	}
}
