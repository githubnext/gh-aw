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
		if !strings.Contains(script, "User instructions") {
			t.Error("Expected script to contain User instructions parsing")
		}
	})

	t.Run("Get unknown log parser script returns empty", func(t *testing.T) {
		script := GetLogParserScript("unknown_parser")
		if script != "" {
			t.Error("Expected empty script for unknown parser")
		}
	})
}
