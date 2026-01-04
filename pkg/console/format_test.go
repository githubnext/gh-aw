package console

import "testing"

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},          // 1.5 * 1024
		{1048576, "1.0 MB"},       // 1024 * 1024
		{2097152, "2.0 MB"},       // 2 * 1024 * 1024
		{1073741824, "1.0 GB"},    // 1024^3
		{1099511627776, "1.0 TB"}, // 1024^4
	}

	for _, tt := range tests {
		result := FormatFileSize(tt.size)
		if result != tt.expected {
			t.Errorf("FormatFileSize(%d) = %q, expected %q", tt.size, result, tt.expected)
		}
	}
}

func TestFormatNumberOrEmpty(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{1, "1"},
		{999, "999"},
		{1000, "1.00k"},
		{1500, "1.50k"},
		{1000000, "1.00M"},
	}

	for _, tt := range tests {
		result := FormatNumberOrEmpty(tt.n)
		if result != tt.expected {
			t.Errorf("FormatNumberOrEmpty(%d) = %q, expected %q", tt.n, result, tt.expected)
		}
	}
}

func TestFormatCostOrEmpty(t *testing.T) {
	tests := []struct {
		cost     float64
		expected string
	}{
		{0, ""},
		{0.001, "0.001"},
		{0.123, "0.123"},
		{1.234, "1.234"},
		{10.5, "10.500"},
	}

	for _, tt := range tests {
		result := FormatCostOrEmpty(tt.cost)
		if result != tt.expected {
			t.Errorf("FormatCostOrEmpty(%f) = %q, expected %q", tt.cost, result, tt.expected)
		}
	}
}

func TestFormatIntOrEmpty(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{1, "1"},
		{42, "42"},
		{1000, "1000"},
	}

	for _, tt := range tests {
		result := FormatIntOrEmpty(tt.n)
		if result != tt.expected {
			t.Errorf("FormatIntOrEmpty(%d) = %q, expected %q", tt.n, result, tt.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s        string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exact length", 12, "exact length"},
		{"this is a long string", 10, "this is..."},
		{"truncate", 5, "tr..."},
		{"abc", 2, "ab"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := TruncateString(tt.s, tt.maxLen)
		if result != tt.expected {
			t.Errorf("TruncateString(%q, %d) = %q, expected %q", tt.s, tt.maxLen, result, tt.expected)
		}
	}
}
