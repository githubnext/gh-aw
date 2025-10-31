package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplateExpressionWrappingIntegration verifies end-to-end compilation
// with template expressions that should be wrapped
func TestTemplateExpressionWrappingIntegration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "template-expression-integration")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Real-world example workflow with template conditionals
	testContent := `---
on:
  issues:
    types: [opened, edited]
  pull_request:
    types: [opened, edited]
permissions:
  contents: read
  issues: write
  pull-requests: read
engine: claude
---

# Issue and PR Analyzer

Analyze the issue or pull request and provide insights.

{{#if github.event.issue.number}}
## Issue Analysis

You are analyzing issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

The issue was created by ${{ github.actor }}.
{{/if}}

{{#if github.event.pull_request.number}}
## Pull Request Analysis

You are analyzing PR #${{ github.event.pull_request.number }} in repository ${{ github.repository }}.

The PR was created by ${{ github.actor }}.
{{/if}}

{{#if needs.activation.outputs.text}}
## Content

${{ needs.activation.outputs.text }}
{{/if}}

## Instructions

1. Review the content above
2. Provide actionable feedback
3. Create a summary comment
`

	testFile := filepath.Join(tmpDir, "analyzer.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	compiledYAML, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledYAML)

	// Verify that template conditionals are present
	if !strings.Contains(compiledStr, "- name: Render template conditionals") {
		t.Error("Compiled workflow should contain template rendering step")
	}

	// Verify GitHub expressions are properly wrapped in template conditionals
	expectedWrappedExpressions := []string{
		"{{#if ${{ github.event.issue.number }} }}",
		"{{#if ${{ github.event.pull_request.number }} }}",
		"{{#if ${{ needs.activation.outputs.text }} }}",
	}

	for _, expectedExpr := range expectedWrappedExpressions {
		if !strings.Contains(compiledStr, expectedExpr) {
			t.Errorf("Compiled workflow should contain wrapped expression: %s", expectedExpr)
		}
	}

	// Verify that expressions OUTSIDE template conditionals are NOT double-wrapped
	// These should remain as ${{ github.event.issue.number }} (not wrapped again)
	if strings.Contains(compiledStr, "${{ ${{ github.event.issue.number }}") {
		t.Error("Expressions outside template conditionals should not be double-wrapped")
	}

	// Verify that GitHub context expressions in prompt content are sanitized to shell variables
	if !strings.Contains(compiledStr, "issue #${GH_EVENT_ISSUE_NUMBER}") {
		t.Error("GitHub expressions in prompt content should be sanitized to shell variables")
	}

	// Verify that the environment variables are defined
	if !strings.Contains(compiledStr, "GH_EVENT_ISSUE_NUMBER: ${{ github.event.issue.number }}") {
		t.Error("Environment variables should be defined for sanitized GitHub expressions")
	}

	// Verify that repository expression is also sanitized
	if !strings.Contains(compiledStr, "in repository ${GH_REPOSITORY}") {
		t.Error("GitHub repository expression should be sanitized to shell variable")
	}

	// Verify that github.actor is sanitized
	if !strings.Contains(compiledStr, "created by ${GH_ACTOR}") {
		t.Error("GitHub actor expression should be sanitized to shell variable")
	}
}

// TestTemplateExpressionAlreadyWrapped verifies that already-wrapped expressions
// are not double-wrapped
func TestTemplateExpressionAlreadyWrapped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "template-already-wrapped")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Workflow with pre-wrapped expressions
	testContent := `---
on: issues
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test Already Wrapped

{{#if ${{ github.event.issue.number }} }}
This expression is already wrapped.
{{/if}}

{{#if github.actor}}
This expression needs wrapping.
{{/if}}
`

	testFile := filepath.Join(tmpDir, "already-wrapped.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	compiledYAML, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledYAML)

	// Verify already-wrapped expression is not double-wrapped
	if !strings.Contains(compiledStr, "{{#if ${{ github.event.issue.number }} }}") {
		t.Error("Already-wrapped expression should be preserved")
	}

	// Verify it's not double-wrapped
	if strings.Contains(compiledStr, "${{ ${{ github.event.issue.number }}") {
		t.Error("Already-wrapped expression should not be double-wrapped")
	}

	// Verify unwrapped expression is wrapped
	if !strings.Contains(compiledStr, "{{#if ${{ github.actor }} }}") {
		t.Error("Unwrapped expression should be wrapped")
	}
}

// TestTemplateWithMixedExpressionsAndLiterals verifies correct handling
// of template conditionals with both GitHub expressions and literal values
func TestTemplateWithMixedExpressionsAndLiterals(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "template-mixed")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
on: issues
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Mixed Template Test

{{#if github.event.issue.number}}
GitHub expression - will be wrapped.
{{/if}}

{{#if true}}
Literal true - will also be wrapped.
{{/if}}

{{#if false}}
Literal false - will also be wrapped.
{{/if}}

{{#if some_variable}}
Unknown variable - will also be wrapped.
{{/if}}

{{#if steps.my_step.outputs.value}}
Steps expression - will be wrapped.
{{/if}}
`

	testFile := filepath.Join(tmpDir, "mixed.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	compiledYAML, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledYAML)

	// Verify all expressions are wrapped (simplified behavior)
	if !strings.Contains(compiledStr, "{{#if ${{ github.event.issue.number }} }}") {
		t.Error("GitHub expression should be wrapped")
	}

	if !strings.Contains(compiledStr, "{{#if ${{ steps.my_step.outputs.value }} }}") {
		t.Error("Steps expression should be wrapped")
	}

	if !strings.Contains(compiledStr, "{{#if ${{ true }} }}") {
		t.Error("Literal 'true' should be wrapped")
	}

	if !strings.Contains(compiledStr, "{{#if ${{ false }} }}") {
		t.Error("Literal 'false' should be wrapped")
	}

	if !strings.Contains(compiledStr, "{{#if ${{ some_variable }} }}") {
		t.Error("Unknown variable should be wrapped")
	}

	// Make sure we didn't create invalid double-wrapping
	if strings.Contains(compiledStr, "${{ ${{") {
		t.Error("Should not double-wrap expressions")
	}
}
