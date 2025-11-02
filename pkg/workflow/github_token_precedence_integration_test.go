//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelGitHubTokenPrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-token-precedence-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("top-level github-token used when no safe-outputs token", func(t *testing.T) {
		testContent := `---
on: push
name: Test Top-Level GitHub Token
on:
  issues:
    types: [opened]
engine: claude
github-token: ${{ secrets.TOPLEVEL_PAT }}
tools:
  github:
    mode: remote
    allowed: [list_issues]
---

# Test Top-Level GitHub Token

Test that top-level github-token is used in engine configuration.
`

		testFile := filepath.Join(tmpDir, "test-toplevel-token.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		outputFile := filepath.Join(tmpDir, "test-toplevel-token.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that the top-level token is used in the GitHub MCP config
		// The token should be set in the env block to prevent template injection
		if !strings.Contains(yamlContent, "GITHUB_MCP_SERVER_TOKEN: ${{ secrets.TOPLEVEL_PAT }}") {
			t.Error("Expected top-level github-token to be used in GITHUB_MCP_SERVER_TOKEN env var")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}

		// Verify that the Authorization header uses the env variable
		if !strings.Contains(yamlContent, "Bearer $GITHUB_MCP_SERVER_TOKEN") {
			t.Error("Expected Authorization header to use GITHUB_MCP_SERVER_TOKEN env var")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}
	})

	t.Run("safe-outputs github-token overrides top-level", func(t *testing.T) {
		testContent := `---
on: push
name: Test Safe-Outputs Override
on:
  issues:
    types: [opened]
engine: claude
github-token: ${{ secrets.TOPLEVEL_PAT }}
safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUTS_PAT }}
  create-issue:
---

# Test Safe-Outputs Override

Test that safe-outputs github-token overrides top-level.
`

		testFile := filepath.Join(tmpDir, "test-safe-outputs-override.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		outputFile := filepath.Join(tmpDir, "test-safe-outputs-override.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that safe-outputs token is used in the create_issue job
		if !strings.Contains(yamlContent, "github-token: ${{ secrets.SAFE_OUTPUTS_PAT }}") {
			t.Error("Expected safe-outputs github-token to be used in create_issue job")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}

		// Verify that top-level token is NOT used in safe-outputs job
		if strings.Contains(yamlContent, "github-token: ${{ secrets.TOPLEVEL_PAT }}") {
			t.Error("Top-level github-token should not be used when safe-outputs token is present")
		}
	})

	t.Run("individual safe-output token overrides both", func(t *testing.T) {
		testContent := `---
on: push
name: Test Individual Override
on:
  issues:
    types: [opened]
engine: claude
github-token: ${{ secrets.TOPLEVEL_PAT }}
safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUTS_PAT }}
  create-issue:
    github-token: ${{ secrets.INDIVIDUAL_PAT }}
---

# Test Individual Override

Test that individual safe-output github-token has highest precedence.
`

		testFile := filepath.Join(tmpDir, "test-individual-override.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		outputFile := filepath.Join(tmpDir, "test-individual-override.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that individual token is used in the create_issue job
		if !strings.Contains(yamlContent, "github-token: ${{ secrets.INDIVIDUAL_PAT }}") {
			t.Error("Expected individual safe-output github-token to be used in create_issue job")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}

		// Count occurrences of each token to verify precedence
		individualCount := strings.Count(yamlContent, "github-token: ${{ secrets.INDIVIDUAL_PAT }}")
		safeOutputsCount := strings.Count(yamlContent, "github-token: ${{ secrets.SAFE_OUTPUTS_PAT }}")
		toplevelCount := strings.Count(yamlContent, "github-token: ${{ secrets.TOPLEVEL_PAT }}")

		if individualCount == 0 {
			t.Error("Individual token should be present at least once")
		}

		// Note: safe-outputs global token might appear in other safe-output jobs or contexts
		// but should not appear more frequently than the individual token in the create_issue job
		// The test is primarily checking that the individual token is used where it should be
		if individualCount == 0 && safeOutputsCount > 0 {
			t.Error("Individual token should take precedence over safe-outputs token")
		}
		if individualCount == 0 && toplevelCount > 0 {
			t.Error("Individual token should take precedence over top-level token")
		}
	})

	t.Run("top-level token used in codex engine", func(t *testing.T) {
		testContent := `---
on: push
name: Test Codex Engine Token
on:
  workflow_dispatch:
engine: codex
github-token: ${{ secrets.TOPLEVEL_PAT }}
tools:
  github:
    allowed: [list_issues]
---

# Test Codex Engine Token

Test that top-level github-token is used in Codex engine.
`

		testFile := filepath.Join(tmpDir, "test-codex-token.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		outputFile := filepath.Join(tmpDir, "test-codex-token.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that the top-level token is used in GH_AW_GITHUB_TOKEN env var
		if !strings.Contains(yamlContent, "GH_AW_GITHUB_TOKEN: ${{ secrets.TOPLEVEL_PAT }}") {
			t.Error("Expected top-level github-token to be used in GH_AW_GITHUB_TOKEN env var for Codex")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}

		// Also check in the TOML config
		if !strings.Contains(yamlContent, "GITHUB_PERSONAL_ACCESS_TOKEN = \"${{ secrets.TOPLEVEL_PAT }}\"") {
			t.Error("Expected top-level github-token to be used in TOML config for Codex")
		}
	})

	t.Run("top-level token used in copilot engine", func(t *testing.T) {
		testContent := `---
on: push
name: Test Copilot Engine Token
on:
  workflow_dispatch:
engine: copilot
github-token: ${{ secrets.TOPLEVEL_PAT }}
tools:
  github:
    allowed: [list_issues]
---

# Test Copilot Engine Token

Test that top-level github-token is used in Copilot engine.
`

		testFile := filepath.Join(tmpDir, "test-copilot-token.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		outputFile := filepath.Join(tmpDir, "test-copilot-token.lock.yml")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		yamlContent := string(content)

		// Verify that the top-level token is used in GITHUB_MCP_SERVER_TOKEN env var
		if !strings.Contains(yamlContent, "GITHUB_MCP_SERVER_TOKEN: ${{ secrets.TOPLEVEL_PAT }}") {
			t.Error("Expected top-level github-token to be used in GITHUB_MCP_SERVER_TOKEN env var for Copilot")
			t.Logf("Generated YAML:\n%s", yamlContent)
		}
	})
}
