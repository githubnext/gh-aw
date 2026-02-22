//go:build !integration

package workflow

import (
	"testing"
)

// BenchmarkValidateExpression benchmarks single expression validation
func BenchmarkValidateExpression(b *testing.B) {
	expression := "github.event.issue.number"
	unauthorizedExprs := []string{}

	for b.Loop() {
		_ = validateSingleExpression(expression, ExpressionValidationOptions{
			NeedsStepsRe:            needsStepsRegex,
			InputsRe:                inputsRegex,
			WorkflowCallInputsRe:    workflowCallInputsRegex,
			AwInputsRe:              awInputsRegex,
			EnvRe:                   envRegex,
			UnauthorizedExpressions: &unauthorizedExprs,
		})
	}
}

// BenchmarkValidateExpression_Complex benchmarks complex expression with comparisons
func BenchmarkValidateExpression_Complex(b *testing.B) {
	expression := "github.event.pull_request.number == github.event.issue.number"
	unauthorizedExprs := []string{}

	for b.Loop() {
		_ = validateSingleExpression(expression, ExpressionValidationOptions{
			NeedsStepsRe:            needsStepsRegex,
			InputsRe:                inputsRegex,
			WorkflowCallInputsRe:    workflowCallInputsRegex,
			AwInputsRe:              awInputsRegex,
			EnvRe:                   envRegex,
			UnauthorizedExpressions: &unauthorizedExprs,
		})
	}
}

// BenchmarkValidateExpression_NeedsOutputs benchmarks needs.*.outputs.* validation
func BenchmarkValidateExpression_NeedsOutputs(b *testing.B) {
	expression := "steps.sanitized.outputs.text"
	unauthorizedExprs := []string{}

	for b.Loop() {
		_ = validateSingleExpression(expression, ExpressionValidationOptions{
			NeedsStepsRe:            needsStepsRegex,
			InputsRe:                inputsRegex,
			WorkflowCallInputsRe:    workflowCallInputsRegex,
			AwInputsRe:              awInputsRegex,
			EnvRe:                   envRegex,
			UnauthorizedExpressions: &unauthorizedExprs,
		})
	}
}

// BenchmarkValidateExpression_StepsOutputs benchmarks steps.*.outputs.* validation
func BenchmarkValidateExpression_StepsOutputs(b *testing.B) {
	expression := "steps.my-step.outputs.result"
	unauthorizedExprs := []string{}

	for b.Loop() {
		_ = validateSingleExpression(expression, ExpressionValidationOptions{
			NeedsStepsRe:            needsStepsRegex,
			InputsRe:                inputsRegex,
			WorkflowCallInputsRe:    workflowCallInputsRegex,
			AwInputsRe:              awInputsRegex,
			EnvRe:                   envRegex,
			UnauthorizedExpressions: &unauthorizedExprs,
		})
	}
}

// BenchmarkValidateExpressionSafety benchmarks full markdown expression validation
func BenchmarkValidateExpressionSafety(b *testing.B) {
	markdown := `# Issue Analysis

Analyze issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

The issue content is: "${{ steps.sanitized.outputs.text }}"

The issue was created by ${{ github.actor }} with title: "${{ github.event.issue.title }}"

Repository: ${{ github.repository }}
Run ID: ${{ github.run_id }}
`

	for b.Loop() {
		_ = validateExpressionSafety(markdown)
	}
}

// BenchmarkValidateExpressionSafety_Complex benchmarks complex markdown with many expressions
func BenchmarkValidateExpressionSafety_Complex(b *testing.B) {
	markdown := `# Complex Workflow Analysis

## Issue Details
- Number: ${{ github.event.issue.number }}
- Title: ${{ github.event.issue.title }}
- Author: ${{ github.actor }}
- Repository: ${{ github.repository }}

## Pull Request Details
- Number: ${{ github.event.pull_request.number }}
- Head Branch: ${{ github.event.pull_request.head.ref }}
- Base Branch: ${{ github.event.pull_request.base.ref }}

## Workflow Context
- Run ID: ${{ github.run_id }}
- Run Number: ${{ github.run_number }}
- Workflow: ${{ github.workflow }}
- Job: ${{ github.job }}

## Previous Step Outputs
- Activation: ${{ steps.sanitized.outputs.text }}
- Analysis: ${{ steps.analyze.outputs.result }}
- Summary: ${{ steps.summarize.outputs.content }}

## Input Parameters
- Environment: ${{ github.event.inputs.environment }}
- Debug Mode: ${{ github.event.inputs.debug }}
- Target: ${{ github.event.inputs.target }}

## Env Variables
- Config: ${{ env.CONFIG_PATH }}
- Mode: ${{ env.DEPLOYMENT_MODE }}
`

	for b.Loop() {
		_ = validateExpressionSafety(markdown)
	}
}

// BenchmarkValidateExpressionSafety_Minimal benchmarks minimal markdown with few expressions
func BenchmarkValidateExpressionSafety_Minimal(b *testing.B) {
	markdown := `# Simple Task

Analyze issue #${{ github.event.issue.number }}.
`

	for b.Loop() {
		_ = validateExpressionSafety(markdown)
	}
}

// BenchmarkParseExpression_Simple benchmarks simple expression parsing
func BenchmarkParseExpression_Simple(b *testing.B) {
	expression := "github.event.issue.number"

	for b.Loop() {
		_, _ = ParseExpression(expression)
	}
}

// BenchmarkParseExpression_Comparison benchmarks comparison expression parsing
func BenchmarkParseExpression_Comparison(b *testing.B) {
	expression := "github.event.issue.number == 123"

	for b.Loop() {
		_, _ = ParseExpression(expression)
	}
}

// BenchmarkParseExpression_Logical benchmarks logical expression parsing
func BenchmarkParseExpression_Logical(b *testing.B) {
	expression := "github.event.issue.state == 'open' && github.event.issue.locked == false"

	for b.Loop() {
		_, _ = ParseExpression(expression)
	}
}

// BenchmarkParseExpression_ComplexNested benchmarks complex nested expression parsing
func BenchmarkParseExpression_ComplexNested(b *testing.B) {
	expression := "(github.event.issue.state == 'open' || github.event.pull_request.state == 'open') && !cancelled()"

	for b.Loop() {
		_, _ = ParseExpression(expression)
	}
}
