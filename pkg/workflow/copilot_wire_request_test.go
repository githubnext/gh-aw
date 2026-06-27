//go:build !integration

package workflow

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestSanitizeJSONBlock verifies that sanitizeJSONBlock correctly extracts the JSON object
// from strings that may contain leading/trailing non-JSON content.
func TestSanitizeJSONBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean JSON",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name: "trailing INFO line",
			input: `{"key": "value"}
2026-01-01T00:00:00Z [INFO] --- End of group ---`,
			want: `{"key": "value"}`,
		},
		{
			name: "trailing INFO line containing brace",
			input: `{"key": "value"}
2026-01-01T00:00:00Z [INFO] state: {done}`,
			want: `{"key": "value"}`,
		},
		{
			name:  "leading whitespace",
			input: `  {"key": "value"}  `,
			want:  `{"key": "value"}`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no braces",
			input: "just text",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeJSONBlock(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeJSONBlock(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestTruncateOutputSample verifies that output samples are capped to outputSampleMaxLines lines
// and outputSampleMaxLineLen characters per line.
func TestTruncateOutputSample(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "short output unchanged",
			input: "line1\nline2",
			want:  "line1\nline2",
		},
		{
			name:  "truncates to max lines",
			input: "line1\nline2\nline3\nline4\nline5",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "truncates long line with ellipsis",
			input: strings.Repeat("a", outputSampleMaxLineLen+50),
			want:  strings.Repeat("a", outputSampleMaxLineLen) + "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateOutputSample(tt.input)
			if result != tt.want {
				t.Errorf("truncateOutputSample(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestTruncateOutputSampleUTF8Safe(t *testing.T) {
	input := strings.Repeat("界", outputSampleMaxLineLen+10)
	got := truncateOutputSample(input)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8 output, got %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected UTF-8 safe truncation suffix, got %q", got)
	}
}

// TestCopilotParseWireRequestResponses tests that the Copilot debug log parser correctly
// handles the wireApi=responses format, which uses [DEBUG] Wire request: blocks to carry
// tool call outputs rather than "Executing tool:" lines.
//
// In this format:
//   - [DEBUG] data: blocks contain the LLM response with choices[].message.tool_calls
//   - [DEBUG] Wire request: blocks contain the full conversation history including
//     function_call_output items with the actual tool response content
//
// The key behaviours under test:
//  1. Tool calls are extracted from data blocks (choices[].message.tool_calls)
//  2. Tool output sizes and samples are extracted from Wire request blocks
//  3. [INFO] lines that appear after the closing } in data blocks do not break parsing
func TestCopilotParseWireRequestResponses(t *testing.T) {
	// Simulate two turns of the wireApi=responses format:
	//
	// Turn 1:
	//   - Wire request (initial, no prior outputs)
	//   - LLM response with tool_calls for "bash" (call_id: call_abc)
	//
	// Turn 2:
	//   - Wire request containing function_call_output for call_abc
	//   - LLM response with no more tool calls (finish_reason: stop)
	logContent := `2026-01-01T00:00:00.001Z [DEBUG] response (Request-ID req-001):
2026-01-01T00:00:00.001Z [DEBUG] data:
2026-01-01T00:00:00.001Z [DEBUG] {
  "choices": [
    {
      "message": {
        "content": "I will run the command.",
        "role": "assistant"
      },
      "index": 0,
      "finish_reason": "tool_calls"
    },
    {
      "message": {
        "content": null,
        "role": "assistant",
        "tool_calls": [
          {
            "id": "call_abc",
            "type": "function",
            "function": {
              "name": "bash",
              "arguments": "{\"command\":\"echo hello\"}"
            }
          }
        ]
      },
      "index": 1,
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 20,
    "total_tokens": 120
  }
}
2026-01-01T00:00:00.002Z [INFO] --- End of group ---
2026-01-01T00:00:00.003Z [DEBUG] Wire request: {
  "model": "gpt-5-mini",
  "input": [
    {
      "type": "function_call",
      "id": "fc_abc",
      "call_id": "call_abc",
      "name": "bash",
      "arguments": "{\"command\":\"echo hello\"}"
    },
    {
      "type": "function_call_output",
      "call_id": "call_abc",
      "output": "hello\nworld\nthird line\nfourth line"
    }
  ]
}
2026-01-01T00:00:00.004Z [DEBUG] response (Request-ID req-002):
2026-01-01T00:00:00.004Z [DEBUG] data:
2026-01-01T00:00:00.004Z [DEBUG] {
  "choices": [
    {
      "message": {
        "content": "Done.",
        "role": "assistant"
      },
      "index": 0,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 200,
    "completion_tokens": 10,
    "total_tokens": 210
  }
}
2026-01-01T00:00:00.005Z [DEBUG] Workflow completed`

	engine := NewCopilotEngine()
	metrics := engine.ParseLogMetrics(logContent, false)

	// Verify turns and token counts
	if metrics.Turns != 2 {
		t.Errorf("Expected 2 turns, got %d", metrics.Turns)
	}
	if metrics.TokenUsage != 330 { // 120 + 210
		t.Errorf("Expected 330 total tokens, got %d", metrics.TokenUsage)
	}

	// Verify tool calls were extracted
	if len(metrics.ToolCalls) == 0 {
		t.Fatal("Expected at least 1 tool call, got 0")
	}

	var bashInfo *ToolCallInfo
	for i := range metrics.ToolCalls {
		if metrics.ToolCalls[i].Name == "bash" {
			bashInfo = &metrics.ToolCalls[i]
			break
		}
	}
	if bashInfo == nil {
		t.Fatalf("Expected 'bash' tool call, got tools: %v", toolCallNames(metrics.ToolCalls))
	}

	// Verify input size
	if bashInfo.MaxInputSize != len(`{"command":"echo hello"}`) {
		t.Errorf("Expected MaxInputSize %d, got %d", len(`{"command":"echo hello"}`), bashInfo.MaxInputSize)
	}

	// Verify output size from Wire request block
	expectedOutputSize := len("hello\nworld\nthird line\nfourth line")
	if bashInfo.MaxOutputSize != expectedOutputSize {
		t.Errorf("Expected MaxOutputSize %d, got %d", expectedOutputSize, bashInfo.MaxOutputSize)
	}

	// Verify output sample (first outputSampleMaxLines lines)
	if bashInfo.OutputSample == "" {
		t.Error("Expected non-empty OutputSample")
	}
	sampleLines := strings.Split(bashInfo.OutputSample, "\n")
	if len(sampleLines) > outputSampleMaxLines {
		t.Errorf("OutputSample has %d lines, expected at most %d", len(sampleLines), outputSampleMaxLines)
	}
	if !strings.HasPrefix(bashInfo.OutputSample, "hello") {
		t.Errorf("OutputSample should start with 'hello', got %q", bashInfo.OutputSample)
	}
}

// TestCopilotInfoLinesDoNotBreakToolParsing tests that [INFO] log lines appearing between
// the JSON closing brace and the next [DEBUG] log line do not prevent tool call extraction.
// This is the wireApi=responses format where [INFO] "--- End of group ---" appears after
// each data block.
func TestCopilotInfoLinesDoNotBreakToolParsing(t *testing.T) {
	logContent := `2026-01-01T00:00:00.001Z [DEBUG] data:
2026-01-01T00:00:00.001Z [DEBUG] {
  "choices": [
    {
      "message": {
        "content": null,
        "role": "assistant",
        "tool_calls": [
          {
            "id": "call_xyz",
            "type": "function",
            "function": {
              "name": "grep",
              "arguments": "{\"pattern\":\"foo\"}"
            }
          }
        ]
      },
      "index": 0,
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 15,
    "total_tokens": 65
  }
}
2026-01-01T00:00:00.002Z [INFO] --- End of group ---
2026-01-01T00:00:00.003Z [DEBUG] Workflow completed`

	engine := NewCopilotEngine()
	metrics := engine.ParseLogMetrics(logContent, false)

	if len(metrics.ToolCalls) == 0 {
		t.Fatal("Expected tool calls to be extracted despite [INFO] line; got 0")
	}

	var grepInfo *ToolCallInfo
	for i := range metrics.ToolCalls {
		if metrics.ToolCalls[i].Name == "grep" {
			grepInfo = &metrics.ToolCalls[i]
			break
		}
	}
	if grepInfo == nil {
		t.Fatalf("Expected 'grep' tool call, got: %v", toolCallNames(metrics.ToolCalls))
	}

	if metrics.TokenUsage != 65 {
		t.Errorf("Expected 65 tokens, got %d", metrics.TokenUsage)
	}
}

func TestCopilotWireRequestBeforeDataBlock(t *testing.T) {
	logContent := `2026-01-01T00:00:00.001Z [DEBUG] Wire request: {
  "input": [
    {
      "type": "function_call",
      "id": "fc_abc",
      "call_id": "call_abc",
      "name": "bash",
      "arguments": "{\"command\":\"echo hello\"}"
    },
    {
      "type": "function_call_output",
      "call_id": "call_abc",
      "output": "hello [INFO] still output\n[DEBUG] still output"
    }
  ]
}
2026-01-01T00:00:00.002Z [DEBUG] data:
2026-01-01T00:00:00.002Z [DEBUG] {
  "choices": [
    {
      "message": {
        "role": "assistant",
        "tool_calls": [
          {
            "id": "call_abc",
            "type": "function",
            "function": {
              "name": "bash",
              "arguments": "{\"command\":\"echo hello\"}"
            }
          }
        ]
      },
      "index": 0,
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 5,
    "total_tokens": 15
  }
}
2026-01-01T00:00:00.003Z [DEBUG] Wire request: {
  "input": []
}
2026-01-01T00:00:00.004Z [DEBUG] Workflow completed`

	engine := NewCopilotEngine()
	metrics := engine.ParseLogMetrics(logContent, false)

	var bashInfo *ToolCallInfo
	for i := range metrics.ToolCalls {
		if metrics.ToolCalls[i].Name == "bash" {
			bashInfo = &metrics.ToolCalls[i]
			break
		}
	}
	if bashInfo == nil {
		t.Fatalf("Expected 'bash' tool call, got tools: %v", toolCallNames(metrics.ToolCalls))
	}
	if bashInfo.CallCount != 1 {
		t.Fatalf("expected CallCount 1, got %d", bashInfo.CallCount)
	}
	if bashInfo.MaxInputSize == 0 {
		t.Fatal("expected MaxInputSize to be populated from data block")
	}
	if bashInfo.MaxOutputSize == 0 {
		t.Fatal("expected MaxOutputSize to be populated from wire request block")
	}
	if bashInfo.OutputSample == "" {
		t.Fatal("expected OutputSample to be populated from wire request block")
	}
	if !strings.Contains(bashInfo.OutputSample, "[INFO] still output") {
		t.Fatalf("expected OutputSample to preserve embedded marker text, got %q", bashInfo.OutputSample)
	}
}

// toolCallNames returns a slice of tool names for error messages.
func toolCallNames(calls []ToolCallInfo) []string {
	names := make([]string, len(calls))
	for i, c := range calls {
		names[i] = c.Name
	}
	return names
}
