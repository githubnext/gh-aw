//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addIntegrationTestSetup holds the setup state for add integration tests
type addIntegrationTestSetup struct {
	tempDir    string
	originalWd string
	binaryPath string
	cleanup    func()
}

// setupAddIntegrationTest creates a minimal test environment for add command:
// - temporary directory
// - git init (required by add command)
// - pre-built gh-aw binary
// Does NOT create .github/workflows - the add command should create it
func setupAddIntegrationTest(t *testing.T) *addIntegrationTestSetup {
	t.Helper()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gh-aw-add-integration-*")
	require.NoError(t, err, "Failed to create temp directory")

	// Save current working directory and change to temp directory
	originalWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	err = os.Chdir(tempDir)
	require.NoError(t, err, "Failed to change to temp directory")

	// Initialize git repository (required by add command)
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = tempDir
	output, err := gitInitCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run git init: %s", string(output))

	// Configure git user for commits (required for some operations)
	gitConfigName := exec.Command("git", "config", "user.name", "Test User")
	gitConfigName.Dir = tempDir
	_ = gitConfigName.Run() // Ignore errors - may already be configured globally

	gitConfigEmail := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigEmail.Dir = tempDir
	_ = gitConfigEmail.Run() // Ignore errors - may already be configured globally

	// Copy the pre-built binary to this test's temp directory
	binaryPath := filepath.Join(tempDir, "gh-aw")
	err = fileutil.CopyFile(globalBinaryPath, binaryPath)
	require.NoError(t, err, "Failed to copy gh-aw binary to temp directory")

	// Make the binary executable
	err = os.Chmod(binaryPath, 0755)
	require.NoError(t, err, "Failed to make binary executable")

	// Setup cleanup function
	cleanup := func() {
		_ = os.Chdir(originalWd)
		_ = os.RemoveAll(tempDir)
	}

	return &addIntegrationTestSetup{
		tempDir:    tempDir,
		originalWd: originalWd,
		binaryPath: binaryPath,
		cleanup:    cleanup,
	}
}

// TestAddRemoteWorkflowFromURL tests adding a remote workflow via GitHub URL
// This test requires GitHub authentication
func TestAddRemoteWorkflowFromURL(t *testing.T) {
	// Skip if GitHub authentication is not available
	// Check by running `gh auth status` - if it fails, skip
	authCmd := exec.Command("gh", "auth", "status")
	if err := authCmd.Run(); err != nil {
		t.Skip("Skipping test: GitHub authentication not available (gh auth status failed)")
	}

	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Add a workflow from a GitHub URL using the non-interactive flag
	// Using a workflow from the gh-aw repo itself for reliability
	workflowURL := "https://github.com/github/gh-aw/blob/v0.45.5/.github/workflows/github-mcp-tools-report.md"

	cmd := exec.Command(setup.binaryPath, "add", workflowURL, "--non-interactive", "--verbose")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Log output for debugging
	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify .github/workflows directory was created
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	info, err := os.Stat(workflowsDir)
	require.NoError(t, err, ".github/workflows directory should exist")
	assert.True(t, info.IsDir(), ".github/workflows should be a directory")

	// Verify the workflow file was created
	workflowFile := filepath.Join(workflowsDir, "github-mcp-tools-report.md")
	_, err = os.Stat(workflowFile)
	require.NoError(t, err, "workflow file should exist: %s", workflowFile)

	// Read and verify the workflow content
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "should be able to read workflow file")
	contentStr := string(content)

	// Verify the workflow has expected content
	assert.Contains(t, contentStr, "---", "workflow should have frontmatter delimiters")
	assert.Contains(t, contentStr, "on:", "workflow should have trigger definition")

	// Verify source field was added with commit pinning
	assert.Contains(t, contentStr, "source:", "workflow should have source field added")
	assert.Contains(t, contentStr, "github/gh-aw", "source should reference the source repo")

	// Verify the compiled .lock.yml file was created
	lockFile := filepath.Join(workflowsDir, "github-mcp-tools-report.lock.yml")
	_, err = os.Stat(lockFile)
	require.NoError(t, err, "lock file should exist: %s", lockFile)

	// Verify the lock file contains expected GitHub Actions content
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "should be able to read lock file")
	lockContentStr := string(lockContent)

	assert.Contains(t, lockContentStr, "name:", "lock file should have workflow name")
	assert.Contains(t, lockContentStr, "jobs:", "lock file should have jobs section")
}

// TestAddAllBlogSeriesWorkflows tests adding all v0.45.5 workflows from the blog series
// This comprehensive test verifies that all workflows referenced in the documentation can be added
// This test requires GitHub authentication
func TestAddAllBlogSeriesWorkflows(t *testing.T) {
	// Skip if GitHub authentication is not available
	authCmd := exec.Command("gh", "auth", "status")
	if err := authCmd.Run(); err != nil {
		t.Skip("Skipping test: GitHub authentication not available (gh auth status failed)")
	}

	// All v0.45.5 workflows from the blog series (58 total)
	workflows := []string{
		"agent-performance-analyzer.md",
		"audit-workflows.md",
		"blog-auditor.md",
		"breaking-change-checker.md",
		"changeset.md",
		"ci-coach.md",
		"ci-doctor.md",
		"cli-consistency-checker.md",
		"code-simplifier.md",
		"copilot-agent-analysis.md",
		"copilot-pr-nlp-analysis.md",
		"copilot-session-insights.md",
		"daily-compiler-quality.md",
		"daily-doc-updater.md",
		"daily-file-diet.md",
		"daily-malicious-code-scan.md",
		"daily-multi-device-docs-tester.md",
		"daily-news.md",
		"daily-repo-chronicle.md",
		"daily-secrets-analysis.md",
		"daily-team-status.md",
		"daily-testify-uber-super-expert.md",
		"daily-workflow-updater.md",
		"discussion-task-miner.md",
		"docs-noob-tester.md",
		"duplicate-code-detector.md",
		"firewall.md",
		"github-mcp-tools-report.md",
		"glossary-maintainer.md",
		"go-fan.md",
		"grumpy-reviewer.md",
		"issue-arborist.md",
		"issue-monster.md",
		"issue-triage-agent.md",
		"mcp-inspector.md",
		"mergefest.md",
		"metrics-collector.md",
		"org-health-report.md",
		"plan.md",
		"poem-bot.md",
		"portfolio-analyst.md",
		"prompt-clustering-analysis.md",
		"q.md",
		"repository-quality-improver.md",
		"schema-consistency-checker.md",
		"security-compliance.md",
		"semantic-function-refactor.md",
		"slide-deck-maintainer.md",
		"stale-repo-identifier.md",
		"static-analysis-report.md",
		"sub-issue-closer.md",
		"terminal-stylist.md",
		"typist.md",
		"ubuntu-image-analyzer.md",
		"unbloat-docs.md",
		"weekly-issue-summary.md",
		"workflow-generator.md",
		"workflow-health-manager.md",
	}

	for _, workflowName := range workflows {
		workflowName := workflowName // capture for loop variable
		t.Run(workflowName, func(t *testing.T) {
			// Note: Cannot use t.Parallel() because setupAddIntegrationTest() uses os.Chdir()
			// which modifies global process state and would cause races between goroutines

			setup := setupAddIntegrationTest(t)
			defer setup.cleanup()

			// Construct GitHub URL for the workflow at v0.45.5
			workflowURL := "https://github.com/github/gh-aw/blob/v0.45.5/.github/workflows/" + workflowName

			// Add the workflow
			cmd := exec.Command(setup.binaryPath, "add", workflowURL, "--non-interactive")
			cmd.Dir = setup.tempDir
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Assert successful addition
			require.NoError(t, err, "add command should succeed for %s: %s", workflowName, outputStr)

			// Verify .github/workflows directory was created
			workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
			info, err := os.Stat(workflowsDir)
			require.NoError(t, err, ".github/workflows directory should exist for %s", workflowName)
			assert.True(t, info.IsDir(), ".github/workflows should be a directory for %s", workflowName)

			// Verify the workflow file was created
			workflowFile := filepath.Join(workflowsDir, workflowName)
			_, err = os.Stat(workflowFile)
			require.NoError(t, err, "workflow file should exist for %s: %s", workflowName, workflowFile)

			// Read and verify the workflow has basic expected content
			content, err := os.ReadFile(workflowFile)
			require.NoError(t, err, "should be able to read workflow file for %s", workflowName)
			contentStr := string(content)

			// Verify basic frontmatter structure
			assert.Contains(t, contentStr, "---", "workflow %s should have frontmatter delimiters", workflowName)
			assert.Contains(t, contentStr, "on:", "workflow %s should have trigger definition", workflowName)

			// Verify source field was added with commit pinning
			assert.Contains(t, contentStr, "source:", "workflow %s should have source field added", workflowName)
			assert.Contains(t, contentStr, "github/gh-aw", "workflow %s source should reference the source repo", workflowName)

			// Verify the compiled .lock.yml file was created
			lockFileName := strings.TrimSuffix(workflowName, ".md") + ".lock.yml"
			lockFile := filepath.Join(workflowsDir, lockFileName)
			_, err = os.Stat(lockFile)
			require.NoError(t, err, "lock file should exist for %s: %s", workflowName, lockFile)

			// Verify the lock file contains expected GitHub Actions content
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "should be able to read lock file for %s", workflowName)
			lockContentStr := string(lockContent)

			assert.Contains(t, lockContentStr, "name:", "lock file for %s should have workflow name", workflowName)
			assert.Contains(t, lockContentStr, "jobs:", "lock file for %s should have jobs section", workflowName)
		})
	}
}

// TestAddLocalWorkflow tests adding a local workflow file
func TestAddLocalWorkflow(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file in a separate "source" directory
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "test-local-workflow.md")
	localWorkflowContent := `---
name: Test Local Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
---

# Test Local Workflow

This is a test workflow for integration testing.

Please analyze the repository and provide a summary.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add the local workflow using non-interactive mode
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive", "--verbose")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Log output for debugging
	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify .github/workflows directory was created
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	info, err := os.Stat(workflowsDir)
	require.NoError(t, err, ".github/workflows directory should exist")
	assert.True(t, info.IsDir(), ".github/workflows should be a directory")

	// Verify the workflow file was copied
	destWorkflowFile := filepath.Join(workflowsDir, "test-local-workflow.md")
	_, err = os.Stat(destWorkflowFile)
	require.NoError(t, err, "workflow file should exist: %s", destWorkflowFile)

	// Read and verify the workflow content
	content, err := os.ReadFile(destWorkflowFile)
	require.NoError(t, err, "should be able to read workflow file")
	contentStr := string(content)

	// Verify the workflow has expected content (original content preserved)
	assert.Contains(t, contentStr, "name: Test Local Workflow", "workflow should have original name")
	assert.Contains(t, contentStr, "workflow_dispatch", "workflow should have original trigger")
	assert.Contains(t, contentStr, "engine: copilot", "workflow should have original engine")
	assert.Contains(t, contentStr, "Please analyze the repository", "workflow should have original prompt")

	// Note: For local workflows without a git remote, source field is NOT added
	// since we can't determine the repo slug

	// Verify the compiled .lock.yml file was created
	lockFile := filepath.Join(workflowsDir, "test-local-workflow.lock.yml")
	_, err = os.Stat(lockFile)
	require.NoError(t, err, "lock file should exist: %s", lockFile)

	// Verify the lock file contains expected GitHub Actions content
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "should be able to read lock file")
	lockContentStr := string(lockContent)

	assert.Contains(t, lockContentStr, "name: \"Test Local Workflow\"", "lock file should have workflow name")
	assert.Contains(t, lockContentStr, "workflow_dispatch", "lock file should have trigger")
	assert.Contains(t, lockContentStr, "jobs:", "lock file should have jobs section")
}

// TestAddWorkflowWithCustomName tests adding a workflow with a custom name
func TestAddWorkflowWithCustomName(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "original-name.md")
	localWorkflowContent := `---
name: Original Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Original Workflow

Test content.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add with a custom name
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive", "--name", "custom-workflow-name")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the workflow file was created with custom name
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	customWorkflowFile := filepath.Join(workflowsDir, "custom-workflow-name.md")
	_, err = os.Stat(customWorkflowFile)
	require.NoError(t, err, "workflow file with custom name should exist: %s", customWorkflowFile)

	// Verify original name file does NOT exist
	originalNameFile := filepath.Join(workflowsDir, "original-name.md")
	_, err = os.Stat(originalNameFile)
	assert.True(t, os.IsNotExist(err), "original name file should NOT exist")
}

// TestAddWorkflowToCustomDir tests adding a workflow to a custom subdirectory
func TestAddWorkflowToCustomDir(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "test-workflow.md")
	localWorkflowContent := `---
name: Test Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Test Workflow

Test content.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add to a custom directory
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive", "--dir", "experimental")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the workflow file was created in the custom subdirectory
	customDir := filepath.Join(setup.tempDir, ".github", "workflows", "experimental")
	info, err := os.Stat(customDir)
	require.NoError(t, err, "custom workflows subdirectory should exist")
	assert.True(t, info.IsDir(), "should be a directory")

	workflowFile := filepath.Join(customDir, "test-workflow.md")
	_, err = os.Stat(workflowFile)
	require.NoError(t, err, "workflow file should exist in custom directory: %s", workflowFile)

	// Verify the lock file is also in the custom directory
	lockFile := filepath.Join(customDir, "test-workflow.lock.yml")
	_, err = os.Stat(lockFile)
	require.NoError(t, err, "lock file should exist in custom directory: %s", lockFile)
}

// TestAddWorkflowForce tests the --force flag to overwrite existing workflows
func TestAddWorkflowForce(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create .github/workflows directory with an existing workflow
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "should create workflows directory")

	existingWorkflowPath := filepath.Join(workflowsDir, "existing-workflow.md")
	existingContent := `---
name: Old Workflow
on: push
permissions:
  contents: read
engine: copilot
---

# Old Workflow

This is the OLD content that should be replaced.
`
	err = os.WriteFile(existingWorkflowPath, []byte(existingContent), 0644)
	require.NoError(t, err, "should write existing workflow file")

	// Create a new workflow file with same name in source directory
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err = os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	newWorkflowPath := filepath.Join(sourceDir, "existing-workflow.md")
	newContent := `---
name: New Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# New Workflow

This is the NEW content that should replace the old.
`
	err = os.WriteFile(newWorkflowPath, []byte(newContent), 0644)
	require.NoError(t, err, "should write new workflow file")

	// First try without --force (should fail)
	cmdNoForce := exec.Command(setup.binaryPath, "add", newWorkflowPath, "--non-interactive")
	cmdNoForce.Dir = setup.tempDir
	outputNoForce, errNoForce := cmdNoForce.CombinedOutput()
	outputNoForceStr := string(outputNoForce)

	t.Logf("Command output (without --force):\n%s", outputNoForceStr)

	assert.Error(t, errNoForce, "add without --force should fail when file exists")
	assert.Contains(t, outputNoForceStr, "already exists", "error should mention file exists")

	// Verify original content is still there
	content, err := os.ReadFile(existingWorkflowPath)
	require.NoError(t, err, "should read existing workflow")
	assert.Contains(t, string(content), "OLD content", "original content should remain")

	// Now try with --force (should succeed)
	cmdForce := exec.Command(setup.binaryPath, "add", newWorkflowPath, "--non-interactive", "--force")
	cmdForce.Dir = setup.tempDir
	outputForce, errForce := cmdForce.CombinedOutput()
	outputForceStr := string(outputForce)

	t.Logf("Command output (with --force):\n%s", outputForceStr)

	require.NoError(t, errForce, "add with --force should succeed: %s", outputForceStr)

	// Verify new content replaced old
	newContentRead, err := os.ReadFile(existingWorkflowPath)
	require.NoError(t, err, "should read updated workflow")
	assert.Contains(t, string(newContentRead), "NEW content", "new content should replace old")
	assert.NotContains(t, string(newContentRead), "OLD content", "old content should be gone")
}

// TestAddWorkflowCreatesGitattributes tests that .gitattributes is properly configured
func TestAddWorkflowCreatesGitattributes(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "test.md")
	localWorkflowContent := `---
name: Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Test

Content.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add the workflow
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify .gitattributes was created
	gitattributesPath := filepath.Join(setup.tempDir, ".gitattributes")
	_, err = os.Stat(gitattributesPath)
	require.NoError(t, err, ".gitattributes file should exist")

	// Verify .gitattributes has the lock file pattern
	gitattributesContent, err := os.ReadFile(gitattributesPath)
	require.NoError(t, err, "should read .gitattributes")
	gitattributesStr := string(gitattributesContent)

	assert.Contains(t, gitattributesStr, ".lock.yml", ".gitattributes should contain lock file pattern")
}

// TestAddWorkflowNoGitattributes tests that --no-gitattributes skips .gitattributes configuration in the add step
// Note: The compile step may still create .gitattributes, so we check the verbose output instead
func TestAddWorkflowNoGitattributes(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "test.md")
	localWorkflowContent := `---
name: Test
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Test

Content.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add the workflow with --no-gitattributes and --verbose to see output
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive", "--no-gitattributes", "--verbose")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the "Configured .gitattributes" message is NOT in the add step output
	// Note: The compile step may still create .gitattributes, but the add step should skip it
	assert.NotContains(t, outputStr, "Configured .gitattributes",
		"add step should NOT configure .gitattributes when --no-gitattributes is set")
}

// TestAddRemoteWorkflowWithVersion tests adding a remote workflow with a specific version tag
// Uses the 4+ part format with explicit path since the workflow is in .github/workflows/
// This test requires GitHub authentication
func TestAddRemoteWorkflowWithVersion(t *testing.T) {
	// Skip if GitHub authentication is not available
	authCmd := exec.Command("gh", "auth", "status")
	if err := authCmd.Run(); err != nil {
		t.Skip("Skipping test: GitHub authentication not available (gh auth status failed)")
	}

	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Use a workflow spec with explicit path (owner/repo/path/workflow.md@version format)
	// The 3-part format (owner/repo/workflow@version) looks in workflows/ directory,
	// but this workflow is in .github/workflows/, so we need the explicit path
	workflowSpec := "github/gh-aw/.github/workflows/github-mcp-tools-report.md@v0.45.5"

	cmd := exec.Command(setup.binaryPath, "add", workflowSpec, "--non-interactive", "--verbose")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the workflow file was created
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowsDir, "github-mcp-tools-report.md")
	_, err = os.Stat(workflowFile)
	require.NoError(t, err, "workflow file should exist: %s", workflowFile)

	// Read and verify source pinning
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "should be able to read workflow file")
	contentStr := string(content)

	// Should have source with version pinning
	assert.Contains(t, contentStr, "source:", "workflow should have source field")
	// The source should reference the commit SHA (not the tag directly)
	// This ensures reproducible builds
	assert.True(t,
		strings.Contains(contentStr, "@") && strings.Contains(contentStr, "github/gh-aw"),
		"source should have commit pinning")
}

// TestAddWorkflowWithEngineOverride tests that --engine flag adds/updates the engine field in the workflow
func TestAddWorkflowWithEngineOverride(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file WITHOUT an engine specified
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "no-engine-workflow.md")
	localWorkflowContent := `---
name: Workflow Without Engine
on: workflow_dispatch
permissions:
  contents: read
---

# Workflow Without Engine

This workflow does not specify an engine in frontmatter.

Please analyze the repository.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add the workflow with --engine claude
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive", "--engine", "claude")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the workflow file was created
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowsDir, "no-engine-workflow.md")
	_, err = os.Stat(workflowFile)
	require.NoError(t, err, "workflow file should exist: %s", workflowFile)

	// Read and verify the engine field was added
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "should be able to read workflow file")
	contentStr := string(content)

	// Should have engine: claude in frontmatter
	assert.Contains(t, contentStr, "engine: claude", "workflow should have engine field added")
}

// TestAddWorkflowEngineOverrideReplacesExisting tests that --engine flag replaces existing engine
func TestAddWorkflowEngineOverrideReplacesExisting(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file WITH an existing engine: copilot
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "copilot-workflow.md")
	localWorkflowContent := `---
name: Copilot Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Copilot Workflow

This workflow originally specifies copilot engine.

Please analyze the repository.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add the workflow with --engine claude (should replace copilot)
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive", "--engine", "claude")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the workflow file was created
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowsDir, "copilot-workflow.md")
	_, err = os.Stat(workflowFile)
	require.NoError(t, err, "workflow file should exist: %s", workflowFile)

	// Read and verify the engine field was updated
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "should be able to read workflow file")
	contentStr := string(content)

	// Should have engine: claude, NOT engine: copilot
	assert.Contains(t, contentStr, "engine: claude", "workflow should have engine field updated to claude")
	assert.NotContains(t, contentStr, "engine: copilot", "original copilot engine should be replaced")
}

// TestAddWorkflowWithoutEngineOverridePreservesOriginal tests that without --engine, original engine is preserved
func TestAddWorkflowWithoutEngineOverridePreservesOriginal(t *testing.T) {
	setup := setupAddIntegrationTest(t)
	defer setup.cleanup()

	// Create a local workflow file WITH an engine specified
	sourceDir := filepath.Join(setup.tempDir, "source-workflows")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err, "should create source directory")

	localWorkflowPath := filepath.Join(sourceDir, "claude-workflow.md")
	localWorkflowContent := `---
name: Claude Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: claude
---

# Claude Workflow

This workflow specifies claude engine.

Please analyze the repository.
`
	err = os.WriteFile(localWorkflowPath, []byte(localWorkflowContent), 0644)
	require.NoError(t, err, "should write local workflow file")

	// Add the workflow WITHOUT --engine flag
	cmd := exec.Command(setup.binaryPath, "add", localWorkflowPath, "--non-interactive")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("Command output:\n%s", outputStr)

	require.NoError(t, err, "add command should succeed: %s", outputStr)

	// Verify the workflow file was created
	workflowsDir := filepath.Join(setup.tempDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowsDir, "claude-workflow.md")
	_, err = os.Stat(workflowFile)
	require.NoError(t, err, "workflow file should exist: %s", workflowFile)

	// Read and verify the original engine is preserved
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "should be able to read workflow file")
	contentStr := string(content)

	// Should still have engine: claude (original)
	assert.Contains(t, contentStr, "engine: claude", "original engine should be preserved")
}
