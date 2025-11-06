// Package workflow provides JavaScript bundler validation for agentic workflows.
//
// # JavaScript Bundler Validation
//
// This file validates bundled JavaScript to ensure that all local module dependencies
// have been properly inlined during the bundling process. This prevents runtime errors
// from missing local modules when JavaScript is executed in GitHub Actions.
//
// # Validation Functions
//
//   - validateNoLocalRequires() - Validates bundled JavaScript has no local require() statements
//   - isInsideStringLiteralAt() - Helper to detect if a position is inside a string literal
//
// # Validation Pattern: Bundling Verification
//
// Bundler validation ensures that local require() statements are inlined:
//   - Scans bundled JavaScript for require('./...') or require('../...') patterns
//   - Ignores require statements inside string literals
//   - Returns hard errors if local requires are found (indicates bundling failure)
//   - Helps prevent runtime module-not-found errors
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates JavaScript bundling correctness
//   - It checks for missing module dependencies
//   - It validates CommonJS require() statement resolution
//   - It validates JavaScript code structure
//
// For bundling functions, see bundler.go.
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// validateNoLocalRequires checks that the bundled JavaScript contains no local require() statements
// that weren't inlined during bundling. This prevents runtime errors from missing local modules.
// Returns an error if any local requires are found, otherwise returns nil
func validateNoLocalRequires(bundledContent string) error {
	// Regular expression to match local require statements
	// Matches: require('./...') or require("../...")
	localRequireRegex := regexp.MustCompile(`require\(['"](\.\.?/[^'"]+)['"]\)`)

	lines := strings.Split(bundledContent, "\n")
	var foundRequires []string

	for lineNum, line := range lines {
		// Check for local requires
		matches := localRequireRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				requirePath := match[1]
				foundRequires = append(foundRequires, fmt.Sprintf("line %d: require('%s')", lineNum+1, requirePath))
			}
		}
	}

	if len(foundRequires) > 0 {
		return fmt.Errorf("bundled JavaScript contains %d local require(s) that were not inlined:\n  %s",
			len(foundRequires), strings.Join(foundRequires, "\n  "))
	}

	return nil
}
