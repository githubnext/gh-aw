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

// TestGitHubMCPAppTokenConfiguration tests that app configuration is correctly parsed for GitHub tool
func TestGitHubMCPAppTokenConfiguration(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	markdown := `---
on: issues
permissions:
  contents: read
  issues: read  # read permission for testing
strict: false  # disable strict mode for testing
tools:
  github:
    mode: local
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
      repositories:
        - "repo1"
        - "repo2"
---

# Test Workflow

Test workflow with GitHub MCP Server app configuration.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse markdown content")
	require.NotNil(t, workflowData.ParsedTools, "ParsedTools should not be nil")
	require.NotNil(t, workflowData.ParsedTools.GitHub, "GitHub tool should be parsed")
	require.NotNil(t, workflowData.ParsedTools.GitHub.App, "App configuration should be parsed")

	// Verify app configuration
	assert.Equal(t, "${{ vars.APP_ID }}", workflowData.ParsedTools.GitHub.App.AppID)
	assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", workflowData.ParsedTools.GitHub.App.PrivateKey)
	assert.Equal(t, []string{"repo1", "repo2"}, workflowData.ParsedTools.GitHub.App.Repositories)
}

// TestGitHubMCPAppTokenMintingStep tests that token minting step is generated
func TestGitHubMCPAppTokenMintingStep(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	markdown := `---
on: issues
permissions:
  contents: read
  issues: read  # read permission for testing
strict: false  # disable strict mode for testing
tools:
  github:
    mode: local
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow

Test workflow with GitHub MCP app token minting.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file (same name with .lock.yml extension)
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")
	lockContent := string(content)

	// Verify token minting step is present
	assert.Contains(t, lockContent, "Generate GitHub App token", "Token minting step should be present")
	assert.Contains(t, lockContent, "actions/create-github-app-token", "Should use create-github-app-token action")
	assert.Contains(t, lockContent, "id: github-mcp-app-token", "Should use github-mcp-app-token as step ID")
	assert.Contains(t, lockContent, "app-id: ${{ vars.APP_ID }}", "Should use configured app ID")
	assert.Contains(t, lockContent, "private-key: ${{ secrets.APP_PRIVATE_KEY }}", "Should use configured private key")

	// Verify permissions are passed to the app token minting
	assert.Contains(t, lockContent, "permission-contents: read", "Should include contents read permission")
	assert.Contains(t, lockContent, "permission-issues: read", "Should include issues read permission")

	// Verify token invalidation step is present
	assert.Contains(t, lockContent, "Invalidate GitHub App token", "Token invalidation step should be present")
	assert.Contains(t, lockContent, "if: always()", "Invalidation step should always run")
	assert.Contains(t, lockContent, "steps.github-mcp-app-token.outputs.token", "Should reference github-mcp-app-token output")

	// Verify the app token is used for GitHub MCP Server
	assert.Contains(t, lockContent, "GITHUB_MCP_SERVER_TOKEN: ${{ steps.github-mcp-app-token.outputs.token }}", "Should use app token for GitHub MCP Server")
}

// TestGitHubMCPAppTokenAndGitHubTokenMutuallyExclusive tests that setting both app and github-token is rejected
func TestGitHubMCPAppTokenAndGitHubTokenMutuallyExclusive(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	markdown := `---
on: issues
permissions:
  contents: read
tools:
  github:
    mode: local
    github-token: ${{ secrets.CUSTOM_GITHUB_TOKEN }}
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test Workflow

Test that setting both app and github-token is an error.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Compile the workflow - should fail because both app and github-token are set
	err = compiler.CompileWorkflow(testFile)
	require.Error(t, err, "Expected error when both app and github-token are set")
	assert.Contains(t, err.Error(), "'tools.github.app' and 'tools.github.github-token' cannot both be set", "Error should mention mutual exclusion")
}

// TestGitHubMCPAppTokenWithRemoteMode tests that app token works with remote mode
func TestGitHubMCPAppTokenWithRemoteMode(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	markdown := `---
on: issues
permissions:
  contents: read
tools:
  github:
    mode: remote
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
engine: claude
---

# Test Workflow

Test app token with remote GitHub MCP Server.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file (same name with .lock.yml extension)
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")
	lockContent := string(content)

	// Verify token minting step is present
	assert.Contains(t, lockContent, "Generate GitHub App token", "Token minting step should be present")
	assert.Contains(t, lockContent, "id: github-mcp-app-token", "Should use github-mcp-app-token as step ID")

	// Verify the app token is used in the authorization header for remote mode
	// The token should be in the HTTP config's Authorization header
	if strings.Contains(lockContent, `"Authorization": "Bearer ${{ steps.github-mcp-app-token.outputs.token }}"`) {
		// Success - app token is used
		t.Log("App token correctly used in remote mode Authorization header")
	} else {
		// Also check for the env var reference pattern used by Claude engine
		assert.Contains(t, lockContent, "GITHUB_MCP_SERVER_TOKEN: ${{ steps.github-mcp-app-token.outputs.token }}", "Should use app token for GitHub MCP Server in remote mode")
	}
}

// TestGitHubMCPAppTokenOrgWide tests org-wide GitHub MCP token with wildcard
func TestGitHubMCPAppTokenOrgWide(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	markdown := `---
on: issues
permissions:
  contents: read
  issues: read
strict: false
tools:
  github:
    mode: local
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
      repositories:
        - "*"
---

# Test Workflow

Test org-wide GitHub MCP app token.
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Read the generated lock file (same name with .lock.yml extension)
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")
	lockContent := string(content)

	// Verify token minting step is present
	assert.Contains(t, lockContent, "Generate GitHub App token", "Token minting step should be present")

	// Verify repositories field is NOT present (org-wide access)
	assert.NotContains(t, lockContent, "repositories:", "Should not include repositories field for org-wide access")

	// Verify other fields are still present
	assert.Contains(t, lockContent, "owner:", "Should include owner field")
	assert.Contains(t, lockContent, "app-id:", "Should include app-id field")
}

// TestGitHubMCPAppTokenNoLockdownDetectionStep tests that determine-automatic-lockdown
// step is NOT generated when a GitHub App is configured.
// GitHub App tokens are already scoped to specific repositories, so automatic lockdown
// detection is unnecessary.
func TestGitHubMCPAppTokenNoLockdownDetectionStep(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	markdown := `---
on: issues
permissions:
  contents: read
  issues: read
strict: false
tools:
  github:
    mode: local
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
      repositories:
        - "repo1"
        - "repo2"
---

# Test Workflow

Test that determine-automatic-lockdown is not generated when app is configured.
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")
	lockContent := string(content)

	// The automatic lockdown detection step must NOT be present when app is configured
	assert.NotContains(t, lockContent, "Determine automatic lockdown mode", "determine-automatic-lockdown step should not be generated when app is configured")
	assert.NotContains(t, lockContent, "id: determine-automatic-lockdown", "determine-automatic-lockdown step ID should not be present")
	assert.NotContains(t, lockContent, "steps.determine-automatic-lockdown.outputs.lockdown", "lockdown step output reference should not be present")

	// App token should still be minted and used
	assert.Contains(t, lockContent, "id: github-mcp-app-token", "GitHub App token step should still be generated")
	assert.Contains(t, lockContent, "GITHUB_MCP_SERVER_TOKEN: ${{ steps.github-mcp-app-token.outputs.token }}", "App token should be used for MCP server")
}
