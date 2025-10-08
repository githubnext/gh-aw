package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestParseAgentLog(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a mock agent_output.json file with Claude log format
	agentOutputPath := filepath.Join(tempDir, "agent_output.json")
	mockLogContent := `[
		{"type": "text", "text": "Starting task"},
		{"type": "tool_use", "id": "1", "name": "bash", "input": {"command": "echo hello"}},
		{"type": "tool_result", "tool_use_id": "1", "content": "hello"}
	]`
	if err := os.WriteFile(agentOutputPath, []byte(mockLogContent), 0644); err != nil {
		t.Fatalf("Failed to create mock agent_output.json: %v", err)
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

func TestParseAgentLogNoAgentOutput(t *testing.T) {
	// Create a temporary directory without agent_output.json
	tempDir := t.TempDir()

	// Get the Claude engine
	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine("claude")
	if err != nil {
		t.Fatalf("Failed to get Claude engine: %v", err)
	}

	// Run the parser - should not fail, just skip
	err = parseAgentLog(tempDir, engine, true)
	if err != nil {
		t.Fatalf("parseAgentLog should not fail when agent_output.json is missing: %v", err)
	}

	// Check that log.md was NOT created
	logMdPath := filepath.Join(tempDir, "log.md")
	if _, err := os.Stat(logMdPath); !os.IsNotExist(err) {
		t.Fatalf("log.md should not be created when agent_output.json is missing")
	}
}

func TestParseAgentLogNoEngine(t *testing.T) {
	// Create a temporary directory with agent_output.json
	tempDir := t.TempDir()

	agentOutputPath := filepath.Join(tempDir, "agent_output.json")
	if err := os.WriteFile(agentOutputPath, []byte("[]"), 0644); err != nil {
		t.Fatalf("Failed to create mock agent_output.json: %v", err)
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
