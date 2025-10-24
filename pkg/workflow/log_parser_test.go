package workflow

import (
	"strings"
	"testing"
)

func TestLogParserScriptMethods(t *testing.T) {
	t.Run("ClaudeEngine returns correct log parser script", func(t *testing.T) {
		engine := NewClaudeEngine()
		scriptName := engine.GetLogParserScriptId()
		if scriptName != "parse_claude_log" {
			t.Errorf("Expected 'parse_claude_log', got '%s'", scriptName)
		}
	})

	t.Run("CodexEngine returns correct log parser script", func(t *testing.T) {
		engine := NewCodexEngine()
		scriptName := engine.GetLogParserScriptId()
		if scriptName != "parse_codex_log" {
			t.Errorf("Expected 'parse_codex_log', got '%s'", scriptName)
		}
	})
}

func TestGetLogParserScript(t *testing.T) {
	t.Run("Get Claude log parser script", func(t *testing.T) {
		script := GetLogParserScript("parse_claude_log")
		if script == "" {
			t.Error("Expected non-empty script for parse_claude_log")
		}
		if !strings.Contains(script, "parseClaudeLog") {
			t.Error("Expected script to contain parseClaudeLog function")
		}
		if !strings.Contains(script, "tool_use") {
			t.Error("Expected script to contain tool_use logic")
		}
	})

	t.Run("Get Codex log parser script", func(t *testing.T) {
		script := GetLogParserScript("parse_codex_log")
		if script == "" {
			t.Error("Expected non-empty script for parse_codex_log")
		}
		if !strings.Contains(script, "parseCodexLog") {
			t.Error("Expected script to contain parseCodexLog function")
		}
	})

	t.Run("Get unknown log parser script returns empty", func(t *testing.T) {
		script := GetLogParserScript("unknown_parser")
		if script != "" {
			t.Error("Expected empty script for unknown parser")
		}
	})
}

// Smoke tests for log parsing functions
func TestParseClaudeLogSmoke(t *testing.T) {
	script := GetLogParserScript("parse_claude_log")
	if script == "" {
		t.Skip("parse_claude_log script not available")
	}

	// Test with minimal valid Claude log
	minimalClaudeLog := `[
  {
    "type": "system",
    "subtype": "init",
    "session_id": "test-123",
    "tools": ["Bash", "Read"]
  },
  {
    "type": "assistant",
    "message": {
      "content": [
        {
          "type": "text",
          "text": "I'll help you with this task."
        },
        {
          "type": "tool_use",
          "id": "tool_123",
          "name": "Bash", 
          "input": {
            "command": "echo 'Hello World'"
          }
        }
      ]
    }
  },
  {
    "type": "user",
    "message": {
      "content": [
        {
          "type": "tool_result",
          "tool_use_id": "tool_123",
          "content": "Hello World"
        }
      ]
    }
  },
  {
    "type": "result",
    "total_cost_usd": 0.0015,
    "usage": {
      "input_tokens": 150,
      "output_tokens": 50
    }
  }
]`

	result, err := runJSLogParser(script, minimalClaudeLog)
	if err != nil {
		t.Fatalf("Failed to parse minimal Claude log: %v", err)
	}

	// Verify essential sections are present
	if !strings.Contains(result, "🤖 Commands and Tools") {
		t.Error("Expected Claude log output to contain Commands and Tools section")
	}
	if !strings.Contains(result, "🤖 Reasoning") {
		t.Error("Expected Claude log output to contain Reasoning section")
	}
	if !strings.Contains(result, "echo 'Hello World'") {
		t.Error("Expected Claude log output to contain the bash command")
	}
	if !strings.Contains(result, "Total Cost") {
		t.Error("Expected Claude log output to contain cost information")
	}

	// Test with invalid JSON
	invalidLog := `{ invalid json }`
	result, err = runJSLogParser(script, invalidLog)
	if err != nil {
		t.Fatalf("Failed to parse invalid Claude log: %v", err)
	}
	if !strings.Contains(result, "Log format not recognized") {
		t.Error("Expected error message for invalid JSON in Claude log")
	}

	// Test with empty input
	result, err = runJSLogParser(script, "")
	if err != nil {
		t.Fatalf("Failed to parse empty Claude log: %v", err)
	}
	if !strings.Contains(result, "Log format not recognized") {
		t.Error("Expected error message for empty Claude log")
	}
}

// Test parsing initialization information from Claude logs
func TestParseClaudeLogInitialization(t *testing.T) {
	script := GetLogParserScript("parse_claude_log")
	if script == "" {
		t.Skip("parse_claude_log script not available")
	}

	// Test with initialization log containing system init entry
	initClaudeLog := `[
  {
    "type": "system",
    "subtype": "init",
    "cwd": "/home/runner/work/gh-aw/gh-aw",
    "session_id": "test-session-123",
    "tools": ["Task", "Bash", "Read", "mcp__github__search_issues", "mcp__github__create_issue"],
    "mcp_servers": [
      {"name": "github", "status": "connected"},
      {"name": "safeoutputs", "status": "failed"}
    ],
    "model": "claude-sonnet-4-20250514",
    "slash_commands": ["help", "status", "config"]
  }
]`

	result, err := runJSLogParser(script, initClaudeLog)
	if err != nil {
		t.Fatalf("Failed to parse initialization Claude log: %v", err)
	}

	// Verify initialization section is present
	if !strings.Contains(result, "🚀 Initialization") {
		t.Error("Expected Claude log output to contain Initialization section")
	}

	// Verify model information
	if !strings.Contains(result, "claude-sonnet-4-20250514") {
		t.Error("Expected Claude log output to contain model information")
	}

	// Verify session ID
	if !strings.Contains(result, "test-session-123") {
		t.Error("Expected Claude log output to contain session ID")
	}

	// Verify MCP servers section
	if !strings.Contains(result, "MCP Servers") {
		t.Error("Expected Claude log output to contain MCP Servers section")
	}

	// Verify specific server statuses
	if !strings.Contains(result, "✅ github (connected)") {
		t.Error("Expected Claude log output to show github server as connected")
	}
	if !strings.Contains(result, "❌ safeoutputs (failed)") {
		t.Error("Expected Claude log output to show safeoutputs server as failed")
	}

	// Verify tools section
	if !strings.Contains(result, "Available Tools") {
		t.Error("Expected Claude log output to contain Available Tools section")
	}

	// Verify slash commands section
	if !strings.Contains(result, "Slash Commands") {
		t.Error("Expected Claude log output to contain Slash Commands section")
	}
}

// Test parsing Claude logs in mixed format (debug logs + JSONL)
func TestParseClaudeMixedFormatLog(t *testing.T) {
	script := GetLogParserScript("parse_claude_log")
	if script == "" {
		t.Skip("parse_claude_log script not available")
	}

	// Test with mixed format log (debug logs + JSONL entries)
	mixedFormatLog := `2025-09-15T23:22:45.123Z [DEBUG] Initializing Claude Code CLI
2025-09-15T23:22:45.125Z [INFO] Session started
{"type":"system","subtype":"init","session_id":"test-123","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}
2025-09-15T23:22:45.130Z [DEBUG] Processing user prompt
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you with this task."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"echo 'Hello World'"}}]}}
2025-09-15T23:22:45.135Z [DEBUG] Executing bash command
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"Hello World"}]}}
2025-09-15T23:22:45.140Z [INFO] Workflow completed successfully
{"type":"result","total_cost_usd":0.0015,"usage":{"input_tokens":150,"output_tokens":50},"num_turns":1,"duration_ms":2000}
2025-09-15T23:22:45.145Z [DEBUG] Cleanup completed`

	result, err := runJSLogParser(script, mixedFormatLog)
	if err != nil {
		t.Fatalf("Failed to parse mixed format Claude log: %v", err)
	}

	// Verify essential sections are present
	if !strings.Contains(result, "🚀 Initialization") {
		t.Error("Expected mixed format Claude log output to contain Initialization section")
	}
	if !strings.Contains(result, "🤖 Commands and Tools") {
		t.Error("Expected mixed format Claude log output to contain Commands and Tools section")
	}
	if !strings.Contains(result, "🤖 Reasoning") {
		t.Error("Expected mixed format Claude log output to contain Reasoning section")
	}
	if !strings.Contains(result, "echo 'Hello World'") {
		t.Error("Expected mixed format Claude log output to contain the bash command")
	}
	if !strings.Contains(result, "Total Cost") {
		t.Error("Expected mixed format Claude log output to contain cost information")
	}
	if !strings.Contains(result, "test-123") {
		t.Error("Expected mixed format Claude log output to contain session ID")
	}

	// Test backward compatibility with pure JSON array format
	jsonArrayLog := `[
		{"type":"system","subtype":"init","session_id":"test-456","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"},
		{"type":"assistant","message":{"content":[{"type":"text","text":"Working on it."},{"type":"tool_use","id":"tool_456","name":"Bash","input":{"command":"ls -la"}}]}},
		{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_456","content":"total 0"}]}},
		{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":40},"num_turns":1}
	]`

	result, err = runJSLogParser(script, jsonArrayLog)
	if err != nil {
		t.Fatalf("Failed to parse JSON array Claude log: %v", err)
	}

	// Verify backward compatibility works
	if !strings.Contains(result, "🚀 Initialization") {
		t.Error("Expected JSON array Claude log output to contain Initialization section")
	}
	if !strings.Contains(result, "ls -la") {
		t.Error("Expected JSON array Claude log output to contain the bash command")
	}
	if !strings.Contains(result, "test-456") {
		t.Error("Expected JSON array Claude log output to contain session ID")
	}
}

// Test Go Claude engine mixed format parsing
func TestClaudeEngineMixedFormatParsing(t *testing.T) {
	engine := NewClaudeEngine()

	// Test with mixed format log (debug logs + JSONL entries)
	mixedFormatLog := `2025-09-15T23:22:45.123Z [DEBUG] Initializing Claude Code CLI
2025-09-15T23:22:45.125Z [INFO] Session started
{"type":"system","subtype":"init","session_id":"test-123","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}
2025-09-15T23:22:45.130Z [DEBUG] Processing user prompt
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you with this task."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"echo 'Hello World'"}}]}}
2025-09-15T23:22:45.135Z [DEBUG] Executing bash command
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"Hello World"}]}}
2025-09-15T23:22:45.140Z [INFO] Workflow completed successfully
{"type":"result","total_cost_usd":0.0015,"usage":{"input_tokens":150,"output_tokens":50},"num_turns":1,"duration_ms":2000}
2025-09-15T23:22:45.145Z [DEBUG] Cleanup completed`

	metrics := engine.ParseLogMetrics(mixedFormatLog, true)

	// Verify that metrics were extracted
	if metrics.TokenUsage != 200 {
		t.Errorf("Expected token usage 200 (150+50), got %d", metrics.TokenUsage)
	}
	if metrics.EstimatedCost != 0.0015 {
		t.Errorf("Expected cost 0.0015, got %f", metrics.EstimatedCost)
	}
	if metrics.Turns != 1 {
		t.Errorf("Expected 1 turn, got %d", metrics.Turns)
	}

	// Test backward compatibility with pure JSON array format
	jsonArrayLog := `[
		{"type":"system","subtype":"init","session_id":"test-456","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"},
		{"type":"assistant","message":{"content":[{"type":"text","text":"Working on it."},{"type":"tool_use","id":"tool_456","name":"Bash","input":{"command":"ls -la"}}]}},
		{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_456","content":"total 0"}]}},
		{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":40},"num_turns":1}
	]`

	metrics = engine.ParseLogMetrics(jsonArrayLog, true)

	// Verify backward compatibility
	if metrics.TokenUsage != 140 {
		t.Errorf("Expected token usage 140 (100+40), got %d", metrics.TokenUsage)
	}
	if metrics.EstimatedCost != 0.002 {
		t.Errorf("Expected cost 0.002, got %f", metrics.EstimatedCost)
	}
	if metrics.Turns != 1 {
		t.Errorf("Expected 1 turn, got %d", metrics.Turns)
	}
}
