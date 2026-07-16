// This file provides centralized regex patterns for GitHub Actions expression matching.
//
// Core language-level patterns and predicate helpers are defined in pkg/ghexpr.
// This file re-exports them as package-level vars for backward compatibility and
// adds gh-aw–specific context patterns (secrets, inputs, aw-inputs, etc.).
//
// # Available Pattern Categories
//
// ## Core Expression Patterns (from pkg/ghexpr)
//   - ExpressionPattern         - Matches GitHub Actions expressions: ${{ ... }}
//   - ExpressionPatternDotAll   - Matches expressions with dotall mode (multiline)
//   - InlineExpressionPattern   - Matches inline ${{ ... }} in template strings
//   - TemplateIfPattern         - Matches {{#if ...}} template conditionals
//   - TemplateElseIfPattern     - Matches {{#elseif ...}} template conditionals
//   - StringLiteralPattern      - Matches string literals ('...', "...", `...`)
//   - NumberLiteralPattern      - Matches numeric literals
//   - OrPattern                 - Splits logical-OR expressions
//   - ComparisonExtractionPattern - Extracts LHS of comparisons
//   - RangePattern              - Matches numeric ranges (e.g., "1-10")
//
// ## Predicate helpers (from pkg/ghexpr)
//   - hasExpressionMarker  - reports ${{ present
//   - containsExpression   - reports complete ${{ ... }} present
//   - isExpression         - reports whole string is ${{ ... }}
//
// ## gh-aw Context Access Patterns
//   - NeedsStepsPattern           - Matches needs.* and steps.* patterns
//   - InputsPattern               - Matches github.event.inputs.* patterns
//   - WorkflowCallInputsPattern   - Matches inputs.* patterns (workflow_call)
//   - AWInputsPattern             - Matches github.aw.inputs.* patterns
//   - AWInputsExpressionPattern   - Matches full ${{ github.aw.inputs.* }} expressions
//   - AWImportInputsPattern       - Matches github.aw.import-inputs.* patterns
//   - AWImportInputsExpressionPattern - Matches full ${{ github.aw.import-inputs.* }} expressions
//   - EnvPattern                  - Matches env.* patterns
//
// ## Secret Patterns
//   - SecretExpressionPattern  - Matches ${{ secrets.SECRET_NAME }} expressions
//   - SecretsExpressionPattern - Validates secrets expression syntax
//
// ## Template Patterns (gh-aw specific)
//   - UnsafeContextPattern - Matches potentially unsafe context patterns

package workflow

import (
	"regexp"

	"github.com/github/gh-aw/pkg/ghexpr"
)

// hasExpressionMarker re-exports [ghexpr.HasExpressionMarker] with the unexported
// name used throughout pkg/workflow.
func hasExpressionMarker(s string) bool { return ghexpr.HasExpressionMarker(s) }

// containsExpression re-exports [ghexpr.ContainsExpression].
func containsExpression(s string) bool { return ghexpr.ContainsExpression(s) }

// isExpression re-exports [ghexpr.IsExpression].
func isExpression(s string) bool { return ghexpr.IsExpression(s) }

// Core Expression Patterns — sourced from pkg/ghexpr.
var (
	ExpressionPattern           = ghexpr.ExpressionPattern
	ExpressionPatternDotAll     = ghexpr.ExpressionPatternDotAll
	InlineExpressionPattern     = ghexpr.InlineExpressionPattern
	TemplateIfPattern           = ghexpr.TemplateIfPattern
	TemplateElseIfPattern       = ghexpr.TemplateElseIfPattern
	ComparisonExtractionPattern = ghexpr.ComparisonExtractionPattern
	OrPattern                   = ghexpr.OrPattern
	StringLiteralPattern        = ghexpr.StringLiteralPattern
	NumberLiteralPattern        = ghexpr.NumberLiteralPattern
	RangePattern                = ghexpr.RangePattern
)

// gh-aw Context Access Patterns
var (
	// NeedsStepsPattern matches needs.* and steps.* context patterns
	// Example: needs.build.outputs.version, steps.setup.outputs.path
	NeedsStepsPattern = regexp.MustCompile(`^(needs|steps)\.[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)

	// InputsPattern matches github.event.inputs.* patterns
	// Example: github.event.inputs.workflow_id
	InputsPattern = regexp.MustCompile(`^github\.event\.inputs\.[a-zA-Z0-9_-]+$`)

	// WorkflowCallInputsPattern matches inputs.* patterns for workflow_call
	// Example: inputs.branch_name
	WorkflowCallInputsPattern = regexp.MustCompile(`^inputs\.[a-zA-Z0-9_-]+$`)

	// AWInputsPattern matches github.aw.inputs.* patterns
	// Example: github.aw.inputs.custom_param
	AWInputsPattern = regexp.MustCompile(`^github\.aw\.inputs\.[a-zA-Z0-9_-]+$`)

	// AWInputsExpressionPattern matches full ${{ github.aw.inputs.* }} expressions
	// Used for extraction rather than validation
	AWInputsExpressionPattern = regexp.MustCompile(`\$\{\{\s*github\.aw\.inputs\.([a-zA-Z0-9_-]+)\s*\}\}`)

	// AWImportInputsPattern matches github.aw.import-inputs.* patterns for import-schema form.
	// Supports both scalar inputs and one-level deep object sub-keys:
	//   github.aw.import-inputs.count
	//   github.aw.import-inputs.config.apiKey
	AWImportInputsPattern = regexp.MustCompile(`^github\.aw\.import-inputs\.[a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)?$`)

	// AWImportInputsExpressionPattern matches full ${{ github.aw.import-inputs.* }} expressions.
	// Captures the full dotted path after "import-inputs." (e.g. "count" or "config.apiKey").
	// Used for substitution of values provided via the 'with' key in import specifications.
	AWImportInputsExpressionPattern = regexp.MustCompile(`\$\{\{\s*github\.aw\.import-inputs\.([a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)?)\s*\}\}`)

	// EnvPattern matches env.* patterns
	// Example: env.NODE_VERSION
	EnvPattern = regexp.MustCompile(`^env\.[a-zA-Z0-9_-]+$`)
)

// Secret Patterns
var (
	// SecretExpressionPattern matches ${{ secrets.SECRET_NAME }} expressions
	// Captures the secret name and supports optional || fallback
	SecretExpressionPattern = regexp.MustCompile(`\$\{\{\s*secrets\.([A-Z_][A-Z0-9_]*)\s*(?:\|\|.*?)?\s*\}\}`)

	// SecretsExpressionPattern validates complete secrets expression syntax
	// Supports chained || fallbacks: ${{ secrets.A || secrets.B }}
	SecretsExpressionPattern = regexp.MustCompile(`^\$\{\{\s*secrets\.[A-Za-z_][A-Za-z0-9_]*(\s*\|\|\s*secrets\.[A-Za-z_][A-Za-z0-9_]*)*\s*\}\}$`)
)

// Template Patterns (gh-aw specific)
var (
	// UnsafeContextPattern matches potentially unsafe context patterns.
	// These patterns may allow injection attacks in templates.
	UnsafeContextPattern = regexp.MustCompile(`\$\{\{\s*(github\.event\.|steps\.[^}]+\.outputs\.|inputs\.)[^}]+\}\}`)
)
