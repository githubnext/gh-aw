package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// Pre-compiled regexes for expression validation (performance optimization)
var (
	expressionRegex = regexp.MustCompile(`(?s)\$\{\{(.*?)\}\}`)
	needsStepsRegex = regexp.MustCompile(`^(needs|steps)\.[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)
	inputsRegex     = regexp.MustCompile(`^github\.event\.inputs\.[a-zA-Z0-9_-]+$`)
	envRegex        = regexp.MustCompile(`^env\.[a-zA-Z0-9_-]+$`)
)

// validateExpressionSafety checks that all GitHub Actions expressions in the markdown content
// are in the allowed list and returns an error if any unauthorized expressions are found.
// If strictMode is true, it first checks for common typos and provides helpful error messages.
func validateExpressionSafety(markdownContent string, strictMode bool) error {
	// In strict mode, check for forbidden expressions first and provide helpful error messages
	if strictMode {
		if err := checkForbiddenExpressions(markdownContent); err != nil {
			return err
		}
	}

	// Regular expression to match GitHub Actions expressions: ${{ ... }}
	// Use (?s) flag to enable dotall mode so . matches newlines to capture multiline expressions
	// Use non-greedy matching with .*? to handle nested braces properly

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

		// Try to parse the expression using the parser
		parsed, parseErr := ParseExpression(expression)
		if parseErr == nil {
			// If we can parse it, validate each literal expression in the tree
			validationErr := VisitExpressionTree(parsed, func(expr *ExpressionNode) error {
				return validateSingleExpression(expr.Expression, needsStepsRegex, inputsRegex, envRegex, &unauthorizedExpressions)
			})
			if validationErr != nil {
				return validationErr
			}
		} else {
			// If parsing fails, fall back to validating the whole expression as a literal
			err := validateSingleExpression(expression, needsStepsRegex, inputsRegex, envRegex, &unauthorizedExpressions)
			if err != nil {
				return err
			}
		}
	}

	// If we found unauthorized expressions, return an error
	if len(unauthorizedExpressions) > 0 {
		// Format unauthorized expressions list
		var unauthorizedList strings.Builder
		unauthorizedList.WriteString("\n")
		for _, expr := range unauthorizedExpressions {
			unauthorizedList.WriteString("  - ")
			unauthorizedList.WriteString(expr)
			unauthorizedList.WriteString("\n")
		}

		// Format allowed expressions list
		var allowedList strings.Builder
		allowedList.WriteString("\n")
		for _, expr := range constants.AllowedExpressions {
			allowedList.WriteString("  - ")
			allowedList.WriteString(expr)
			allowedList.WriteString("\n")
		}
		allowedList.WriteString("  - needs.*\n")
		allowedList.WriteString("  - steps.*\n")
		allowedList.WriteString("  - github.event.inputs.*\n")
		allowedList.WriteString("  - env.*\n")

		return fmt.Errorf("unauthorized expressions:%s\nallowed:%s",
			unauthorizedList.String(), allowedList.String())
	}

	return nil
}

// validateSingleExpression validates a single literal expression
func validateSingleExpression(expression string, needsStepsRe, inputsRe, envRe *regexp.Regexp, unauthorizedExpressions *[]string) error {
	expression = strings.TrimSpace(expression)

	// Check if this expression is in the allowed list
	allowed := false

	// Check if this expression starts with "needs." or "steps." and is a simple property access
	if needsStepsRe.MatchString(expression) {
		allowed = true
	} else if inputsRe.MatchString(expression) {
		// Check if this expression matches github.event.inputs.* pattern
		allowed = true
	} else if envRe.MatchString(expression) {
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
		*unauthorizedExpressions = append(*unauthorizedExpressions, expression)
	}

	return nil
}

// checkForbiddenExpressions checks for common typo expressions in strict mode
// and provides helpful error messages for users
func checkForbiddenExpressions(markdownContent string) error {
	// Find all expressions in the markdown content
	matches := expressionRegex.FindAllStringSubmatch(markdownContent, -1)

	// List of forbidden expression patterns in strict mode
	forbiddenPatterns := []string{
		"git.workflow",
		"git.agent",
	}

	var foundForbidden []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		// Extract the expression content (everything between ${{ and }})
		expression := strings.TrimSpace(match[1])

		// Check if this expression contains any forbidden pattern
		for _, forbidden := range forbiddenPatterns {
			if strings.Contains(expression, forbidden) {
				foundForbidden = append(foundForbidden, expression)
				break
			}
		}
	}

	// If we found forbidden expressions, return an error
	if len(foundForbidden) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString("strict mode: forbidden expressions found (common typos):\n")
		for _, expr := range foundForbidden {
			errMsg.WriteString(fmt.Sprintf("  - %s\n", expr))
		}
		errMsg.WriteString("\nDid you mean:\n")
		errMsg.WriteString("  - github.workflow instead of git.workflow\n")
		errMsg.WriteString("  - github.actor instead of git.agent")

		return fmt.Errorf("%s", errMsg.String())
	}

	return nil
}
