package workflow

import (
"strings"
"testing"
)

// TestTokenCountingConsistency verifies that Go and JavaScript parsers calculate total tokens the same way
func TestTokenCountingConsistency(t *testing.T) {
// Test log with all token types including cache tokens
claudeLogWithCache := `[
  {
    "type": "system",
    "subtype": "init",
    "session_id": "test-token-count",
    "tools": ["Bash", "Read"],
    "model": "claude-sonnet-4-20250514"
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
    "total_cost_usd": 0.0025,
    "usage": {
      "input_tokens": 1500,
      "output_tokens": 500,
      "cache_creation_input_tokens": 200,
      "cache_read_input_tokens": 100
    },
    "num_turns": 1,
    "duration_ms": 3000
  }
]`

// Test with Go parser
engine := NewClaudeEngine()
goMetrics := engine.ParseLogMetrics(claudeLogWithCache, false)

// Expected total: 1500 + 500 + 200 + 100 = 2300
expectedTotal := 2300
if goMetrics.TokenUsage != expectedTotal {
t.Errorf("Go parser: expected total tokens %d, got %d", expectedTotal, goMetrics.TokenUsage)
}

// Test with JavaScript parser
script := GetLogParserScript("parse_claude_log")
if script == "" {
t.Skip("parse_claude_log script not available")
}

jsResult, err := runJSLogParser(script, claudeLogWithCache)
if err != nil {
t.Fatalf("Failed to parse log with JavaScript parser: %v", err)
}

// Check that JavaScript output shows the total
if !strings.Contains(jsResult, "Total: 2,300") {
t.Errorf("JavaScript parser: expected to show 'Total: 2,300' in output, but didn't find it.\nOutput:\n%s", jsResult)
}

// Verify individual token counts are also shown
if !strings.Contains(jsResult, "Input: 1,500") {
t.Error("JavaScript parser: expected to show 'Input: 1,500' in output")
}
if !strings.Contains(jsResult, "Output: 500") {
t.Error("JavaScript parser: expected to show 'Output: 500' in output")
}
if !strings.Contains(jsResult, "Cache Creation: 200") {
t.Error("JavaScript parser: expected to show 'Cache Creation: 200' in output")
}
if !strings.Contains(jsResult, "Cache Read: 100") {
t.Error("JavaScript parser: expected to show 'Cache Read: 100' in output")
}
}

// TestTokenCountingWithoutCacheTokens verifies token counting works without cache tokens
func TestTokenCountingWithoutCacheTokens(t *testing.T) {
// Test log without cache tokens
claudeLogSimple := `[
  {
    "type": "system",
    "subtype": "init",
    "session_id": "test-simple",
    "tools": ["Bash"]
  },
  {
    "type": "assistant",
    "message": {
      "content": [
        {
          "type": "tool_use",
          "id": "tool_1",
          "name": "Bash",
          "input": {
            "command": "ls"
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
          "tool_use_id": "tool_1",
          "content": "file1.txt"
        }
      ]
    }
  },
  {
    "type": "result",
    "total_cost_usd": 0.001,
    "usage": {
      "input_tokens": 100,
      "output_tokens": 50
    },
    "num_turns": 1
  }
]`

// Test with Go parser
engine := NewClaudeEngine()
goMetrics := engine.ParseLogMetrics(claudeLogSimple, false)

// Expected total: 100 + 50 = 150
expectedTotal := 150
if goMetrics.TokenUsage != expectedTotal {
t.Errorf("Go parser: expected total tokens %d, got %d", expectedTotal, goMetrics.TokenUsage)
}

// Test with JavaScript parser
script := GetLogParserScript("parse_claude_log")
if script == "" {
t.Skip("parse_claude_log script not available")
}

jsResult, err := runJSLogParser(script, claudeLogSimple)
if err != nil {
t.Fatalf("Failed to parse log with JavaScript parser: %v", err)
}

// Check that JavaScript output shows the total
if !strings.Contains(jsResult, "Total: 150") {
t.Errorf("JavaScript parser: expected to show 'Total: 150' in output, but didn't find it.\nOutput:\n%s", jsResult)
}
}
