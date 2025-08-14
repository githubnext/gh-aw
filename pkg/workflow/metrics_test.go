package workflow

import (
	"encoding/json"
	"testing"
)

func TestExtractFirstMatch(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pattern  string
		expected string
	}{
		{
			name:     "Basic match",
			text:     "Token count: 1500 tokens",
			pattern:  `Token count: (\d+)`,
			expected: "1500",
		},
		{
			name:     "No match",
			text:     "No tokens here",
			pattern:  `Token count: (\d+)`,
			expected: "",
		},
		{
			name:     "Case insensitive match",
			text:     "TOKEN COUNT: 2000 tokens",
			pattern:  `token count: (\d+)`,
			expected: "2000",
		},
		{
			name:     "Multiple matches - first one returned",
			text:     "Token count: 1000 tokens, Cost: 0.05",
			pattern:  `(\d+)`,
			expected: "1000",
		},
		{
			name:     "Empty text",
			text:     "",
			pattern:  `Token count: (\d+)`,
			expected: "",
		},
		{
			name:     "Empty pattern",
			text:     "Token count: 1500 tokens",
			pattern:  ``,
			expected: "",
		},
		{
			name:     "Complex pattern with named groups",
			text:     "Usage: input=500, output=300",
			pattern:  `input=(\d+)`,
			expected: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFirstMatch(tt.text, tt.pattern)
			if result != tt.expected {
				t.Errorf("ExtractFirstMatch(%q, %q) = %q, want %q", tt.text, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestExtractJSONMetrics(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		verbose  bool
		expected LogMetrics
	}{
		{
			name: "Claude API format with tokens",
			line: `{"usage": {"input_tokens": 100, "output_tokens": 50}}`,
			expected: LogMetrics{
				TokenUsage: 150,
			},
		},
		{
			name: "Claude API format with cache tokens",
			line: `{"usage": {"input_tokens": 100, "output_tokens": 50, "cache_creation_input_tokens": 200, "cache_read_input_tokens": 75}}`,
			expected: LogMetrics{
				TokenUsage: 425, // 100 + 50 + 200 + 75
			},
		},
		{
			name: "Simple token count",
			line: `{"tokens": 250}`,
			expected: LogMetrics{
				TokenUsage: 250,
			},
		},
		{
			name: "Cost information",
			line: `{"cost": 0.05, "tokens": 1000}`,
			expected: LogMetrics{
				TokenUsage:    1000,
				EstimatedCost: 0.05,
			},
		},
		{
			name: "Delta streaming format",
			line: `{"delta": {"usage": {"input_tokens": 10, "output_tokens": 15}}}`,
			expected: LogMetrics{
				TokenUsage: 25,
			},
		},
		{
			name: "Billing information",
			line: `{"billing": {"total_cost_usd": 0.12}, "tokens": 500}`,
			expected: LogMetrics{
				TokenUsage:    500,
				EstimatedCost: 0.12,
			},
		},
		{
			name:     "Non-JSON line",
			line:     "This is not JSON",
			expected: LogMetrics{},
		},
		{
			name:     "Empty JSON object",
			line:     "{}",
			expected: LogMetrics{},
		},
		{
			name:     "Malformed JSON",
			line:     `{"invalid": json}`,
			expected: LogMetrics{},
		},
		{
			name:     "Empty line",
			line:     "",
			expected: LogMetrics{},
		},
		{
			name: "Total tokens field",
			line: `{"total_tokens": 750}`,
			expected: LogMetrics{
				TokenUsage: 750,
			},
		},
		{
			name: "Mixed token fields - should use first found",
			line: `{"input_tokens": 200, "total_tokens": 500}`,
			expected: LogMetrics{
				TokenUsage: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJSONMetrics(tt.line, tt.verbose)
			if result.TokenUsage != tt.expected.TokenUsage {
				t.Errorf("ExtractJSONMetrics(%q, %t).TokenUsage = %d, want %d",
					tt.line, tt.verbose, result.TokenUsage, tt.expected.TokenUsage)
			}
			if result.EstimatedCost != tt.expected.EstimatedCost {
				t.Errorf("ExtractJSONMetrics(%q, %t).EstimatedCost = %f, want %f",
					tt.line, tt.verbose, result.EstimatedCost, tt.expected.EstimatedCost)
			}
		})
	}
}

func TestExtractJSONTokenUsage(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected int
	}{
		{
			name: "Direct tokens field",
			data: map[string]interface{}{
				"tokens": 500,
			},
			expected: 500,
		},
		{
			name: "Token count field",
			data: map[string]interface{}{
				"token_count": 300,
			},
			expected: 300,
		},
		{
			name: "Usage object with input/output tokens",
			data: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
			expected: 150,
		},
		{
			name: "Usage object with cache tokens",
			data: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens":                100,
					"output_tokens":               50,
					"cache_creation_input_tokens": 200,
					"cache_read_input_tokens":     75,
				},
			},
			expected: 425,
		},
		{
			name: "Delta format",
			data: map[string]interface{}{
				"delta": map[string]interface{}{
					"usage": map[string]interface{}{
						"input_tokens":  25,
						"output_tokens": 35,
					},
				},
			},
			expected: 60,
		},
		{
			name: "String token count",
			data: map[string]interface{}{
				"tokens": "750",
			},
			expected: 750,
		},
		{
			name: "Float token count",
			data: map[string]interface{}{
				"tokens": 123.45,
			},
			expected: 123,
		},
		{
			name: "No token information",
			data: map[string]interface{}{
				"message": "hello",
			},
			expected: 0,
		},
		{
			name: "Invalid usage object",
			data: map[string]interface{}{
				"usage": "not an object",
			},
			expected: 0,
		},
		{
			name: "Partial usage information",
			data: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens": 100,
					// No output_tokens
				},
			},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJSONTokenUsage(tt.data)
			if result != tt.expected {
				t.Errorf("ExtractJSONTokenUsage(%+v) = %d, want %d", tt.data, result, tt.expected)
			}
		})
	}
}

func TestExtractJSONCost(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected float64
	}{
		{
			name: "Direct cost field",
			data: map[string]interface{}{
				"cost": 0.05,
			},
			expected: 0.05,
		},
		{
			name: "Price field",
			data: map[string]interface{}{
				"price": 1.25,
			},
			expected: 1.25,
		},
		{
			name: "Total cost USD",
			data: map[string]interface{}{
				"total_cost_usd": 0.125,
			},
			expected: 0.125,
		},
		{
			name: "Billing object",
			data: map[string]interface{}{
				"billing": map[string]interface{}{
					"total_cost": 0.75,
				},
			},
			expected: 0.75,
		},
		{
			name: "String cost value",
			data: map[string]interface{}{
				"cost": "0.25",
			},
			expected: 0.25,
		},
		{
			name: "Integer cost value",
			data: map[string]interface{}{
				"cost": 2,
			},
			expected: 2.0,
		},
		{
			name: "No cost information",
			data: map[string]interface{}{
				"message": "hello",
			},
			expected: 0.0,
		},
		{
			name: "Invalid billing object",
			data: map[string]interface{}{
				"billing": "not an object",
			},
			expected: 0.0,
		},
		{
			name: "Zero cost",
			data: map[string]interface{}{
				"cost": 0.0,
			},
			expected: 0.0,
		},
		{
			name: "Negative cost (should be ignored)",
			data: map[string]interface{}{
				"cost": -1.0,
			},
			expected: 0.0,
		},
		{
			name: "Multiple cost fields - first found wins",
			data: map[string]interface{}{
				"cost":  0.10,
				"price": 0.20,
			},
			expected: 0.10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJSONCost(tt.data)
			if result != tt.expected {
				t.Errorf("ExtractJSONCost(%+v) = %f, want %f", tt.data, result, tt.expected)
			}
		})
	}
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name     string
		val      interface{}
		expected int
	}{
		{
			name:     "Integer value",
			val:      123,
			expected: 123,
		},
		{
			name:     "Int64 value",
			val:      int64(456),
			expected: 456,
		},
		{
			name:     "Float64 value",
			val:      789.0,
			expected: 789,
		},
		{
			name:     "Float64 with decimals",
			val:      123.99,
			expected: 123,
		},
		{
			name:     "String integer",
			val:      "555",
			expected: 555,
		},
		{
			name:     "String with whitespace",
			val:      " 777 ",
			expected: 0, // strconv.Atoi will fail with spaces
		},
		{
			name:     "Invalid string",
			val:      "not a number",
			expected: 0,
		},
		{
			name:     "Boolean value",
			val:      true,
			expected: 0,
		},
		{
			name:     "Nil value",
			val:      nil,
			expected: 0,
		},
		{
			name:     "Array value",
			val:      []int{1, 2, 3},
			expected: 0,
		},
		{
			name:     "Zero values",
			val:      0,
			expected: 0,
		},
		{
			name:     "Negative integer",
			val:      -100,
			expected: -100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt(tt.val)
			if result != tt.expected {
				t.Errorf("ConvertToInt(%v) = %d, want %d", tt.val, result, tt.expected)
			}
		})
	}
}

func TestConvertToFloat(t *testing.T) {
	tests := []struct {
		name     string
		val      interface{}
		expected float64
	}{
		{
			name:     "Float64 value",
			val:      123.45,
			expected: 123.45,
		},
		{
			name:     "Integer value",
			val:      100,
			expected: 100.0,
		},
		{
			name:     "Int64 value",
			val:      int64(200),
			expected: 200.0,
		},
		{
			name:     "String float",
			val:      "99.99",
			expected: 99.99,
		},
		{
			name:     "String integer",
			val:      "50",
			expected: 50.0,
		},
		{
			name:     "Invalid string",
			val:      "not a number",
			expected: 0.0,
		},
		{
			name:     "Boolean value",
			val:      false,
			expected: 0.0,
		},
		{
			name:     "Nil value",
			val:      nil,
			expected: 0.0,
		},
		{
			name:     "Zero float",
			val:      0.0,
			expected: 0.0,
		},
		{
			name:     "Negative float",
			val:      -25.5,
			expected: -25.5,
		},
		{
			name:     "Scientific notation string",
			val:      "1.5e2",
			expected: 150.0,
		},
		{
			name:     "Map value",
			val:      map[string]int{"key": 1},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToFloat(tt.val)
			if result != tt.expected {
				t.Errorf("ConvertToFloat(%v) = %f, want %f", tt.val, result, tt.expected)
			}
		})
	}
}

// TestExtractJSONMetricsIntegration tests the integration between different metric extraction functions
func TestExtractJSONMetricsIntegration(t *testing.T) {
	// Test with realistic Claude API response
	claudeResponse := map[string]interface{}{
		"id":   "msg_01ABC123",
		"type": "message",
		"role": "assistant",
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "Hello, world!",
			},
		},
		"model":         "claude-3-5-sonnet-20241022",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":  25,
			"output_tokens": 5,
		},
	}

	jsonBytes, err := json.Marshal(claudeResponse)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	metrics := ExtractJSONMetrics(string(jsonBytes), false)

	if metrics.TokenUsage != 30 {
		t.Errorf("Expected token usage 30, got %d", metrics.TokenUsage)
	}

	if metrics.EstimatedCost != 0.0 {
		t.Errorf("Expected no cost information, got %f", metrics.EstimatedCost)
	}
}
