package console

import (
	"strings"
	"testing"
)

func TestFormatValidationSummary_NoErrors(t *testing.T) {
	results := &ValidationResults{
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	output := FormatValidationSummary(results, false)
	if output != "" {
		t.Errorf("Expected empty output for no errors, got: %s", output)
	}
}

func TestFormatValidationSummary_SingleError(t *testing.T) {
	results := &ValidationResults{
		Errors: []ValidationError{
			{
				Category: "schema",
				Severity: "high",
				Message:  "Invalid field 'enginee', did you mean 'engine'?",
				File:     ".github/workflows/test.md",
				Line:     5,
			},
		},
	}

	output := FormatValidationSummary(results, false)

	// Check for key components
	if !strings.Contains(output, "Compilation failed with 1 error(s)") {
		t.Errorf("Expected error count in output, got: %s", output)
	}

	if !strings.Contains(output, "Error Summary:") {
		t.Errorf("Expected error summary section, got: %s", output)
	}

	if !strings.Contains(output, "High: 1 error(s)") {
		t.Errorf("Expected severity count, got: %s", output)
	}

	if !strings.Contains(output, "By Category:") {
		t.Errorf("Expected category section, got: %s", output)
	}

	if !strings.Contains(output, "Schema: 1 error(s)") {
		t.Errorf("Expected schema category, got: %s", output)
	}

	if !strings.Contains(output, "Recommended Fix Order:") {
		t.Errorf("Expected recommended fix order, got: %s", output)
	}

	if !strings.Contains(output, "Use --verbose") {
		t.Errorf("Expected verbose flag hint, got: %s", output)
	}
}

func TestFormatValidationSummary_MultipleErrors(t *testing.T) {
	results := &ValidationResults{
		Errors: []ValidationError{
			{
				Category: "schema",
				Severity: "high",
				Message:  "Invalid field 'enginee'",
				File:     ".github/workflows/test.md",
				Line:     5,
			},
			{
				Category: "permissions",
				Severity: "critical",
				Message:  "Missing required permission 'contents: read'",
				File:     ".github/workflows/test.md",
				Line:     8,
			},
			{
				Category: "schema",
				Severity: "medium",
				Message:  "Unknown property 'timeout_minutes'",
				File:     ".github/workflows/test.md",
				Line:     12,
			},
		},
	}

	output := FormatValidationSummary(results, false)

	// Check for error count
	if !strings.Contains(output, "Compilation failed with 3 error(s)") {
		t.Errorf("Expected 3 errors in output, got: %s", output)
	}

	// Check severity counts
	if !strings.Contains(output, "Critical: 1 error(s)") {
		t.Errorf("Expected critical severity count, got: %s", output)
	}

	if !strings.Contains(output, "High: 1 error(s)") {
		t.Errorf("Expected high severity count, got: %s", output)
	}

	if !strings.Contains(output, "Medium: 1 error(s)") {
		t.Errorf("Expected medium severity count, got: %s", output)
	}

	// Check category grouping
	if !strings.Contains(output, "Schema: 2 error(s)") {
		t.Errorf("Expected 2 schema errors grouped, got: %s", output)
	}

	if !strings.Contains(output, "Permissions: 1 error(s)") {
		t.Errorf("Expected 1 permissions error grouped, got: %s", output)
	}
}

func TestFormatValidationSummary_VerboseMode(t *testing.T) {
	results := &ValidationResults{
		Errors: []ValidationError{
			{
				Category: "schema",
				Severity: "high",
				Message:  "Invalid field 'enginee'",
				File:     ".github/workflows/test.md",
				Line:     5,
				Hint:     "Did you mean 'engine'?",
			},
			{
				Category: "permissions",
				Severity: "critical",
				Message:  "Missing required permission",
				File:     ".github/workflows/test.md",
				Line:     8,
			},
		},
	}

	output := FormatValidationSummary(results, true)

	// Verbose mode should include detailed errors
	if !strings.Contains(output, "Detailed Errors:") {
		t.Errorf("Expected detailed errors section in verbose mode, got: %s", output)
	}

	// Should contain the error message
	if !strings.Contains(output, "Invalid field 'enginee'") {
		t.Errorf("Expected detailed error message in verbose mode, got: %s", output)
	}

	// Should contain file location
	if !strings.Contains(output, "Location: .github/workflows/test.md:5") {
		t.Errorf("Expected file location in verbose mode, got: %s", output)
	}

	// Should contain hint
	if !strings.Contains(output, "Hint: Did you mean 'engine'?") {
		t.Errorf("Expected hint in verbose mode, got: %s", output)
	}

	// Should NOT show "Use --verbose" message in verbose mode
	if strings.Contains(output, "Use --verbose") {
		t.Errorf("Should not show verbose hint when already in verbose mode, got: %s", output)
	}

	// Should NOT show recommended fix order in verbose mode
	if strings.Contains(output, "Recommended Fix Order:") {
		t.Errorf("Should not show fix order in verbose mode, got: %s", output)
	}
}

func TestGroupErrorsByCategory(t *testing.T) {
	errors := []ValidationError{
		{Category: "schema", Message: "Error 1"},
		{Category: "permissions", Message: "Error 2"},
		{Category: "schema", Message: "Error 3"},
		{Category: "", Message: "Error 4"}, // Empty category
	}

	groups := groupErrorsByCategory(errors)

	// Check schema group has 2 errors
	if len(groups["schema"]) != 2 {
		t.Errorf("Expected 2 schema errors, got %d", len(groups["schema"]))
	}

	// Check permissions group has 1 error
	if len(groups["permissions"]) != 1 {
		t.Errorf("Expected 1 permissions error, got %d", len(groups["permissions"]))
	}

	// Check empty category is assigned to "validation"
	if len(groups["validation"]) != 1 {
		t.Errorf("Expected 1 validation error (empty category), got %d", len(groups["validation"]))
	}
}

func TestFormatValidationSummary_AllSeverityLevels(t *testing.T) {
	results := &ValidationResults{
		Errors: []ValidationError{
			{Category: "security", Severity: "critical", Message: "Critical security issue"},
			{Category: "schema", Severity: "high", Message: "High priority schema error"},
			{Category: "network", Severity: "medium", Message: "Medium network config issue"},
			{Category: "tools", Severity: "low", Message: "Low priority tool warning"},
		},
	}

	output := FormatValidationSummary(results, false)

	// All severity levels should be present
	if !strings.Contains(output, "Critical: 1 error(s)") {
		t.Errorf("Expected critical severity in output")
	}
	if !strings.Contains(output, "High: 1 error(s)") {
		t.Errorf("Expected high severity in output")
	}
	if !strings.Contains(output, "Medium: 1 error(s)") {
		t.Errorf("Expected medium severity in output")
	}
	if !strings.Contains(output, "Low: 1 error(s)") {
		t.Errorf("Expected low severity in output")
	}
}

func TestFormatValidationSummary_CategoryEmojis(t *testing.T) {
	results := &ValidationResults{
		Errors: []ValidationError{
			{Category: "schema", Severity: "high", Message: "Schema error"},
			{Category: "permissions", Severity: "high", Message: "Permission error"},
			{Category: "network", Severity: "high", Message: "Network error"},
			{Category: "security", Severity: "high", Message: "Security error"},
			{Category: "engine", Severity: "high", Message: "Engine error"},
			{Category: "tools", Severity: "high", Message: "Tools error"},
		},
	}

	output := FormatValidationSummary(results, true)

	// In verbose mode, emojis should appear in detailed errors
	// Just verify the output is generated without error
	if output == "" {
		t.Errorf("Expected non-empty output with emojis")
	}
}
