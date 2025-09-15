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
	if !strings.Contains(result, "Error parsing Claude log") {
		t.Error("Expected error message for invalid JSON in Claude log")
	}

	// Test with empty input
	result, err = runJSLogParser(script, "")
	if err != nil {
		t.Fatalf("Failed to parse empty Claude log: %v", err)
	}
	if !strings.Contains(result, "Error parsing Claude log") {
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
      {"name": "safe_outputs", "status": "failed"}
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
	if !strings.Contains(result, "❌ safe_outputs (failed)") {
		t.Error("Expected Claude log output to show safe_outputs server as failed")
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
