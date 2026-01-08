// Package workflow provides GitHub Actions expression security validation.
//
// # Expression Safety Validation
//
// This file validates that GitHub Actions expressions used in workflow markdown
// are safe and authorized. It prevents injection attacks and ensures workflows
// only use approved expression patterns.
//
// # Validation Functions
//
//   - validateExpressionSafety() - Validates all expressions in markdown content
//   - validateSingleExpression() - Validates individual expression syntax
//
// # Validation Pattern: Allowlist Security
//
// Expression validation uses a strict allowlist approach:
//   - Only pre-approved GitHub context expressions are allowed
//   - Unauthorized expressions cause compilation to fail
//   - Prevents injection of secrets or environment variables
//   - Uses regex patterns to match allowed expression formats
//
// # Allowed Expression Patterns
//
// Expressions must match one of these patterns:
//   - github.event.* (event context properties)
//   - github.actor, github.repository, etc. (core GitHub context)
//   - needs.*.outputs.* (job dependencies)
//   - steps.*.outputs.* (step outputs)
//   - github.event.inputs.* (workflow_dispatch inputs)
//
// See pkg/constants for the complete list of allowed expressions.
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates GitHub Actions expression parsing
//   - It enforces expression security policies
//   - It prevents expression injection attacks
//   - It validates expression syntax and structure
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var expressionValidationLog = logger.New("workflow:expression_validation")

// maxFuzzyMatchSuggestions is the maximum number of similar expressions to suggest
// when an unauthorized expression is found
const maxFuzzyMatchSuggestions = 7

// Pre-compiled regexes for expression validation (performance optimization)
var (
	expressionRegex         = regexp.MustCompile(`(?s)\$\{\{(.*?)\}\}`)
	needsStepsRegex         = regexp.MustCompile(`^(needs|steps)\.[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)
	inputsRegex             = regexp.MustCompile(`^github\.event\.inputs\.[a-zA-Z0-9_-]+$`)
	workflowCallInputsRegex = regexp.MustCompile(`^inputs\.[a-zA-Z0-9_-]+$`)
	awInputsRegex           = regexp.MustCompile(`^github\.aw\.inputs\.[a-zA-Z0-9_-]+$`)
	envRegex                = regexp.MustCompile(`^env\.[a-zA-Z0-9_-]+$`)
	// comparisonExtractionRegex extracts property accesses from comparison expressions
	// Matches patterns like "github.workflow == 'value'" and extracts "github.workflow"
	comparisonExtractionRegex = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_.]*)\s*(?:==|!=|<|>|<=|>=)\s*`)
)

// validateExpressionSafety checks that all GitHub Actions expressions in the markdown content
// are in the allowed list and returns an error if any unauthorized expressions are found
func validateExpressionSafety(markdownContent string) error {
	expressionValidationLog.Print("Validating expression safety in markdown content")

	// Regular expression to match GitHub Actions expressions: ${{ ... }}
	// Use (?s) flag to enable dotall mode so . matches newlines to capture multiline expressions
	// Use non-greedy matching with .*? to handle nested braces properly

	// Find all expressions in the markdown content
	matches := expressionRegex.FindAllStringSubmatch(markdownContent, -1)
	expressionValidationLog.Printf("Found %d expressions to validate", len(matches))

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
				return validateSingleExpression(expr.Expression, ExpressionValidationOptions{
					NeedsStepsRegex:         needsStepsRegex,
					InputsRegex:             inputsRegex,
					WorkflowCallInputsRegex: workflowCallInputsRegex,
					AwInputsRegex:           awInputsRegex,
					EnvRegex:                envRegex,
					UnauthorizedExpressions: &unauthorizedExpressions,
				})
			})
			if validationErr != nil {
				return validationErr
			}
		} else {
			// If parsing fails, fall back to validating the whole expression as a literal
			err := validateSingleExpression(expression, ExpressionValidationOptions{
				NeedsStepsRegex:         needsStepsRegex,
				InputsRegex:             inputsRegex,
				WorkflowCallInputsRegex: workflowCallInputsRegex,
				AwInputsRegex:           awInputsRegex,
				EnvRegex:                envRegex,
				UnauthorizedExpressions: &unauthorizedExpressions,
			})
			if err != nil {
				return err
			}
		}
	}

	// If we found unauthorized expressions, return an error
	if len(unauthorizedExpressions) > 0 {
		expressionValidationLog.Printf("Expression safety validation failed: %d unauthorized expressions found", len(unauthorizedExpressions))
		// Format unauthorized expressions list with fuzzy match suggestions
		var unauthorizedList strings.Builder
		unauthorizedList.WriteString("\n")
		for _, expr := range unauthorizedExpressions {
			unauthorizedList.WriteString("  - ")
			unauthorizedList.WriteString(expr)

			// Find closest matches using fuzzy string matching
			closestMatches := parser.FindClosestMatches(expr, constants.AllowedExpressions, maxFuzzyMatchSuggestions)
			if len(closestMatches) > 0 {
				unauthorizedList.WriteString(" (did you mean: ")
				unauthorizedList.WriteString(strings.Join(closestMatches, ", "))
				unauthorizedList.WriteString("?)")
			}

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
		allowedList.WriteString("  - github.aw.inputs.* (shared workflow inputs)\n")
		allowedList.WriteString("  - inputs.* (workflow_call)\n")
		allowedList.WriteString("  - env.*\n")

		return fmt.Errorf("unauthorized expressions:%s\nallowed:%s",
			unauthorizedList.String(), allowedList.String())
	}

	expressionValidationLog.Print("Expression safety validation passed")
	return nil
}

// ExpressionValidationOptions contains the options for validating a single expression
type ExpressionValidationOptions struct {
	NeedsStepsRegex         *regexp.Regexp
	InputsRegex             *regexp.Regexp
	WorkflowCallInputsRegex *regexp.Regexp
	AwInputsRegex           *regexp.Regexp
	EnvRegex                *regexp.Regexp
	UnauthorizedExpressions *[]string
}

// validateSingleExpression validates a single literal expression
func validateSingleExpression(expression string, opts ExpressionValidationOptions) error {
	expression = strings.TrimSpace(expression)

	// Check if this expression is in the allowed list
	allowed := false

	// Check if this expression starts with "needs." or "steps." and is a simple property access
	if opts.NeedsStepsRegex.MatchString(expression) {
		allowed = true
	} else if opts.InputsRegex.MatchString(expression) {
		// Check if this expression matches github.event.inputs.* pattern
		allowed = true
	} else if opts.WorkflowCallInputsRegex.MatchString(expression) {
		// Check if this expression matches inputs.* pattern (workflow_call inputs)
		allowed = true
	} else if opts.AwInputsRegex.MatchString(expression) {
		// Check if this expression matches github.agentics.inputs.* pattern (shared workflow inputs)
		allowed = true
	} else if opts.EnvRegex.MatchString(expression) {
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
					if opts.NeedsStepsRegex.MatchString(property) {
						propertyAllowed = true
					} else if opts.InputsRegex.MatchString(property) {
						propertyAllowed = true
					} else if opts.WorkflowCallInputsRegex.MatchString(property) {
						propertyAllowed = true
					} else if opts.AwInputsRegex.MatchString(property) {
						propertyAllowed = true
					} else if opts.EnvRegex.MatchString(property) {
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
		*opts.UnauthorizedExpressions = append(*opts.UnauthorizedExpressions, expression)
	}

	return nil
}

// ValidateExpressionSafetyPublic is a public wrapper for validateExpressionSafety
// that allows testing expression validation from external packages
func ValidateExpressionSafetyPublic(markdownContent string) error {
	return validateExpressionSafety(markdownContent)
}
