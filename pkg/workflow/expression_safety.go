package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// validateExpressionSafety checks that all GitHub Actions expressions in the markdown content
// are in the allowed list and returns an error if any unauthorized expressions are found
func validateExpressionSafety(markdownContent string) error {
	// Regular expression to match GitHub Actions expressions: ${{ ... }}
	// Use (?s) flag to enable dotall mode so . matches newlines to capture multiline expressions
	// Use non-greedy matching with .*? to handle nested braces properly
	expressionRegex := regexp.MustCompile(`(?s)\$\{\{(.*?)\}\}`)
	needsStepsRegex := regexp.MustCompile(`^(needs|steps)\.[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)
	inputsRegex := regexp.MustCompile(`^github\.event\.inputs\.[a-zA-Z0-9_-]+$`)
	envRegex := regexp.MustCompile(`^env\.[a-zA-Z0-9_-]+$`)

	// Find all expressions in the markdown content
	matches := expressionRegex.FindAllStringSubmatch(markdownContent, -1)

	var unauthorizedExpressions []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		// Extract the expression content (everything between ${{ and }})
		expression := strings.TrimSpace(match[1])

		// Reject expressions that span multiple lines (contain newlines)
		if strings.Contains(match[1], "\n") {
			unauthorizedExpressions = append(unauthorizedExpressions, expression)
			continue
		}

		// Check if this expression is in the allowed list
		allowed := false

		// Check if this expression starts with "needs." or "steps." and is a simple property access
		if needsStepsRegex.MatchString(expression) {
			allowed = true
		} else if inputsRegex.MatchString(expression) {
			// Check if this expression matches github.event.inputs.* pattern
			allowed = true
		} else if envRegex.MatchString(expression) {
			// check if this expression matches env.* pattern
			allowed = true
		} else {
			for _, allowedExpr := range constants.AllowedExpressions {
				if expression == allowedExpr {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			unauthorizedExpressions = append(unauthorizedExpressions, expression)
		}
	}

	// If we found unauthorized expressions, return an error
	if len(unauthorizedExpressions) > 0 {
		return fmt.Errorf("unauthorized expressions: %v. allowed: %v",
			unauthorizedExpressions, constants.AllowedExpressions)
	}

	return nil
}
