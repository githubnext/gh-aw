//go:build !integration

package scanfindings_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/scanfindings"
)

// TestSpec_PublicAPI_ParseSeverity validates the documented behavior of ParseSeverity.
func TestSpec_PublicAPI_ParseSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected scanfindings.SeverityLevel
	}{
		{name: "critical aliases normalize", input: "crit", expected: scanfindings.SeverityCritical},
		{name: "high is case insensitive", input: "High", expected: scanfindings.SeverityHigh},
		{name: "warning maps to medium", input: "warning", expected: scanfindings.SeverityMedium},
		{name: "minor maps to low", input: "minor", expected: scanfindings.SeverityLow},
		{name: "notice maps to info", input: "notice", expected: scanfindings.SeverityInfo},
		{name: "unknown label returns unknown", input: "mystery", expected: scanfindings.SeverityUnknown},
		{name: "empty string returns unknown", input: "", expected: scanfindings.SeverityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanfindings.ParseSeverity(tt.input)
			assert.Equal(t, tt.expected, result, "result mismatch for: %s", tt.name)
		})
	}
}

// TestSpec_Types_SeverityLevel validates the documented severity vocabulary and ordering.
func TestSpec_Types_SeverityLevel(t *testing.T) {
	assert.Equal(t, "unknown", scanfindings.SeverityUnknown.String(), "unknown severity should have canonical lowercase name")
	assert.Equal(t, "info", scanfindings.SeverityInfo.String(), "info severity should have canonical lowercase name")
	assert.Equal(t, "low", scanfindings.SeverityLow.String(), "low severity should have canonical lowercase name")
	assert.Equal(t, "medium", scanfindings.SeverityMedium.String(), "medium severity should have canonical lowercase name")
	assert.Equal(t, "high", scanfindings.SeverityHigh.String(), "high severity should have canonical lowercase name")
	assert.Equal(t, "critical", scanfindings.SeverityCritical.String(), "critical severity should have canonical lowercase name")

	assert.Equal(t, 0, scanfindings.SeverityUnknown.Rank(), "unknown should rank lowest")
	assert.Equal(t, 1, scanfindings.SeverityInfo.Rank(), "info rank mismatch")
	assert.Equal(t, 2, scanfindings.SeverityLow.Rank(), "low rank mismatch")
	assert.Equal(t, 3, scanfindings.SeverityMedium.Rank(), "medium rank mismatch")
	assert.Equal(t, 4, scanfindings.SeverityHigh.Rank(), "high rank mismatch")
	assert.Equal(t, 5, scanfindings.SeverityCritical.Rank(), "critical rank mismatch")

	assert.True(t, scanfindings.SeverityCritical.AtLeast(scanfindings.SeverityHigh), "critical should satisfy a high threshold")
	assert.True(t, scanfindings.SeverityMedium.AtLeast(scanfindings.SeverityMedium), "a severity should satisfy its own threshold")
	assert.False(t, scanfindings.SeverityInfo.AtLeast(scanfindings.SeverityLow), "info should not satisfy a low threshold")

	assert.Equal(t, "error", scanfindings.SeverityCritical.ErrorType(), "critical findings should render as console errors")
	assert.Equal(t, "error", scanfindings.SeverityHigh.ErrorType(), "high findings should render as console errors")
	assert.Equal(t, "warning", scanfindings.SeverityMedium.ErrorType(), "medium findings should render as warnings")
	assert.Equal(t, "info", scanfindings.SeverityLow.ErrorType(), "low findings should render as info")
	assert.Equal(t, "info", scanfindings.SeverityInfo.ErrorType(), "info findings should render as info")
	assert.Equal(t, "warning", scanfindings.SeverityUnknown.ErrorType(), "unknown findings should remain visible as warnings")
}

// TestSpec_PublicAPI_FormatMessage validates the documented message format.
func TestSpec_PublicAPI_FormatMessage(t *testing.T) {
	tests := []struct {
		name        string
		severity    string
		ruleID      string
		description string
		expected    string
	}{
		{
			name:        "full message uses documented bracketed severity format",
			severity:    "High",
			ruleID:      "template-injection",
			description: "template injection with untrusted input",
			expected:    "[High] template-injection: template injection with untrusted input",
		},
		{name: "omits empty severity", ruleID: "rule", description: "desc", expected: "rule: desc"},
		{name: "omits empty description", severity: "info", ruleID: "rule", expected: "[info] rule"},
		{name: "omits empty rule id", severity: "info", description: "desc", expected: "[info] desc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanfindings.FormatMessage(tt.severity, tt.ruleID, tt.description)
			assert.Equal(t, tt.expected, result, "result mismatch for: %s", tt.name)
		})
	}
}

// TestSpec_Types_Finding validates the documented Finding fields and conversion behavior.
func TestSpec_Types_Finding(t *testing.T) {
	finding := scanfindings.Finding{
		RuleID:   "template-injection",
		Severity: scanfindings.SeverityHigh,
		Message:  scanfindings.FormatMessage("High", "template-injection", "template injection with untrusted input"),
		File:     ".github/workflows/demo.lock.yml",
		Line:     12,
		Column:   24,
		Context:  []string{"line 10", "line 11", "line 12"},
	}

	compilerError := finding.CompilerError()
	assert.Equal(t, finding.File, compilerError.Position.File, "file should be preserved in compiler error conversion")
	assert.Equal(t, finding.Line, compilerError.Position.Line, "line should be preserved in compiler error conversion")
	assert.Equal(t, finding.Column, compilerError.Position.Column, "column should be preserved in compiler error conversion")
	assert.Equal(t, "error", compilerError.Type, "high severity should convert to console error type")
	assert.Equal(t, finding.Message, compilerError.Message, "message should be preserved in compiler error conversion")
	assert.Equal(t, finding.Context, compilerError.Context, "context should be preserved in compiler error conversion")
}

// TestSpec_PublicAPI_ContextLines validates the documented surrounding-line window behavior.
func TestSpec_PublicAPI_ContextLines(t *testing.T) {
	fileLines := []string{"one", "two", "three", "four", "five", "six"}

	assert.Equal(t, []string{"one", "two", "three"}, scanfindings.ContextLines(fileLines, 2), "context should shrink near the start of the file")
	assert.Equal(t, []string{"two", "three", "four", "five", "six"}, scanfindings.ContextLines(fileLines, 4), "context should include up to two lines before and after the target line")
	assert.Nil(t, scanfindings.ContextLines(fileLines, 0), "out-of-range line numbers should return nil")
}

// TestSpec_PublicAPI_Render validates the documented shared console rendering entrypoint.
func TestSpec_PublicAPI_Render(t *testing.T) {
	findings := []scanfindings.Finding{{
		RuleID:   "template-injection",
		Severity: scanfindings.ParseSeverity("High"),
		Message:  scanfindings.FormatMessage("High", "template-injection", "template injection with untrusted input"),
		File:     ".github/workflows/demo.lock.yml",
		Line:     12,
		Column:   24,
	}}

	var buf bytes.Buffer
	scanfindings.Render(&buf, findings)

	output := buf.String()
	assert.Contains(t, output, findings[0].Message, "rendered output should contain the finding message")
	assert.Contains(t, output, findings[0].File, "rendered output should contain the finding file")
	assert.Contains(t, output, ":12:24", "rendered output should contain the finding position")
}

// SPEC_MISMATCH: README.md documents exported Sort and CountAtLeast functions,
// but those exported functions are not currently present in the package source.
