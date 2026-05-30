//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCronExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"midnight daily", "0 0 * * *", true},
		{"every 15 minutes", "*/15 * * * *", true},
		{"weekdays at 2pm", "0 14 * * 1-5", true},
		{"monday at 6:30am", "30 6 * * 1", true},
		{"christmas noon", "0 12 25 12 *", true},
		{"natural language daily", "daily", false},
		{"natural language weekly", "weekly on monday", false},
		{"natural language interval", "every 10 minutes", false},
		{"too few fields", "0 0 * *", false},
		{"too many fields", "0 0 * * * *", false},
		{"extra tokens", "0 0 * * * extra", false},
		{"invalid expression", "invalid cron expression", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCronExpression(tt.input)
			assert.Equal(t, tt.expected, result, "IsCronExpression(%q)", tt.input)
		})
	}
}

func TestIsDailyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"midnight", "0 0 * * *", true},
		{"2:30pm", "30 14 * * *", true},
		{"9am", "0 9 * * *", true},
		{"interval - not daily", "*/15 * * * *", false},
		{"monthly - not daily", "0 0 1 * *", false},
		{"weekly - not daily", "0 0 * * 1", false},
		{"weekdays only - not daily", "0 14 * * 1-5", false},
		{"invalid", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDailyCron(tt.input)
			assert.Equal(t, tt.expected, result, "IsDailyCron(%q)", tt.input)
		})
	}
}

func TestIsHourlyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"every hour at :00", "0 */1 * * *", true},
		{"every 2 hours at :30", "30 */2 * * *", true},
		{"every 6 hours at :15", "15 */6 * * *", true},
		{"daily - not hourly", "0 0 * * *", false},
		{"minute interval - not hourly", "*/30 * * * *", false},
		{"monthly - not hourly", "0 0 1 * *", false},
		{"invalid", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHourlyCron(tt.input)
			assert.Equal(t, tt.expected, result, "IsHourlyCron(%q)", tt.input)
		})
	}
}

func TestIsWeeklyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"monday midnight", "0 0 * * 1", true},
		{"friday 2:30pm", "30 14 * * 5", true},
		{"sunday 9am", "0 9 * * 0", true},
		{"saturday 5pm", "0 17 * * 6", true},
		{"interval - not weekly", "*/15 * * * *", false},
		{"monthly - not weekly", "0 0 1 * *", false},
		{"daily - not weekly", "0 0 * * *", false},
		{"weekday range - not simple weekly", "0 14 * * 1-5", false},
		{"DOW 7 - out of range", "0 0 * * 7", false},
		{"invalid", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWeeklyCron(tt.input)
			assert.Equal(t, tt.expected, result, "IsWeeklyCron(%q)", tt.input)
		})
	}
}

func TestIsFuzzyCron(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"fuzzy daily", "FUZZY:DAILY * * *", true},
		{"fuzzy hourly", "FUZZY:HOURLY * * *", true},
		{"fuzzy weekly", "FUZZY:WEEKLY * * *", true},
		{"real cron - not fuzzy", "0 0 * * *", false},
		{"natural language - not fuzzy", "daily", false},
		{"lowercase prefix - not fuzzy", "fuzzy:DAILY * * *", false},
		{"missing colon - not fuzzy", "FUZZYDAILY * * *", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFuzzyCron(tt.input)
			assert.Equal(t, tt.expected, result, "IsFuzzyCron(%q)", tt.input)
		})
	}
}
