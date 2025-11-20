package workflow

import (
	"math"
	"testing"
)

// TestConvertToIntEdgeCases tests edge cases and boundary conditions for ConvertToInt
func TestConvertToIntEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		// Valid numeric conversions
		{"valid positive int", 123, 123},
		{"valid negative int", -42, -42},
		{"zero int", 0, 0},
		{"max int32", int(2147483647), 2147483647},
		{"min int32", int(-2147483648), -2147483648},
		{"int64 positive", int64(456), 456},
		{"int64 negative", int64(-789), -789},
		{"int64 zero", int64(0), 0},

		// Float to int truncation
		{"float64 clean conversion", 60.0, 60},
		{"float64 truncation positive", 60.7, 60},
		{"float64 truncation negative", -60.3, -60},
		{"float64 zero", 0.0, 0},
		{"float64 negative zero", -0.0, 0},
		{"float64 small positive", 0.9, 0},
		{"float64 small negative", -0.9, 0},
		{"float64 large positive", 999999.999, 999999},
		{"float64 large negative", -999999.999, -999999},

		// String conversions - valid
		{"valid string positive", "123", 123},
		{"valid string negative", "-42", -42},
		{"valid string zero", "0", 0},
		{"valid string with plus sign", "+123", 123},

		// String conversions - invalid (should return 0)
		{"invalid string letters", "abc", 0},
		{"invalid string mixed", "12abc", 0},
		{"invalid string with dots", "12.5.6", 0},
		{"empty string", "", 0},
		{"string with leading whitespace", " 123", 0}, // strconv.Atoi fails with spaces
		{"string with trailing whitespace", "123 ", 0},
		{"string with surrounding whitespace", " 123 ", 0},
		{"string with newline", "123\n", 0},
		{"string with tab", "123\t", 0},

		// Scientific notation strings - strconv.Atoi doesn't support these
		{"scientific notation lowercase", "1.5e3", 0},
		{"scientific notation uppercase", "1.5E3", 0},
		{"scientific notation negative exponent", "1.5e-3", 0},
		{"scientific notation positive", "1e10", 0},

		// Hexadecimal strings - strconv.Atoi doesn't support these
		{"hex string lowercase", "0xff", 0},
		{"hex string uppercase", "0xFF", 0},
		{"hex string without prefix", "FF", 0},

		// Special float values - behavior depends on conversion
		// Note: math.MaxFloat64 conversion to int is implementation-defined
		// We just test that it doesn't panic and returns some int value
		{"float64 positive infinity", math.Inf(1), int(math.Inf(1))},  // Converts but may overflow
		{"float64 negative infinity", math.Inf(-1), int(math.Inf(-1))}, // Converts but may overflow
		{"float64 NaN", math.NaN(), int(math.NaN())},                   // Converts to 0

		// Invalid types - should return 0
		{"nil", nil, 0},
		{"bool true", true, 0},
		{"bool false", false, 0},
		{"slice", []int{1, 2, 3}, 0},
		{"map", map[string]int{"a": 1}, 0},
		{"struct", struct{ x int }{x: 1}, 0},
		{"pointer to int", new(int), 0},

		// Additional edge cases
		{"uint32 value", uint32(100), 0}, // Not explicitly handled, returns 0
		{"uint64 value", uint64(200), 0}, // Not explicitly handled, returns 0
		{"float32 value", float32(50.5), 0}, // Not explicitly handled, returns 0
		{"int8 value", int8(10), 0},      // Not explicitly handled, returns 0
		{"int16 value", int16(20), 0},    // Not explicitly handled, returns 0
		{"int32 value", int32(30), 0},    // Not explicitly handled, returns 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToInt(%v) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConvertToIntNoPanic verifies that ConvertToInt never panics
func TestConvertToIntNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ConvertToInt panicked: %v", r)
		}
	}()

	// Test with various potentially problematic inputs
	inputs := []any{
		nil,
		true,
		false,
		"",
		"not a number",
		[]int{},
		map[string]int{},
		struct{}{},
		make(chan int),
		func() {},
		math.Inf(1),
		math.Inf(-1),
		math.NaN(),
	}

	for _, input := range inputs {
		_ = ConvertToInt(input)
	}
}

// TestParseIntValueEdgeCases tests edge cases and boundary conditions for parseIntValue
func TestParseIntValueEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expectedVal int
		expectedOk  bool
	}{
		// Valid numeric conversions
		{"valid int", 42, 42, true},
		{"negative int", -100, -100, true},
		{"zero int", 0, 0, true},
		{"max int32", int(2147483647), 2147483647, true},
		{"min int32", int(-2147483648), -2147483648, true},

		// Int64 conversions
		{"int64 positive", int64(1000), 1000, true},
		{"int64 negative", int64(-2000), -2000, true},
		{"int64 zero", int64(0), 0, true},
		{"int64 max", int64(9223372036854775807), int(9223372036854775807), true},

		// Uint64 conversions
		{"uint64 small", uint64(100), 100, true},
		{"uint64 zero", uint64(0), 0, true},
		{"uint64 large", uint64(1000000), 1000000, true},

		// Float64 truncation
		{"float64 clean", 50.0, 50, true},
		{"float64 truncation positive", 99.9, 99, true},
		{"float64 truncation negative", -99.9, -99, true},
		{"float64 zero", 0.0, 0, true},
		{"float64 small positive", 0.5, 0, true},
		{"float64 small negative", -0.5, 0, true},

		// Special float values
		// Note: Special float values behavior is implementation-defined
		{"float64 infinity positive", math.Inf(1), int(math.Inf(1)), true},
		{"float64 infinity negative", math.Inf(-1), int(math.Inf(-1)), true},
		{"float64 NaN", math.NaN(), int(math.NaN()), true}, // Converts but result is unpredictable

		// Invalid types - should return (0, false)
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
		{"bool true", true, 0, false},
		{"bool false", false, 0, false},
		{"slice", []int{1, 2}, 0, false},
		{"map", map[string]int{"x": 1}, 0, false},
		{"struct", struct{ x int }{x: 1}, 0, false},
		{"pointer", new(int), 0, false},

		// Additional numeric types not explicitly handled
		{"uint32", uint32(100), 0, false},
		{"uint", uint(200), 0, false},
		{"int8", int8(10), 0, false},
		{"int16", int16(20), 0, false},
		{"int32", int32(30), 0, false},
		{"float32", float32(40.5), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := parseIntValue(tt.input)
			if ok != tt.expectedOk {
				t.Errorf("parseIntValue(%v) ok = %v; want %v", tt.input, ok, tt.expectedOk)
			}
			if val != tt.expectedVal {
				t.Errorf("parseIntValue(%v) value = %d; want %d", tt.input, val, tt.expectedVal)
			}
		})
	}
}

// TestParseIntValueNoPanic verifies that parseIntValue never panics
func TestParseIntValueNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("parseIntValue panicked: %v", r)
		}
	}()

	// Test with various potentially problematic inputs
	inputs := []any{
		nil,
		true,
		"string",
		[]int{},
		map[string]int{},
		make(chan int),
		func() {},
		math.Inf(1),
		math.NaN(),
	}

	for _, input := range inputs {
		_, _ = parseIntValue(input)
	}
}

// TestConvertToFloatEdgeCases tests edge cases for ConvertToFloat
func TestConvertToFloatEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		// Valid conversions
		{"float64 exact", 123.45, 123.45},
		{"int to float", 100, 100.0},
		{"int64 to float", int64(200), 200.0},
		{"negative float", -50.75, -50.75},
		{"negative int", -25, -25.0},
		{"zero float", 0.0, 0.0},
		{"zero int", 0, 0.0},

		// String conversions
		{"valid float string", "99.99", 99.99},
		{"valid int string", "50", 50.0},
		{"scientific notation string", "1.5e2", 150.0}, // ParseFloat supports this
		{"negative float string", "-25.5", -25.5},

		// Invalid strings
		{"invalid string", "not a number", 0.0},
		{"empty string", "", 0.0},
		{"string with spaces", " 123.45 ", 0.0}, // ParseFloat fails with spaces

		// Special float values
		{"positive infinity", math.Inf(1), math.Inf(1)},
		{"negative infinity", math.Inf(-1), math.Inf(-1)},
		{"max float64", math.MaxFloat64, math.MaxFloat64},
		{"smallest positive float64", math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64},

		// Invalid types
		{"nil", nil, 0.0},
		{"bool true", true, 0.0},
		{"bool false", false, 0.0},
		{"slice", []float64{1.0, 2.0}, 0.0},
		{"map", map[string]float64{"x": 1.0}, 0.0},

		// Additional numeric types
		{"uint32", uint32(100), 0.0},
		{"uint64", uint64(200), 0.0},
		{"int32", int32(30), 0.0},
		{"float32", float32(40.5), 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToFloat(tt.input)

			// Special handling for NaN comparison
			if math.IsNaN(tt.expected) {
				if !math.IsNaN(result) {
					t.Errorf("ConvertToFloat(%v) = %f; want NaN", tt.input, result)
				}
			} else if math.IsInf(tt.expected, 1) {
				if !math.IsInf(result, 1) {
					t.Errorf("ConvertToFloat(%v) = %f; want +Inf", tt.input, result)
				}
			} else if math.IsInf(tt.expected, -1) {
				if !math.IsInf(result, -1) {
					t.Errorf("ConvertToFloat(%v) = %f; want -Inf", tt.input, result)
				}
			} else if result != tt.expected {
				t.Errorf("ConvertToFloat(%v) = %f; want %f", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConvertToFloatNoPanic verifies that ConvertToFloat never panics
func TestConvertToFloatNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ConvertToFloat panicked: %v", r)
		}
	}()

	// Test with various potentially problematic inputs
	inputs := []any{
		nil,
		true,
		"not a number",
		[]float64{},
		map[string]float64{},
		make(chan float64),
		func() {},
		math.Inf(1),
		math.NaN(),
	}

	for _, input := range inputs {
		_ = ConvertToFloat(input)
	}
}

// TestTypeConversionConsistency verifies consistent behavior between similar inputs
func TestTypeConversionConsistency(t *testing.T) {
	t.Run("zero values return zero", func(t *testing.T) {
		if ConvertToInt(0) != 0 {
			t.Error("ConvertToInt(0) should return 0")
		}
		if ConvertToInt(int64(0)) != 0 {
			t.Error("ConvertToInt(int64(0)) should return 0")
		}
		if ConvertToInt(0.0) != 0 {
			t.Error("ConvertToInt(0.0) should return 0")
		}
		if ConvertToInt("0") != 0 {
			t.Error("ConvertToInt(\"0\") should return 0")
		}
	})

	t.Run("negative numbers preserve sign", func(t *testing.T) {
		if ConvertToInt(-42) != -42 {
			t.Error("ConvertToInt(-42) should return -42")
		}
		if ConvertToInt(int64(-100)) != -100 {
			t.Error("ConvertToInt(int64(-100)) should return -100")
		}
		if ConvertToInt(-50.5) != -50 {
			t.Error("ConvertToInt(-50.5) should return -50 (truncated)")
		}
		if ConvertToInt("-75") != -75 {
			t.Error("ConvertToInt(\"-75\") should return -75")
		}
	})

	t.Run("invalid inputs return zero", func(t *testing.T) {
		if ConvertToInt(nil) != 0 {
			t.Error("ConvertToInt(nil) should return 0")
		}
		if ConvertToInt(true) != 0 {
			t.Error("ConvertToInt(true) should return 0")
		}
		if ConvertToInt("abc") != 0 {
			t.Error("ConvertToInt(\"abc\") should return 0")
		}
		if ConvertToInt([]int{}) != 0 {
			t.Error("ConvertToInt([]int{}) should return 0")
		}
	})
}

// TestFloatTruncationBehavior documents and tests the truncation behavior
func TestFloatTruncationBehavior(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected int
		note     string
	}{
		{"positive truncation", 60.7, 60, "Positive decimals are truncated toward zero"},
		{"negative truncation", -60.7, -60, "Negative decimals are truncated toward zero"},
		{"exact conversion", 100.0, 100, "Exact floats convert cleanly"},
		{"small positive", 0.9, 0, "Small positive values truncate to 0"},
		{"small negative", -0.9, 0, "Small negative values truncate to 0"},
		{"large positive", 123456.789, 123456, "Large positive decimals truncate"},
		{"large negative", -123456.789, -123456, "Large negative decimals truncate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToInt(%f) = %d; want %d (%s)", tt.input, result, tt.expected, tt.note)
			}
		})
	}
}
