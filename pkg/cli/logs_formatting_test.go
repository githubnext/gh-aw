//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/console"
	"github.com/stretchr/testify/assert"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"single digit", 5, "5"},
		{"two digits", 42, "42"},
		{"three digits", 999, "999"},
		{"one thousand", 1000, "1.00k"},
		{"1.2k", 1200, "1.20k"},
		{"1.23k", 1234, "1.23k"},
		{"12k", 12000, "12.0k"},
		{"12.3k", 12300, "12.3k"},
		{"123k", 123000, "123k"},
		{"999.999k rounds to 1000k", 999999, "1000k"},
		{"one million", 1000000, "1.00M"},
		{"1.2M", 1200000, "1.20M"},
		{"1.23M", 1234567, "1.23M"},
		{"12M", 12000000, "12.0M"},
		{"12.3M", 12300000, "12.3M"},
		{"123M", 123000000, "123M"},
		{"999.999M rounds to 1000M", 999999999, "1000M"},
		{"one billion", 1000000000, "1.00B"},
		{"1.2B", 1200000000, "1.20B"},
		{"1.23B", 1234567890, "1.23B"},
		{"12B", 12000000000, "12.0B"},
		{"123B", 123000000000, "123B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := console.FormatNumber(tt.input)
			assert.Equal(t, tt.expected, result, "FormatNumber(%d)", tt.input)
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"100 bytes", 100, "100 B"},
		{"one kilobyte", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},                // 1.5 * 1024
		{"one megabyte", 1048576, "1.0 MB"},       // 1024 * 1024
		{"two megabytes", 2097152, "2.0 MB"},      // 2 * 1024 * 1024
		{"one gigabyte", 1073741824, "1.0 GB"},    // 1024^3
		{"one terabyte", 1099511627776, "1.0 TB"}, // 1024^4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := console.FormatFileSize(tt.size)
			assert.Equal(t, tt.expected, result, "FormatFileSize(%d)", tt.size)
		})
	}
}
