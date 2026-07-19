//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubMCPToolsPromptBoundsCodeScanningAlertQueries verifies that the
// github_mcp_tools_prompt.md file includes the required guardrail instructing
// agents to always bound list_code_scanning_alerts calls with state: open and
// severity: critical,high. Unbounded queries produce oversized responses that
// break downstream workflow runs.
func TestGitHubMCPToolsPromptBoundsCodeScanningAlertQueries(t *testing.T) {
	promptPath := filepath.Join("..", "..", "actions", "setup", "md", "github_mcp_tools_prompt.md")
	data, err := os.ReadFile(promptPath)
	require.NoError(t, err, "should be able to read github_mcp_tools_prompt.md")

	content := string(data)

	assert.Contains(t, content, "list_code_scanning_alerts",
		"prompt should mention list_code_scanning_alerts to set agent expectations")
	assert.Contains(t, content, "state: open",
		"prompt should require state: open to bound list_code_scanning_alerts queries")
	assert.Contains(t, content, "severity: critical,high",
		"prompt should require severity: critical,high to bound list_code_scanning_alerts queries")
}

// TestGitHubMCPToolsWithSafeOutputsPromptBoundsCodeScanningAlertQueries verifies that
// the github_mcp_tools_with_safeoutputs_prompt.md file includes the same guardrail.
func TestGitHubMCPToolsWithSafeOutputsPromptBoundsCodeScanningAlertQueries(t *testing.T) {
	promptPath := filepath.Join("..", "..", "actions", "setup", "md", "github_mcp_tools_with_safeoutputs_prompt.md")
	data, err := os.ReadFile(promptPath)
	require.NoError(t, err, "should be able to read github_mcp_tools_with_safeoutputs_prompt.md")

	content := string(data)

	assert.Contains(t, content, "list_code_scanning_alerts",
		"prompt should mention list_code_scanning_alerts to set agent expectations")
	assert.Contains(t, content, "state: open",
		"prompt should require state: open to bound list_code_scanning_alerts queries")
	assert.Contains(t, content, "severity: critical,high",
		"prompt should require severity: critical,high to bound list_code_scanning_alerts queries")
}

// TestGitHubMCPToolsPromptHasFieldSelectionGuidance verifies that both prompt files
// contain guidance about the fields parameter introduced in GitHub MCP server 1.6.0.
// Field selection allows agents to request only the fields they need, significantly
// reducing response size for list/search tools.
func TestGitHubMCPToolsPromptHasFieldSelectionGuidance(t *testing.T) {
	promptFiles := []string{
		filepath.Join("..", "..", "actions", "setup", "md", "github_mcp_tools_prompt.md"),
		filepath.Join("..", "..", "actions", "setup", "md", "github_mcp_tools_with_safeoutputs_prompt.md"),
	}

	for _, promptPath := range promptFiles {
		t.Run(filepath.Base(promptPath), func(t *testing.T) {
			data, err := os.ReadFile(promptPath)
			require.NoError(t, err, "should be able to read %s", promptPath)

			content := string(data)

			assert.Contains(t, content, "fields",
				"prompt should mention fields parameter for response size reduction")
			assert.Contains(t, content, "list_pull_requests",
				"prompt should reference list_pull_requests as a tool that supports fields")
			assert.Contains(t, content, "1.6.0",
				"prompt should reference the GitHub MCP server version that introduced fields")
		})
	}
}

// TestGitHubMCPToolsPromptIncludedForCodeSecurityToolset verifies that when a
// workflow uses the code_security GitHub toolset the generated lock file references
// one of the github_mcp_tools prompt files (which carry the list_code_scanning_alerts
// guardrail). This is a regression test for unbounded alert queries.
func TestGitHubMCPToolsPromptIncludedForCodeSecurityToolset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gh-aw-code-scanning-prompt-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	testContent := `---
on: push
engine: claude
permissions:
  security-events: read
tools:
  github:
    toolsets: [code_security]
---

# Test Workflow with Code Security Toolset

List open critical code scanning alerts.
`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile))

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	lockStr := string(lockContent)

	// The generated workflow must reference one of the github_mcp_tools prompt files
	// (plain or with_safeoutputs variant) so the list_code_scanning_alerts guardrail
	// is injected at runtime. Both variants carry the same guardrail.
	hasGitHubMCPPrompt := strings.Contains(lockStr, githubMCPToolsPromptFile) ||
		strings.Contains(lockStr, githubMCPToolsWithSafeOutputsPromptFile)
	assert.True(t, hasGitHubMCPPrompt,
		"lock file should reference a github_mcp_tools prompt file when code_security toolset is active")
}

// TestGeneratedWorkflowsValidatePromptStep tests that all generated workflows
// include the prompt validation step
func TestGeneratedWorkflowsValidatePromptStep(t *testing.T) {
	// Get the workflows directory
	workflowsDir := filepath.Join("..", "..", ".github", "workflows")

	// Check if directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory not found, skipping test")
	}

	// Read all .lock.yml files
	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	require.NoError(t, err, "Should be able to list lock files")

	if len(files) == 0 {
		t.Skip("No lock files found, skipping test")
	}

	// Check each workflow
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err, "Should be able to read lock file")

			lockStr := string(content)

			// Skip workflows that don't have agent jobs (some workflows might not need prompts)
			if !strings.Contains(lockStr, "name: agent") {
				t.Skip("Workflow doesn't have agent job")
			}

			// Check for the validation step
			assert.Contains(t, lockStr, "Validate prompt placeholders",
				"Workflow should include prompt validation step")

			// Check that validation script is called
			assert.Contains(t, lockStr, "validate_prompt_placeholders.sh",
				"Workflow should call validation script")

			// Verify the validation step comes after interpolation
			interpolatePos := strings.Index(lockStr, "Interpolate variables and render templates")
			validatePos := strings.Index(lockStr, "Validate prompt placeholders")

			if interpolatePos != -1 && validatePos != -1 {
				assert.Less(t, interpolatePos, validatePos,
					"Validation should come after interpolation")
			}

			// Verify validation comes before print
			printPos := strings.Index(lockStr, "Print prompt")
			if validatePos != -1 && printPos != -1 {
				assert.Less(t, validatePos, printPos,
					"Validation should come before print")
			}
		})
	}
}

// TestGeneratedWorkflowsPromptStructure tests that generated workflows
// have proper prompt structure with system tags and ordering
func TestGeneratedWorkflowsPromptStructure(t *testing.T) {
	workflowsDir := filepath.Join("..", "..", ".github", "workflows")

	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory not found, skipping test")
	}

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	require.NoError(t, err)

	if len(files) == 0 {
		t.Skip("No lock files found")
	}

	// Sample a few workflows to test
	sampleSize := 5
	if len(files) > sampleSize {
		files = files[:sampleSize]
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)

			lockStr := string(content)

			// Skip workflows without agent jobs
			if !strings.Contains(lockStr, "name: agent") {
				t.Skip("Workflow doesn't have agent job")
			}

			// Check for system tags in the prompt creation
			if strings.Contains(lockStr, "Create prompt with built-in context") {
				// Should have opening system tag
				assert.Contains(t, lockStr, "<system>",
					"Workflow should have opening system tag")

				// Should have closing system tag
				assert.Contains(t, lockStr, "</system>",
					"Workflow should have closing system tag")

				// Verify system tags come in order
				systemOpenPos := strings.Index(lockStr, "<system>")
				systemClosePos := strings.Index(lockStr, "</system>")

				if systemOpenPos != -1 && systemClosePos != -1 {
					assert.Less(t, systemOpenPos, systemClosePos,
						"Opening system tag should come before closing tag")
				}
			}
		})
	}
}

// TestGeneratedWorkflowsPlaceholderFormat tests that placeholders in generated
// workflows follow the correct format and are in appropriate locations
func TestGeneratedWorkflowsPlaceholderFormat(t *testing.T) {
	workflowsDir := filepath.Join("..", "..", ".github", "workflows")

	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory not found, skipping test")
	}

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	require.NoError(t, err)

	if len(files) == 0 {
		t.Skip("No lock files found")
	}

	// Sample one workflow for detailed check
	file := files[0]
	content, err := os.ReadFile(file)
	require.NoError(t, err)

	lockStr := string(content)

	// Skip if no agent job
	if !strings.Contains(lockStr, "name: agent") {
		t.Skip("Workflow doesn't have agent job")
	}

	// Find all __GH_AW_*__ placeholders
	// These should only appear in:
	// 1. Heredoc content (between cat << 'PROMPT_EOF' and PROMPT_EOF)
	// 2. Environment variable values

	// Count placeholders
	placeholderCount := strings.Count(lockStr, "__GH_AW_")
	if placeholderCount > 0 {
		t.Logf("Found %d placeholder occurrences in %s", placeholderCount, filepath.Base(file))

		// This is expected - placeholders should be in the heredoc content
		// They will be replaced at runtime by the substitution step

		// Verify that these placeholders are NOT in step names or other critical areas
		assert.NotContains(t, lockStr, "name: __GH_AW_",
			"Placeholders should not be in step names")
		assert.NotContains(t, lockStr, "uses: __GH_AW_",
			"Placeholders should not be in action uses")
	}
}
