package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorSeverityHeading(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected string
	}{
		{
			name:     "critical severity",
			severity: SeverityCritical,
			expected: "CRITICAL (fix first)",
		},
		{
			name:     "high severity",
			severity: SeverityHigh,
			expected: "HIGH PRIORITY",
		},
		{
			name:     "medium severity",
			severity: SeverityMedium,
			expected: "MEDIUM PRIORITY",
		},
		{
			name:     "low severity",
			severity: SeverityLow,
			expected: "LOW PRIORITY",
		},
		{
			name:     "unknown severity falls back to default",
			severity: ErrorSeverity(999),
			expected: "PRIORITY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.Heading())
		})
	}
}

func TestErrorSeverityIcon(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected string
	}{
		{
			name:     "critical severity",
			severity: SeverityCritical,
			expected: "🔴",
		},
		{
			name:     "high severity",
			severity: SeverityHigh,
			expected: "🟠",
		},
		{
			name:     "medium severity",
			severity: SeverityMedium,
			expected: "🟡",
		},
		{
			name:     "low severity",
			severity: SeverityLow,
			expected: "🔵",
		},
		{
			name:     "unknown severity falls back to default",
			severity: ErrorSeverity(999),
			expected: "•",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.Icon())
		})
	}
}
