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
	// comparisonExtractionRegex extracts property accesses from comparison expressions
	// Matches patterns like "github.workflow == 'value'" and extracts "github.workflow"
	comparisonExtractionRegex = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_.]*)\s*(?:==|!=|<|>|<=|>=)\s*`)
)

// validateExpressionSafety checks that all GitHub Actions expressions in the markdown content
// are in the allowed list and returns an error if any unauthorized expressions are found
func validateExpressionSafety(markdownContent string) error {
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

	// If not allowed as a whole, try to extract and validate property accesses from comparisons
	if !allowed {
		// Extract property accesses from comparison expressions (e.g., "github.workflow == 'value'")
		matches := comparisonExtractionRegex.FindAllStringSubmatch(expression, -1)
		if len(matches) > 0 {
			// Assume it's allowed if all extracted properties are allowed
			allPropertiesAllowed := true
			for _, match := range matches {
				if len(match) > 1 {
					property := strings.TrimSpace(match[1])
					propertyAllowed := false

					// Check if extracted property is allowed
					if needsStepsRe.MatchString(property) {
						propertyAllowed = true
					} else if inputsRe.MatchString(property) {
						propertyAllowed = true
					} else if envRe.MatchString(property) {
						propertyAllowed = true
					} else {
						for _, allowedExpr := range constants.AllowedExpressions {
							if property == allowedExpr {
								propertyAllowed = true
								break
							}
						}
					}

					if !propertyAllowed {
						allPropertiesAllowed = false
						break
					}
				}
			}

			if allPropertiesAllowed && len(matches) > 0 {
				allowed = true
			}
		}
	}

	if !allowed {
		*unauthorizedExpressions = append(*unauthorizedExpressions, expression)
	}

	return nil
}
