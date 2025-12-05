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

	"github.com/githubnext/gh-aw/pkg/logger"
)

var bundlerValidationLog = logger.New("workflow:bundler_validation")

// validateNoLocalRequires checks that the bundled JavaScript contains no local require() statements
// that weren't inlined during bundling. This prevents runtime errors from missing local modules.
// Returns an error if any local requires are found, otherwise returns nil
func validateNoLocalRequires(bundledContent string) error {
	bundlerValidationLog.Printf("Validating bundled JavaScript: %d bytes, %d lines", len(bundledContent), strings.Count(bundledContent, "\n")+1)

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
		bundlerValidationLog.Printf("Validation failed: found %d un-inlined local require statements", len(foundRequires))
		return fmt.Errorf("bundled JavaScript contains %d local require(s) that were not inlined:\n  %s",
			len(foundRequires), strings.Join(foundRequires, "\n  "))
	}

	bundlerValidationLog.Print("Validation successful: no local require statements found")
	return nil
}

// validateNoModuleReferences checks that the bundled JavaScript contains no module.exports or exports references
// This is required for GitHub Script mode where no module system exists.
// Returns an error if any module references are found, otherwise returns nil
func validateNoModuleReferences(bundledContent string) error {
	bundlerValidationLog.Printf("Validating no module references: %d bytes", len(bundledContent))

	// Regular expressions to match module references
	// Match: module.exports or exports.
	moduleExportsRegex := regexp.MustCompile(`\bmodule\.exports\b`)
	exportsRegex := regexp.MustCompile(`\bexports\.\w+`)

	lines := strings.Split(bundledContent, "\n")
	var foundReferences []string

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip comment lines
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Check for module.exports
		if moduleExportsRegex.MatchString(line) {
			foundReferences = append(foundReferences, fmt.Sprintf("line %d: module.exports reference", lineNum+1))
		}

		// Check for exports.
		if exportsRegex.MatchString(line) {
			foundReferences = append(foundReferences, fmt.Sprintf("line %d: exports reference", lineNum+1))
		}
	}

	if len(foundReferences) > 0 {
		bundlerValidationLog.Printf("Validation failed: found %d module references", len(foundReferences))
		return fmt.Errorf("bundled JavaScript for GitHub Script mode contains %d module reference(s) that should have been removed:\n  %s\n\nGitHub Script mode does not support module.exports or exports. These references must be removed during bundling.",
			len(foundReferences), strings.Join(foundReferences, "\n  "))
	}

	bundlerValidationLog.Print("Validation successful: no module references found")
	return nil
}
