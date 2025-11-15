package console

import (
	"fmt"
	"sort"
	"strings"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Category string // "schema", "permissions", "network", "security", "engine", etc.
	Severity string // "critical", "high", "medium", "low"
	Message  string
	File     string
	Line     int
	Hint     string
}

// ValidationResults holds all validation errors and warnings
type ValidationResults struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

// severityOrder defines the display order for severity levels
var severityOrder = map[string]int{
	"critical": 1,
	"high":     2,
	"medium":   3,
	"low":      4,
}

// categoryEmoji maps category to emoji for visual identification
var categoryEmoji = map[string]string{
	"schema":      "❌",
	"permissions": "🔒",
	"network":     "🌐",
	"security":    "🛡️",
	"engine":      "🤖",
	"tools":       "🔧",
	"validation":  "⚠️",
	"syntax":      "📝",
}

// FormatValidationSummary formats validation results into a user-friendly summary
func FormatValidationSummary(results *ValidationResults, verbose bool) string {
	if len(results.Errors) == 0 && len(results.Warnings) == 0 {
		return ""
	}

	var output strings.Builder

	// Header
	if len(results.Errors) > 0 {
		output.WriteString(FormatErrorMessage(fmt.Sprintf("Compilation failed with %d error(s)", len(results.Errors))))
		output.WriteString("\n\n")
	}

	// Summary counts by severity
	if len(results.Errors) > 0 {
		severityCounts := make(map[string]int)
		for _, err := range results.Errors {
			if err.Severity != "" {
				severityCounts[err.Severity]++
			}
		}

		if len(severityCounts) > 0 {
			output.WriteString(FormatListHeader("Error Summary:"))
			output.WriteString("\n")

			// Sort by severity
			severities := []string{"critical", "high", "medium", "low"}
			for _, severity := range severities {
				if count, ok := severityCounts[severity]; ok && count > 0 {
					output.WriteString(fmt.Sprintf("  %s: %d error(s)\n", strings.Title(severity), count))
				}
			}
			output.WriteString("\n")
		}
	}

	// Group errors by category
	if len(results.Errors) > 0 {
		categoryGroups := groupErrorsByCategory(results.Errors)

		if len(categoryGroups) > 0 {
			output.WriteString(FormatListHeader("By Category:"))
			output.WriteString("\n")

			// Sort categories alphabetically
			categories := make([]string, 0, len(categoryGroups))
			for category := range categoryGroups {
				categories = append(categories, category)
			}
			sort.Strings(categories)

			for _, category := range categories {
				errors := categoryGroups[category]
				emoji := categoryEmoji[category]
				if emoji == "" {
					emoji = "⚠️"
				}
				output.WriteString(fmt.Sprintf("  %s %s: %d error(s)\n", emoji, strings.Title(category), len(errors)))
			}
			output.WriteString("\n")
		}
	}

	// Recommended fix order
	if len(results.Errors) > 0 && !verbose {
		output.WriteString(FormatListHeader("Recommended Fix Order:"))
		output.WriteString("\n")
		output.WriteString("  1. Fix schema errors first (typos, invalid fields)\n")
		output.WriteString("  2. Address permission issues\n")
		output.WriteString("  3. Configure network access\n")
		output.WriteString("  4. Review security warnings\n")
		output.WriteString("\n")
	}

	// Detailed errors in verbose mode
	if verbose && len(results.Errors) > 0 {
		output.WriteString(FormatListHeader("Detailed Errors:"))
		output.WriteString("\n\n")

		// Sort errors by severity, then category
		sortedErrors := make([]ValidationError, len(results.Errors))
		copy(sortedErrors, results.Errors)
		sort.Slice(sortedErrors, func(i, j int) bool {
			// Sort by severity first
			iSeverity := severityOrder[sortedErrors[i].Severity]
			jSeverity := severityOrder[sortedErrors[j].Severity]
			if iSeverity != jSeverity {
				return iSeverity < jSeverity
			}
			// Then by category
			return sortedErrors[i].Category < sortedErrors[j].Category
		})

		for i, err := range sortedErrors {
			// Category and severity badge
			emoji := categoryEmoji[err.Category]
			if emoji == "" {
				emoji = "⚠️"
			}
			output.WriteString(fmt.Sprintf("%d. %s [%s] %s\n", i+1, emoji, strings.ToUpper(err.Severity), strings.Title(err.Category)))

			// Error message
			output.WriteString(fmt.Sprintf("   %s\n", err.Message))

			// File location if available
			if err.File != "" {
				location := err.File
				if err.Line > 0 {
					location = fmt.Sprintf("%s:%d", location, err.Line)
				}
				output.WriteString(fmt.Sprintf("   Location: %s\n", location))
			}

			// Hint if available
			if err.Hint != "" {
				output.WriteString(fmt.Sprintf("   Hint: %s\n", err.Hint))
			}

			output.WriteString("\n")
		}
	}

	// Help text
	if !verbose && len(results.Errors) > 0 {
		output.WriteString(FormatInfoMessage("Use --verbose to see detailed error messages"))
		output.WriteString("\n")
	}

	return output.String()
}

// groupErrorsByCategory groups errors by their category
func groupErrorsByCategory(errors []ValidationError) map[string][]ValidationError {
	groups := make(map[string][]ValidationError)
	for _, err := range errors {
		category := err.Category
		if category == "" {
			category = "validation"
		}
		groups[category] = append(groups[category], err)
	}
	return groups
}
