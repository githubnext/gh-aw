package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/console"
)

// TestValidationSummaryIntegration tests the full integration of validation error collection
// This demonstrates how validation errors from different categories can be collected
// and displayed with a comprehensive summary
func TestValidationSummaryIntegration(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetCollectErrors(true)
	compiler.ResetValidationResults()

	// Simulate validation errors from different validation stages
	// These would normally come from different validation functions

	// 1. Schema validation errors
	compiler.AddValidationError(
		"schema",
		"high",
		"Invalid field 'enginee' in frontmatter. Did you mean 'engine'?",
		".github/workflows/example.md",
		3,
		"Check the spelling of frontmatter fields",
	)

	compiler.AddValidationError(
		"schema",
		"medium",
		"Unknown property 'timeout_minutes'. Use 'timeout-minutes' instead",
		".github/workflows/example.md",
		5,
		"Use hyphenated field names for GitHub Actions compatibility",
	)

	// 2. Permission validation errors
	compiler.AddValidationError(
		"permissions",
		"critical",
		"Missing required permission 'contents: read' for repository access",
		".github/workflows/example.md",
		10,
		"Add 'contents: read' to the permissions section",
	)

	compiler.AddValidationError(
		"permissions",
		"high",
		"Permission 'issues: write' requires 'issues: read' to be set",
		".github/workflows/example.md",
		11,
		"Add 'issues: read' along with 'issues: write'",
	)

	// 3. Network validation error
	compiler.AddValidationError(
		"network",
		"medium",
		"Network access to 'api.example.com' is not allowed by default",
		".github/workflows/example.md",
		20,
		"Add 'api.example.com' to the allowed domains list",
	)

	// 4. Security validation error
	compiler.AddValidationError(
		"security",
		"critical",
		"Expression ${{ github.event.issue.title }} contains untrusted user input",
		".github/workflows/example.md",
		30,
		"Use needs.activation.outputs.text for sanitized content",
	)

	// 5. Tool validation warning
	compiler.AddValidationWarning(
		"tools",
		"low",
		"Tool 'bash' configured but not used in workflow",
		".github/workflows/example.md",
		15,
		"Remove unused tool configurations to reduce overhead",
	)

	// Verify error collection
	if !compiler.HasValidationErrors() {
		t.Fatal("Expected validation errors to be collected")
	}

	results := compiler.GetValidationResults()

	// Verify error counts
	expectedErrorCount := 6
	if len(results.Errors) != expectedErrorCount {
		t.Errorf("Expected %d errors, got %d", expectedErrorCount, len(results.Errors))
	}

	// Verify warning count
	expectedWarningCount := 1
	if len(results.Warnings) != expectedWarningCount {
		t.Errorf("Expected %d warnings, got %d", expectedWarningCount, len(results.Warnings))
	}

	// Verify errors by category
	categoryCount := make(map[string]int)
	for _, err := range results.Errors {
		categoryCount[err.Category]++
	}

	expectedCategories := map[string]int{
		"schema":      2,
		"permissions": 2,
		"network":     1,
		"security":    1,
	}

	for category, expectedCount := range expectedCategories {
		if categoryCount[category] != expectedCount {
			t.Errorf("Expected %d %s errors, got %d", expectedCount, category, categoryCount[category])
		}
	}

	// Verify errors by severity
	severityCount := make(map[string]int)
	for _, err := range results.Errors {
		severityCount[err.Severity]++
	}

	expectedSeverities := map[string]int{
		"critical": 2,
		"high":     2,
		"medium":   2,
	}

	for severity, expectedCount := range expectedSeverities {
		if severityCount[severity] != expectedCount {
			t.Errorf("Expected %d %s severity errors, got %d", expectedCount, severity, severityCount[severity])
		}
	}

	// Test non-verbose summary
	t.Run("NonVerboseSummary", func(t *testing.T) {
		summary := console.FormatValidationSummary(results, false)

		// Verify key components are present
		requiredStrings := []string{
			"Compilation failed with 6 error(s)",
			"Error Summary:",
			"Critical: 2 error(s)",
			"High: 2 error(s)",
			"Medium: 2 error(s)",
			"By Category:",
			"Schema: 2 error(s)",
			"Permissions: 2 error(s)",
			"Network: 1 error(s)",
			"Security: 1 error(s)",
			"Recommended Fix Order:",
			"Use --verbose",
		}

		for _, required := range requiredStrings {
			if !strings.Contains(summary, required) {
				t.Errorf("Expected summary to contain '%s', but it didn't.\nSummary:\n%s", required, summary)
			}
		}

		t.Logf("Non-verbose summary:\n%s", summary)
	})

	// Test verbose summary
	t.Run("VerboseSummary", func(t *testing.T) {
		verboseSummary := console.FormatValidationSummary(results, true)

		// Verify detailed error information is present
		requiredStrings := []string{
			"Compilation failed with 6 error(s)",
			"Detailed Errors:",
			"Invalid field 'enginee'",
			"Missing required permission",
			"Network access to 'api.example.com'",
			"Expression ${{ github.event.issue.title }}",
			"Location: .github/workflows/example.md",
			"Hint:",
		}

		for _, required := range requiredStrings {
			if !strings.Contains(verboseSummary, required) {
				t.Errorf("Expected verbose summary to contain '%s', but it didn't.\nSummary:\n%s", required, verboseSummary)
			}
		}

		// Verify that "Use --verbose" is NOT in verbose mode
		if strings.Contains(verboseSummary, "Use --verbose") {
			t.Error("Verbose summary should not contain 'Use --verbose' message")
		}

		// Verify that "Recommended Fix Order" is NOT in verbose mode
		if strings.Contains(verboseSummary, "Recommended Fix Order:") {
			t.Error("Verbose summary should not contain 'Recommended Fix Order' section")
		}

		t.Logf("Verbose summary:\n%s", verboseSummary)
	})
}

// TestValidationSummaryRealWorldScenario tests a realistic scenario with common validation errors
func TestValidationSummaryRealWorldScenario(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetCollectErrors(true)
	compiler.ResetValidationResults()

	// Simulate common mistakes in workflow configuration
	compiler.AddValidationError(
		"schema",
		"high",
		"Field 'timeout_minutes' should be 'timeout-minutes'",
		".github/workflows/issue-triage.md",
		7,
		"GitHub Actions uses hyphenated field names",
	)

	compiler.AddValidationError(
		"permissions",
		"high",
		"Write permission 'issues: write' requires explicit 'contents: read'",
		".github/workflows/issue-triage.md",
		10,
		"Always specify 'contents: read' when using write permissions",
	)

	compiler.AddValidationError(
		"engine",
		"medium",
		"Engine field value 'copliot' is not recognized. Did you mean 'copilot'?",
		".github/workflows/issue-triage.md",
		4,
		"Valid engines are: copilot, claude, codex, custom",
	)

	results := compiler.GetValidationResults()
	summary := console.FormatValidationSummary(results, false)

	// This should produce a helpful, actionable summary
	if summary == "" {
		t.Error("Expected non-empty validation summary")
	}

	// The summary should guide the user to fix errors in a logical order
	if !strings.Contains(summary, "Recommended Fix Order:") {
		t.Error("Expected recommended fix order in summary")
	}

	t.Logf("Real-world scenario summary:\n%s", summary)
}
