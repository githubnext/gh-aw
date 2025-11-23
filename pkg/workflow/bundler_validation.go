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

// MaxLineLengthForActions is the maximum line length allowed in GitHub Actions YAML
// GitHub Actions has a limit of 21k characters per line, but we use 20k to be safe
const MaxLineLengthForActions = 20000

// validateLineLength checks that no line in the bundled JavaScript exceeds the GitHub Actions limit
// GitHub Actions has a 21k character limit per line in YAML. We validate at 20k to be safe.
// Returns an error if any line exceeds the limit, otherwise returns nil
func validateLineLength(bundledContent string) error {
	lines := strings.Split(bundledContent, "\n")
	
	var longLines []string
	maxLineLength := 0
	
	for lineNum, line := range lines {
		lineLength := len(line)
		if lineLength > maxLineLength {
			maxLineLength = lineLength
		}
		
		if lineLength > MaxLineLengthForActions {
			longLines = append(longLines, fmt.Sprintf("line %d: %d characters (exceeds %d limit)", 
				lineNum+1, lineLength, MaxLineLengthForActions))
		}
	}
	
	if len(longLines) > 0 {
		bundlerValidationLog.Printf("Validation failed: found %d lines exceeding %d character limit", 
			len(longLines), MaxLineLengthForActions)
		return fmt.Errorf("bundled JavaScript contains %d line(s) exceeding GitHub Actions %d character limit:\n  %s\n\nConsider breaking up long lines or splitting functionality into separate files",
			len(longLines), MaxLineLengthForActions, strings.Join(longLines, "\n  "))
	}
	
	bundlerValidationLog.Printf("Validation successful: max line length is %d characters (within %d limit)", 
		maxLineLength, MaxLineLengthForActions)
	return nil
}
