//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func promptCreationStepContent(compiled string) string {
	start := strings.Index(compiled, "- name: Create prompt with built-in context")
	if start < 0 {
		return ""
	}
	step := compiled[start:]
	if before, _, found := strings.Cut(step, "\n      - name:"); found {
		return before
	}
	return step
}

// TestJavaScriptPromptRendering verifies that prompt content is created and
// interpolated exclusively by JavaScript actions.
func TestJavaScriptPromptRendering(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "heredoc-interpolation-test")

	// Workflow with markdown content containing GitHub expressions
	// These should be extracted and replaced with ${GH_AW_...} references
	// Simple expressions like github.repository generate pretty names like GH_AW_GITHUB_REPOSITORY
	testContent := `---
on: issues
permissions:
  contents: read
engine: copilot
---

# Test Workflow with Expressions

Repository: ${{ github.repository }}
Actor: ${{ github.actor }}
`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := stringutil.MarkdownToLockFile(testFile)
	compiledYAML, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledYAML)
	promptStep := promptCreationStepContent(compiledStr)

	if strings.Contains(promptStep, "<<") || strings.Contains(promptStep, "create_prompt_first.sh") {
		t.Error("Prompt creation should not contain heredocs or shell helpers")
	}
	if !strings.Contains(promptStep, "create_prompt.cjs") {
		t.Error("Prompt creation should use create_prompt.cjs")
	}

	// Verify that the prompt content contains __GH_AW_...__ references
	// These are Handlebars-style placeholders interpolated by the github-script step
	// Simple expressions like github.repository generate pretty names like GH_AW_GITHUB_REPOSITORY
	if !strings.Contains(compiledStr, "__GH_AW_") {
		t.Error("Prompt content should contain __GH_AW_...__ references for JavaScript interpolation")
	}

	// Verify the original expressions have been replaced in the prompt content.
	if strings.Contains(compiledStr, "Repository: ${{ github.repository }}") {
		t.Error("Original GitHub expressions should be replaced with __GH_AW_...__ references in prompt heredoc")
	}
	if !strings.Contains(compiledStr, "__GH_AW_") {
		t.Error("Prompt content should contain __GH_AW_...__ references for JavaScript interpolation")
	}

	// Verify that the interpolation and template rendering step exists
	if !strings.Contains(compiledStr, "- name: Interpolate variables and render templates") {
		t.Error("Compiled workflow should contain interpolation and template rendering step")
	}

	// Verify that the step uses github-script
	if !strings.Contains(compiledStr, "uses: actions/github-script@") {
		t.Error("Interpolation and template rendering step should use actions/github-script")
	}

	// Verify environment variables are defined in the step
	// Simple expressions like github.repository generate pretty names like GH_AW_GITHUB_REPOSITORY
	if !strings.Contains(compiledStr, "GH_AW_GITHUB_") {
		t.Error("Interpolation and template rendering step should contain GH_AW_* environment variables")
	}
}

// TestJavaScriptPromptRenderingMainPrompt tests the main prompt renderer.
func TestJavaScriptPromptRenderingMainPrompt(t *testing.T) {
	tmpDir := testutil.TempDir(t, "heredoc-main-test")

	testContent := `---
on: issues
permissions:
  contents: read
engine: copilot
---

# Test Workflow

Repository: ${{ github.repository }}
Actor: ${{ github.actor }}
`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(testFile)
	compiledYAML, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiledYAML)
	promptStep := promptCreationStepContent(compiledStr)

	if strings.Contains(promptStep, "<<") {
		t.Error("Expected prompt creation without heredoc delimiters")
	}
	if !strings.Contains(promptStep, "create_prompt.cjs") {
		t.Error("Expected JavaScript prompt creation")
	}

	// Verify interpolation and template rendering step exists
	if !strings.Contains(compiledStr, "- name: Interpolate variables and render templates") {
		t.Error("Expected interpolation and template rendering step for JavaScript-based variable interpolation")
	}
}
