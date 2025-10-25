package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateRenderingStep(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "template-rendering-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with conditional blocks that use GitHub expressions
	testContent := `---
on: issues
permissions:
  contents: read
tools:
  github:
    allowed: [list_issues]
engine: claude
---

# Test Template Rendering

{{#if github.event.issue.number}}
This section should be shown if there's an issue number.
{{/if}}

{{#if github.actor}}
This section should be shown if there's an actor.
{{/if}}

{{#if true}}
This section should be kept (literal true).
{{/if}}

{{#if false}}
This section should be removed (literal false).
{{/if}}

Normal content here.
`

	testFile := filepath.Join(tmpDir, "test-template-rendering.md")
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

	// Verify the template rendering step is present
	if !strings.Contains(compiledStr, "- name: Render template conditionals") {
		t.Error("Compiled workflow should contain template rendering step")
	}

	if !strings.Contains(compiledStr, "uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd") {
		t.Error("Template rendering step should use github-script action")
	}

	// Verify that GitHub expressions are wrapped in ${{ }}
	if !strings.Contains(compiledStr, "{{#if ${{ github.event.issue.number }} }}") {
		t.Error("Compiled workflow should contain wrapped github.event.issue.number expression")
	}

	if !strings.Contains(compiledStr, "{{#if ${{ github.actor }} }}") {
		t.Error("Compiled workflow should contain wrapped github.actor expression")
	}

	// Verify that literal values are also wrapped (simplified behavior)
	if !strings.Contains(compiledStr, "{{#if ${{ true }} }}") {
		t.Error("Compiled workflow should contain wrapped literal true")
	}

	if !strings.Contains(compiledStr, "{{#if ${{ false }} }}") {
		t.Error("Compiled workflow should contain wrapped literal false")
	}

	// Verify the render function is present
	if !strings.Contains(compiledStr, "function renderMarkdownTemplate") {
		t.Error("Template rendering step should contain renderMarkdownTemplate function")
	}

	if !strings.Contains(compiledStr, "function isTruthy") {
		t.Error("Template rendering step should contain isTruthy function")
	}
}

func TestTemplateRenderingStepSkipped(t *testing.T) {
	// NOTE: This test is now less relevant because GitHub tools are added by default,
	// which means GitHub context (with template conditionals) is always added.
	// However, we keep this test to verify that template rendering behaves correctly
	// even when the user's markdown doesn't have conditionals.

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "template-rendering-skip-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case WITHOUT conditional blocks in user's markdown
	// Note: GitHub tools are added by default, so GitHub context will still be added
	testContent := `---
on: issues
permissions:
  contents: read
tools:
  edit:
  web-fetch:
engine: claude
---

# Test Without Template

Normal content without conditionals.
`

	testFile := filepath.Join(tmpDir, "test-no-template.md")
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

	// Verify the template rendering step IS present (because GitHub tool is added by default)
	if !strings.Contains(compiledStr, "- name: Render template conditionals") {
		t.Error("Compiled workflow should contain template rendering step because GitHub tool is added by default")
	}

	// Verify the GitHub context was added
	if !strings.Contains(compiledStr, "- name: Append GitHub context to prompt") {
		t.Error("Compiled workflow should contain GitHub context step because GitHub tool is added by default")
	}
}

func TestTemplateRenderingStepWithGitHubTool(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "template-rendering-github-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case WITHOUT conditional blocks in markdown but WITH GitHub tool
	testContent := `---
on: issues
permissions:
  contents: read
tools:
  github:
    allowed: [list_issues]
engine: claude
---

# Test With GitHub Tool

Normal content without conditionals in markdown.
`

	testFile := filepath.Join(tmpDir, "test-github-tool.md")
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

	// Verify the template rendering step IS present (because GitHub tool adds conditionals)
	if !strings.Contains(compiledStr, "- name: Render template conditionals") {
		t.Error("Compiled workflow should contain template rendering step when GitHub tool is enabled")
	}

	// Verify the GitHub context was added
	if !strings.Contains(compiledStr, "- name: Append GitHub context to prompt") {
		t.Error("Compiled workflow should contain GitHub context step when GitHub tool is enabled")
	}
}
