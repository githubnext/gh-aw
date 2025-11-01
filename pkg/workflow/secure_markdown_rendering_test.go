package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecureMarkdownRendering_Integration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "secure-markdown-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Simple workflow with GitHub expressions
	testContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
---

# Test Workflow

Repository: ${{ github.repository }}
Actor: ${{ github.actor }}
Run ID: ${{ github.run_id }}
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

	// Debug: print the compiled YAML section we care about
	lines := strings.Split(compiledStr, "\n")
	inPromptStep := false
	for i, line := range lines {
		if strings.Contains(line, "name: Create prompt") {
			inPromptStep = true
		}
		if inPromptStep {
			t.Logf("Line %d: %s", i, line)
			if i > 0 && strings.Contains(lines[i-1], "PROMPT_EOF") && strings.Contains(line, "name:") && !strings.Contains(line, "Create prompt") {
				break
			}
		}
	}

	// Verify that environment variables are defined for GitHub expressions
	if !strings.Contains(compiledStr, "GH_AW_EXPR_") {
		t.Error("Compiled workflow should contain GH_AW_EXPR_ environment variables")
	}

	// Verify that the original ${{ }} expressions are NOT in the heredoc content
	// They should be replaced with ${GH_AW_EXPR_...} references
	if strings.Contains(compiledStr, "Repository: ${{ github.repository }}") {
		t.Error("Original GitHub expressions should be replaced with environment variable references in the prompt content")
	}

	// Verify that environment variable references ARE in the heredoc content
	if !strings.Contains(compiledStr, "${GH_AW_EXPR_") {
		t.Error("Environment variable references should be in the prompt content")
	}

	// Verify environment variables are set with GitHub expressions
	if !strings.Contains(compiledStr, "GH_AW_EXPR_") || !strings.Contains(compiledStr, ": ${{ github.repository }}") {
		t.Error("Environment variables should be set to GitHub expression values")
	}
}
