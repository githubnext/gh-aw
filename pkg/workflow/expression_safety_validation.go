// This file provides GitHub Actions expression security validation.
// It enforces an allowlist of approved expressions to prevent injection attacks.
// For syntax helpers, see expression_syntax_validation.go.
// For runtime-import validation, see runtime_import_validation.go.

package workflow

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var expressionValidationLog = logger.New("workflow:expression_safety_validation")

// maxFuzzyMatchSuggestions is the maximum number of similar expressions to suggest
// when an unauthorized expression is found
const maxFuzzyMatchSuggestions = 7

// Pre-compiled regexes for expression safety validation (performance optimization)
var (
	// comparisonExpressionPattern matches a full comparison expression so both sides can be
	// validated recursively instead of allowing a safe-looking prefix to bypass validation.
	comparisonExpressionPattern = regexp.MustCompile(`^(.+?)\s*(?:==|!=|<|>|<=|>=)\s*(.+)$`)
	// orExpressionPattern matches "left || right" for fallback literal/expression checking
	orExpressionPattern = regexp.MustCompile(`^(.+?)\s*\|\|\s*(.+)$`)
)

// validateExpressionSafety checks that all GitHub Actions expressions in the markdown content
// are in the allowed list and returns an error if any unauthorized expressions are found
func validateExpressionSafety(markdownContent string) error {
	expressionValidationLog.Print("Validating expression safety in markdown content")
	matches := ExpressionPatternDotAll.FindAllStringSubmatch(markdownContent, -1)
	expressionValidationLog.Printf("Found %d expressions to validate", len(matches))

	var unauthorizedExpressions []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if err := validateExpressionMatch(match[1], &unauthorizedExpressions); err != nil {
			return err
		}
	}
	if len(unauthorizedExpressions) > 0 {
		return expressionSafetyError(unauthorizedExpressions)
	}
	expressionValidationLog.Print("Expression safety validation passed")
	return nil
}

func validateExpressionMatch(rawExpression string, unauthorizedExpressions *[]string) error {
	expression := strings.TrimSpace(rawExpression)
	if strings.Contains(rawExpression, "\n") {
		*unauthorizedExpressions = append(*unauthorizedExpressions, expression)
		return nil
	}
	opts := defaultExpressionValidationOptions(unauthorizedExpressions)
	parsed, parseErr := ParseExpression(expression)
	if parseErr != nil {
		return validateSingleExpression(expression, opts)
	}
	return VisitExpressionTree(parsed, func(expr *ExpressionNode) error {
		return validateSingleExpression(expr.Expression, opts)
	})
}

func defaultExpressionValidationOptions(unauthorizedExpressions *[]string) ExpressionValidationOptions {
	return ExpressionValidationOptions{
		NeedsStepsRe:            NeedsStepsPattern,
		InputsRe:                InputsPattern,
		WorkflowCallInputsRe:    WorkflowCallInputsPattern,
		AwInputsRe:              AWInputsPattern,
		AwImportInputsRe:        AWImportInputsPattern,
		EnvRe:                   EnvPattern,
		UnauthorizedExpressions: unauthorizedExpressions,
	}
}

func expressionSafetyError(unauthorizedExpressions []string) error {
	expressionValidationLog.Printf("Expression safety validation failed: %d unauthorized expressions found", len(unauthorizedExpressions))
	return NewValidationError(
		"expressions",
		fmt.Sprintf("%d unauthorized expressions found", len(unauthorizedExpressions)),
		"expressions are not in the allowed list:"+unauthorizedExpressionList(unauthorizedExpressions),
		fmt.Sprintf("Use only allowed expressions:%s\nFor more details, see the expression security documentation.", allowedExpressionList()),
	)
}

func unauthorizedExpressionList(unauthorizedExpressions []string) string {
	var unauthorizedList strings.Builder
	unauthorizedList.WriteString("\n")
	for _, expr := range unauthorizedExpressions {
		unauthorizedList.WriteString("  - ")
		unauthorizedList.WriteString(expr)
		closestMatches := parser.FindClosestMatches(expr, constants.AllowedExpressions, maxFuzzyMatchSuggestions)
		if len(closestMatches) > 0 {
			unauthorizedList.WriteString(" (did you mean: ")
			unauthorizedList.WriteString(strings.Join(closestMatches, ", "))
			unauthorizedList.WriteString("?)")
		}
		unauthorizedList.WriteString("\n")
	}
	return unauthorizedList.String()
}

func allowedExpressionList() string {
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
	allowedList.WriteString("  - github.aw.import-inputs.* (import-schema inputs)\n")
	allowedList.WriteString("  - inputs.* (workflow_call)\n")
	allowedList.WriteString("  - env.*\n")
	return allowedList.String()
}

// ExpressionValidationOptions contains the options for validating a single expression
type ExpressionValidationOptions struct {
	NeedsStepsRe            *regexp.Regexp
	InputsRe                *regexp.Regexp
	WorkflowCallInputsRe    *regexp.Regexp
	AwInputsRe              *regexp.Regexp
	AwImportInputsRe        *regexp.Regexp
	EnvRe                   *regexp.Regexp
	UnauthorizedExpressions *[]string
}

// validateExpressionForDangerousProps checks if an expression contains dangerous JavaScript
// property names that could be used for prototype pollution or traversal attacks.
// This matches the JavaScript runtime validation in actions/setup/js/runtime_import.cjs
// Returns an error if dangerous properties are found.
func validateExpressionForDangerousProps(expression string) error {
	expressionValidationLog.Printf("Checking expression for dangerous properties: %s", expression)
	trimmed := strings.TrimSpace(expression)

	// Split expression into parts using both dot and bracket notation;
	// filter out numeric indices (e.g., "0" in "assets[0]")
	parts := exprPartSplitRe.Split(trimmed, -1)

	for _, part := range parts {
		if part == "" || exprNumericPartRe.MatchString(part) {
			continue
		}

		if _, isDangerous := constants.DangerousPropertyNamesSet[part]; isDangerous {
			return NewValidationError(
				"expressions",
				fmt.Sprintf("dangerous property name %q found in expression", part),
				fmt.Sprintf("expression %q contains the dangerous property name %q", expression, part),
				fmt.Sprintf("Remove the dangerous property %q from the expression. Property names like constructor, __proto__, prototype, and similar JavaScript built-ins are blocked to prevent prototype pollution attacks. See PR #14826 for more details.", part),
			)
		}
	}

	return nil
}

// validateSingleExpression validates a single literal expression
func validateSingleExpression(expression string, opts ExpressionValidationOptions) error {
	expression = strings.TrimSpace(expression)
	if isSafeExpressionLiteral(expression) {
		return nil
	}
	if err := validateExpressionForDangerousProps(expression); err != nil {
		return err
	}
	allowed := isDirectlyAllowedExpression(expression, opts)
	if !allowed {
		allowed = isAllowedOrExpression(expression, opts)
	}
	if !allowed {
		allowed = isAllowedComparisonExpression(expression, opts)
	}
	if !allowed {
		*opts.UnauthorizedExpressions = append(*opts.UnauthorizedExpressions, expression)
	}
	return nil
}

func isSafeExpressionLiteral(expression string) bool {
	return stringLiteralRegex.MatchString(expression) || numberLiteralRegex.MatchString(expression) ||
		expression == "true" || expression == "false"
}

func isDirectlyAllowedExpression(expression string, opts ExpressionValidationOptions) bool {
	switch {
	case opts.NeedsStepsRe.MatchString(expression):
		return true
	case opts.InputsRe.MatchString(expression):
		return true
	case opts.WorkflowCallInputsRe.MatchString(expression):
		return true
	case opts.AwInputsRe.MatchString(expression):
		return true
	case opts.AwImportInputsRe != nil && opts.AwImportInputsRe.MatchString(expression):
		return true
	case opts.EnvRe.MatchString(expression):
		return true
	default:
		_, ok := constants.AllowedExpressionsSet[expression]
		return ok
	}
}

func isAllowedOrExpression(expression string, opts ExpressionValidationOptions) bool {
	orMatch := orExpressionPattern.FindStringSubmatch(expression)
	if len(orMatch) <= 2 {
		return false
	}
	leftExpr := strings.TrimSpace(orMatch[1])
	rightExpr := strings.TrimSpace(orMatch[2])
	leftErr := validateSingleExpression(leftExpr, opts)
	leftIsSafe := leftErr == nil && !containsExpressionInList(opts.UnauthorizedExpressions, leftExpr)
	if !leftIsSafe {
		return false
	}
	if isSafeExpressionLiteral(rightExpr) {
		return true
	}
	rightErr := validateSingleExpression(rightExpr, opts)
	return rightErr == nil && !containsExpressionInList(opts.UnauthorizedExpressions, rightExpr)
}

func isAllowedComparisonExpression(expression string, opts ExpressionValidationOptions) bool {
	comparisonMatch := comparisonExpressionPattern.FindStringSubmatch(expression)
	if len(comparisonMatch) <= 2 {
		return false
	}
	leftExpr := strings.TrimSpace(comparisonMatch[1])
	rightExpr := strings.TrimSpace(comparisonMatch[2])
	leftErr := validateSingleExpression(leftExpr, opts)
	rightErr := validateSingleExpression(rightExpr, opts)
	return leftExpr != "" && rightExpr != "" &&
		leftErr == nil && !containsExpressionInList(opts.UnauthorizedExpressions, leftExpr) &&
		rightErr == nil && !containsExpressionInList(opts.UnauthorizedExpressions, rightExpr)
}

// containsExpressionInList checks if an expression is in the list.
func containsExpressionInList(list *[]string, expr string) bool {
	return slices.Contains(*list, expr)
}
