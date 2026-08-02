//go:build !integration

package typeutil

import (
	"math"
	"testing"
)

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
