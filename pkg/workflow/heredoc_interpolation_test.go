package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHeredocInterpolation verifies that PROMPT_EOF heredoc delimiter is unquoted
// to allow bash variable interpolation of GH_AW_EXPR_* environment variables
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

	// Verify that heredoc delimiters are NOT quoted (should be PROMPT_EOF not 'PROMPT_EOF')
	if strings.Contains(compiledStr, "<< 'PROMPT_EOF'") {
		t.Error("PROMPT_EOF delimiter should NOT be quoted - this prevents variable interpolation")
		
		// Show the problematic lines
		lines := strings.Split(compiledStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "<< 'PROMPT_EOF'") {
				t.Logf("Line %d with quoted delimiter: %s", i, line)
			}
		}
	}

	// Verify that heredoc delimiters are unquoted (should be PROMPT_EOF)
	if !strings.Contains(compiledStr, "<< PROMPT_EOF") {
		t.Error("PROMPT_EOF delimiter should be unquoted to allow variable interpolation")
	}

	// Verify environment variables are defined for GitHub expressions
	if !strings.Contains(compiledStr, "GH_AW_EXPR_") {
		t.Error("Compiled workflow should contain GH_AW_EXPR_ environment variables")
	}

	// Verify that the prompt content contains ${GH_AW_EXPR_...} references
	// This proves that the unquoted delimiter will allow bash to interpolate them
	if !strings.Contains(compiledStr, "${GH_AW_EXPR_") {
		t.Error("Prompt content should contain ${GH_AW_EXPR_...} references that bash can interpolate")
	}

	// Verify the original expressions are NOT in the prompt content (they've been replaced)
	if strings.Contains(compiledStr, "Repository: ${{ github.repository }}") {
		t.Error("Original GitHub expressions should be replaced with ${GH_AW_EXPR_...} references in prompt")
	}
}

// TestHeredocInterpolationMainPrompt tests that the main prompt content also uses unquoted delimiter
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

	// All heredoc delimiters should be unquoted
	quotedCount := strings.Count(compiledStr, "<< 'PROMPT_EOF'")
	if quotedCount > 0 {
		t.Errorf("Found %d quoted PROMPT_EOF delimiters, expected 0", quotedCount)
	}

	// Should have unquoted delimiters
	unquotedCount := strings.Count(compiledStr, "<< PROMPT_EOF")
	if unquotedCount == 0 {
		t.Error("Expected unquoted PROMPT_EOF delimiters for variable interpolation")
	}
}
