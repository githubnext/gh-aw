package workflow

import (
	"strings"
	"testing"
)

func TestLogParserScriptMethods(t *testing.T) {
	t.Run("ClaudeEngine returns correct log parser script", func(t *testing.T) {
		engine := NewClaudeEngine()
		scriptName := engine.GetLogParserScript()
		if scriptName != "parse_claude_log" {
			t.Errorf("Expected 'parse_claude_log', got '%s'", scriptName)
		}
	})

	t.Run("CodexEngine returns correct log parser script", func(t *testing.T) {
		engine := NewCodexEngine()
		scriptName := engine.GetLogParserScript()
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
