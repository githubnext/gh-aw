package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHeredocInterpolation verifies that PROMPT_EOF heredoc delimiter is quoted
// to prevent bash variable interpolation. Variables are interpolated using github-script instead.
func TestHeredocInterpolation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "heredoc-interpolation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Workflow with markdown content containing GitHub expressions
	// These should be extracted and replaced with ${GH_AW_EXPR_...} references
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

	// Verify that heredoc delimiters ARE quoted (should be 'PROMPT_EOF' not PROMPT_EOF)
	// This prevents shell variable interpolation
	if !strings.Contains(compiledStr, "<< 'PROMPT_EOF'") {
		t.Error("PROMPT_EOF delimiter should be quoted to prevent shell variable interpolation")

		// Show the problematic lines
		lines := strings.Split(compiledStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "<< PROMPT_EOF") && !strings.Contains(line, "'PROMPT_EOF'") {
				t.Logf("Line %d with unquoted delimiter: %s", i, line)
			}
		}
	}

	// Verify that the prompt content contains ${GH_AW_EXPR_...} references
	// These will be interpolated by the github-script step, not by bash
	if !strings.Contains(compiledStr, "${GH_AW_EXPR_") {
		t.Error("Prompt content should contain ${GH_AW_EXPR_...} references for JavaScript interpolation")
	}

	// Verify the original expressions are NOT in the prompt content (they've been replaced)
	if strings.Contains(compiledStr, "Repository: ${{ github.repository }}") {
		t.Error("Original GitHub expressions should be replaced with ${GH_AW_EXPR_...} references in prompt")
	}

	// Verify that the interpolation step exists
	if !strings.Contains(compiledStr, "- name: Interpolate variables in prompt") {
		t.Error("Compiled workflow should contain interpolation step")
	}

	// Verify that the interpolation step uses github-script
	if !strings.Contains(compiledStr, "uses: actions/github-script@") {
		t.Error("Interpolation step should use actions/github-script")
	}

	// Verify environment variables are defined in the interpolation step
	if !strings.Contains(compiledStr, "GH_AW_EXPR_") {
		t.Error("Interpolation step should contain GH_AW_EXPR_ environment variables")
	}
}

// TestHeredocInterpolationMainPrompt tests that the main prompt content uses quoted delimiter
func TestHeredocInterpolationMainPrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heredoc-main-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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

	// All heredoc delimiters should be quoted to prevent shell expansion
	quotedCount := strings.Count(compiledStr, "<< 'PROMPT_EOF'")
	if quotedCount == 0 {
		t.Error("Expected quoted PROMPT_EOF delimiters to prevent shell variable interpolation")
	}

	// Verify interpolation step exists
	if !strings.Contains(compiledStr, "- name: Interpolate variables in prompt") {
		t.Error("Expected interpolation step for JavaScript-based variable interpolation")
	}
}

