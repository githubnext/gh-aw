//go:build !integration

package typeutil

import (
	"math"
	"reflect"
	"testing"
)

func TestPtr(t *testing.T) {
	b := Ptr(true)
	if b == nil || !*b {
		t.Fatalf("Ptr(true) = %v, want pointer to true", b)
	}

	s := Ptr("hello")
	if s == nil || *s != "hello" {
		t.Fatalf("Ptr(\"hello\") = %v, want pointer to hello", s)
	}
}

func TestParseIntValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected int
		ok       bool
	}{
		{"int value", 42, 42, true},
		{"int64 value", int64(100), 100, true},
		{"uint64 value", uint64(200), 200, true},
		{"float64 value", float64(3.14), 3, true},
		{"string value (not supported)", "42", 0, false},
		{"nil value", nil, 0, false},
		{"bool value (not supported)", true, 0, false},
		{"uint64 overflow returns 0", ^uint64(0), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := ParseIntValue(tt.value)
			if ok != tt.ok {
				t.Errorf("ParseIntValue(%v) ok = %v, want %v for test case %q", tt.value, ok, tt.ok, tt.name)
			}
			if result != tt.expected {
				t.Errorf("ParseIntValue(%v) result = %v, want %v for test case %q", tt.value, result, tt.expected, tt.name)
			}
		})
	}
}

func TestSafeUint64ToInt(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		expected int
	}{
		{"zero", 0, 0},
		{"small value", 42, 42},
		{"large value within int range", 1000000, 1000000},
		{"max int value", uint64(^uint(0) >> 1), int(^uint(0) >> 1)},
		{"overflow: max uint64 returns 0", ^uint64(0), 0},
		{"overflow: max int + 1 returns 0", uint64(^uint(0)>>1) + 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeUint64ToInt(tt.value)
			if result != tt.expected {
				t.Errorf("SafeUint64ToInt(%d) = %d, want %d", tt.value, result, tt.expected)
			}
		})
	}
}

func TestSafeUintToInt(t *testing.T) {
	tests := []struct {
		name     string
		value    uint
		expected int
	}{
		{"zero", 0, 0},
		{"small value", 100, 100},
		{"large value within range", uint(math.MaxInt32), math.MaxInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeUintToInt(tt.value)
			if result != tt.expected {
				t.Errorf("SafeUintToInt(%d) = %d, want %d", tt.value, result, tt.expected)
			}
		})
	}
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"int", 42, 42},
		{"int64", int64(100), 100},
		{"float64 clean", 60.0, 60},
		{"float64 truncation", 60.7, 60},
		{"string number", "123", 123},
		{"invalid string", "abc", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToInt(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"float64", 123.45, 123.45},
		{"int", 100, 100.0},
		{"int64", int64(200), 200.0},
		{"string", "99.99", 99.99},
		{"invalid string", "not a number", 0.0},
		{"nil", nil, 0.0},
		{"bool", true, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToFloat(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToFloat(%v) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLookupMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected map[string]any
		ok       bool
	}{
		{
			name: "existing map value",
			input: map[string]any{
				"tool": map[string]any{"name": "Bash"},
			},
			key:      "tool",
			expected: map[string]any{"name": "Bash"},
			ok:       true,
		},
		{
			name:     "missing key",
			input:    map[string]any{},
			key:      "tool",
			expected: nil,
			ok:       false,
		},
		{
			name: "wrong type",
			input: map[string]any{
				"tool": "not-a-map",
			},
			key:      "tool",
			expected: nil,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := LookupMap(tt.input, tt.key)
			if ok != tt.ok {
				t.Errorf("LookupMap(%v, %q) ok = %v, want %v", tt.input, tt.key, ok, tt.ok)
			}
			if ok && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("LookupMap(%v, %q) = %v, want %v", tt.input, tt.key, result, tt.expected)
			}
		})
	}
}

func TestLookupString(t *testing.T) {
	nested := map[string]any{
		"type": "tool_use",
		"input": map[string]any{
			"command": "echo hello",
		},
	}

	if value, ok := LookupString(nested, "type"); !ok || value != "tool_use" {
		t.Errorf("LookupString(type) = (%q, %v), want (%q, true)", value, ok, "tool_use")
	}

	if _, ok := LookupString(nested, "missing"); ok {
		t.Error("LookupString should return ok=false for missing key")
	}
}

func TestLookupStringPath(t *testing.T) {
	nested := map[string]any{
		"type": "tool_use",
		"input": map[string]any{
			"command": "echo hello",
		},
	}

	if value, ok := LookupStringPath(nested, "input", "command"); !ok || value != "echo hello" {
		t.Errorf("LookupStringPath(input, command) = (%q, %v), want (%q, true)", value, ok, "echo hello")
	}

	if _, ok := LookupStringPath(nested, "input", "missing"); ok {
		t.Error("LookupStringPath should return ok=false for missing path segment")
	}

	if _, ok := LookupStringPath(nested); ok {
		t.Error("LookupStringPath should return ok=false for empty path")
	}
}
