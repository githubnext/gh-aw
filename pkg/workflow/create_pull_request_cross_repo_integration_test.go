//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreatePullRequestCrossRepoCheckout tests that target-repo properly configures checkout and git
func TestCreatePullRequestCrossRepoCheckout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cross-repo-checkout-test")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	// Create test workflow with cross-repo target
	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-pull-request:
    target-repo: "microsoft/vscode-docs"
    base-branch: vnext
    draft: true
---

# Cross-Repo Test Workflow

Create a pull request in a different repository.
`

	workflowPath := filepath.Join(tmpDir, "cross-repo.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "Failed to write workflow file")

	// Compile the workflow
	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath), "Failed to compile workflow")

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "cross-repo.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read compiled output")

	compiledContent := string(compiledBytes)

	// Test 1: Verify target-repo is propagated to the safe-outputs handler config
	assert.Contains(t, compiledContent, `"target-repo":"microsoft/vscode-docs"`,
		"Expected target-repo to be included in safe-outputs handler config")

	// Test 2: Verify checkout remains a default workspace checkout (no explicit cross-repo checkout)
	checkoutSection := extractCheckoutSection(compiledContent)
	assert.NotContains(t, checkoutSection, "repository:",
		"Checkout section should not have explicit repository when using default workspace checkout")

	// Test 3: Verify checkout does not inject a dedicated token for target-repo
	assert.NotContains(t, checkoutSection, "token:",
		"Checkout section should not have an explicit token for target-repo")

	// Test 4: Verify GITHUB_REPOSITORY remains the source repository expression
	assert.Contains(t, compiledContent, "GITHUB_REPOSITORY: ${{ github.repository }}",
		"Expected GITHUB_REPOSITORY to remain github.repository expression")
}

// TestCreatePullRequestSameRepoCheckout tests that without target-repo, we use default checkout
func TestCreatePullRequestSameRepoCheckout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "same-repo-checkout-test")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	// Create test workflow without target-repo
	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-pull-request:
    draft: true
---

# Same-Repo Test Workflow

Create a pull request in the same repository.
`

	workflowPath := filepath.Join(tmpDir, "same-repo.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "Failed to write workflow file")

	// Compile the workflow
	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath), "Failed to compile workflow")

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "same-repo.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read compiled output")

	compiledContent := string(compiledBytes)

	// Test 1: Verify no explicit repository parameter (uses default)
	checkoutSection := extractCheckoutSection(compiledContent)
	assert.NotContains(t, checkoutSection, "repository:",
		"Checkout section should not have explicit repository when using source repo")

	// Test 2: Verify GITHUB_REPOSITORY uses github.repository expression
	assert.Contains(t, compiledContent, "GITHUB_REPOSITORY: ${{ github.repository }}",
		"Expected GITHUB_REPOSITORY to use github.repository expression for same-repo")

	// Test 3: Verify no token in checkout (not needed for same repo)
	assert.NotContains(t, checkoutSection, "token:",
		"Checkout section should not have token for same-repo checkout")
}

// extractCheckoutSection extracts the checkout step from compiled YAML for inspection
func extractCheckoutSection(content string) string {
	lines := strings.Split(content, "\n")
	inCheckout := false
	var checkoutLines []string

	for _, line := range lines {
		if strings.Contains(line, "name: Checkout repository") {
			inCheckout = true
		}
		if inCheckout {
			checkoutLines = append(checkoutLines, line)
			// Stop at the next step (less indentation than "      -")
			if strings.HasPrefix(line, "      - name:") && !strings.Contains(line, "Checkout repository") {
				break
			}
		}
	}

	return strings.Join(checkoutLines, "\n")
}
