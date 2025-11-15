package workflow

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/console"
)

func TestValidationErrorCollection(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")

	// Enable error collection
	compiler.SetCollectErrors(true)
	compiler.ResetValidationResults()

	// Simulate adding multiple validation errors
	compiler.AddValidationError(
		"schema",
		"high",
		"Invalid field 'enginee', did you mean 'engine'?",
		".github/workflows/test.md",
		5,
		"Check for typos in field names",
	)

	compiler.AddValidationError(
		"permissions",
		"critical",
		"Missing required permission 'contents: read'",
		".github/workflows/test.md",
		8,
		"Add permissions section to frontmatter",
	)

	compiler.AddValidationError(
		"schema",
		"medium",
		"Unknown property 'timeout_minutes', use 'timeout-minutes' instead",
		".github/workflows/test.md",
		12,
		"Use hyphenated field names in YAML",
	)

	// Verify errors were collected
	if !compiler.HasValidationErrors() {
		t.Fatal("Expected validation errors to be collected")
	}

	results := compiler.GetValidationResults()
	if len(results.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(results.Errors))
	}

	// Verify errors have correct categories
	categories := make(map[string]int)
	for _, err := range results.Errors {
		categories[err.Category]++
	}

	if categories["schema"] != 2 {
		t.Errorf("Expected 2 schema errors, got %d", categories["schema"])
	}

	if categories["permissions"] != 1 {
		t.Errorf("Expected 1 permissions error, got %d", categories["permissions"])
	}

	// Test formatting the summary
	summary := console.FormatValidationSummary(results, false)
	if summary == "" {
		t.Error("Expected non-empty validation summary")
	}

	t.Logf("Validation Summary (non-verbose):\n%s", summary)

	// Test verbose formatting
	verboseSummary := console.FormatValidationSummary(results, true)
	if verboseSummary == "" {
		t.Error("Expected non-empty verbose validation summary")
	}

	t.Logf("Validation Summary (verbose):\n%s", verboseSummary)
}

func TestValidationErrorCollectionDisabled(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")

	// When error collection is disabled (default), errors should not accumulate
	compiler.SetCollectErrors(false)

	// Even if we add errors, they should still be collected for reporting purposes
	// The difference is in how the compilation flow handles them
	compiler.ResetValidationResults()
	compiler.AddValidationError(
		"schema",
		"high",
		"Test error",
		".github/workflows/test.md",
		1,
		"",
	)

	// Errors should still be added to results even when collection is disabled
	// The collectErrors flag controls whether compilation continues or stops
	if !compiler.HasValidationErrors() {
		t.Fatal("Expected validation error to be added even with collection disabled")
	}
}

func TestValidationWarningCollection(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetCollectErrors(true)
	compiler.ResetValidationResults()

	// Add some warnings
	compiler.AddValidationWarning(
		"tools",
		"low",
		"Unused tool configuration detected",
		".github/workflows/test.md",
		15,
		"Remove unused tool configurations",
	)

	results := compiler.GetValidationResults()
	if len(results.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(results.Warnings))
	}

	if len(results.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(results.Errors))
	}
}

func TestResetValidationResults(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetCollectErrors(true)
	compiler.ResetValidationResults()

	// Add an error
	compiler.AddValidationError(
		"schema",
		"high",
		"Test error",
		".github/workflows/test.md",
		1,
		"",
	)

	if !compiler.HasValidationErrors() {
		t.Fatal("Expected validation error")
	}

	// Reset validation results
	compiler.ResetValidationResults()

	// Should have no errors after reset
	if compiler.HasValidationErrors() {
		t.Error("Expected no validation errors after reset")
	}

	results := compiler.GetValidationResults()
	if len(results.Errors) != 0 {
		t.Errorf("Expected 0 errors after reset, got %d", len(results.Errors))
	}
}
