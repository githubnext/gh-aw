//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// extractEditWikiHandlerConfig extracts the edit_wiki handler config from a compiled
// lock file's GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG env var.
func extractEditWikiHandlerConfig(t *testing.T, lockContent []byte) map[string]any {
	t.Helper()

	var workflowDoc map[string]any
	require.NoError(t, yaml.Unmarshal(lockContent, &workflowDoc), "Failed to unmarshal lock workflow YAML")

	jobsRaw, ok := workflowDoc["jobs"].(map[string]any)
	require.True(t, ok, "Generated workflow should contain jobs map")

	safeOutputsJobRaw, ok := jobsRaw["safe_outputs"].(map[string]any)
	require.True(t, ok, "Generated workflow should contain safe_outputs job")

	stepsRaw, ok := safeOutputsJobRaw["steps"].([]any)
	require.True(t, ok, "Generated workflow safe_outputs job should contain steps array")

	var handlerConfigJSON string
	for _, step := range stepsRaw {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		envMap, ok := stepMap["env"].(map[string]any)
		if !ok {
			continue
		}
		rawConfig, ok := envMap["GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG"].(string)
		if ok && rawConfig != "" {
			handlerConfigJSON = rawConfig
			break
		}
	}

	require.NotEmpty(t, handlerConfigJSON, "Generated workflow should contain GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG env var")

	var handlerConfig map[string]any
	require.NoError(t, json.Unmarshal([]byte(handlerConfigJSON), &handlerConfig), "Failed to unmarshal GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG JSON")

	editWikiCfgRaw, ok := handlerConfig["edit_wiki"].(map[string]any)
	require.True(t, ok, "Handler config should contain edit_wiki object")

	return editWikiCfgRaw
}

func TestEditWikiConfigParsing(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	testMarkdown := `---
on:
  issues:
    types: [opened]
safe-outputs:
  edit-wiki:
  noop:
    report-as-issue: false
---

# Test Edit Wiki

This is a test workflow to validate edit-wiki configuration parsing.
`

	mdFile := filepath.Join(tmpDir, "test-edit-wiki.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(testMarkdown), 0644), "Failed to write test markdown file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdFile), "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContentStr := string(lockContent)

	// Verify safe_outputs job is generated
	assert.Contains(t, lockContentStr, "safe_outputs:", "Generated workflow should contain safe_outputs job")

	// Verify handler manager step is present
	assert.Contains(t, lockContentStr, "id: process_safe_outputs", "Generated workflow should contain process_safe_outputs step")

	// Verify edit_wiki config is in handler manager config
	assert.Contains(t, lockContentStr, "edit_wiki", "Generated workflow should contain edit_wiki in handler config")

	// Verify that required permissions are present
	safeOutputsJobSection := extractJobSection(lockContentStr, "safe_outputs")
	assert.NotEmpty(t, safeOutputsJobSection, "safe_outputs job section should be present")
	assert.Contains(t, safeOutputsJobSection, "contents: write", "Generated workflow should have contents: write permission")

	// Verify that the patch download step is included (edit-wiki needs patches)
	assert.Contains(t, lockContentStr, "Download patch artifact", "Generated workflow should download patch artifact")

	// Verify there is no Checkout repository step specifically for the main repo in the safe_outputs job
	// (edit-wiki does not need a compile-time checkout of the source repo)
	assert.NotContains(t, safeOutputsJobSection, "Checkout repository", "edit-wiki alone should not generate a repo checkout step")
}

func TestEditWikiWithAllowedRepos(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	testMarkdown := `---
on:
  issues:
    types: [opened]
safe-outputs:
  edit-wiki:
    allowed-repos:
      - "org/other-repo"
  noop:
    report-as-issue: false
---

# Test Edit Wiki Allowed Repos
`

	mdFile := filepath.Join(tmpDir, "test-edit-wiki-allowed-repos.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(testMarkdown), 0644), "Failed to write test markdown file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdFile), "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	wikiCfg := extractEditWikiHandlerConfig(t, lockContent)

	// Verify allowed_repos is in the handler config
	allowedRepos, exists := wikiCfg["allowed_repos"]
	assert.True(t, exists, "edit_wiki handler config should contain allowed_repos")
	allowedReposSlice, ok := allowedRepos.([]any)
	assert.True(t, ok, "allowed_repos should be a slice")
	assert.Len(t, allowedReposSlice, 1, "allowed_repos should have 1 entry")
	assert.Equal(t, "org/other-repo", allowedReposSlice[0], "allowed_repos should contain org/other-repo")
}

func TestEditWikiWithTargetRepo(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	testMarkdown := `---
on:
  issues:
    types: [opened]
safe-outputs:
  edit-wiki:
    repo: "org/target-repo"
  noop:
    report-as-issue: false
---

# Test Edit Wiki Target Repo
`

	mdFile := filepath.Join(tmpDir, "test-edit-wiki-target-repo.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(testMarkdown), 0644), "Failed to write test markdown file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdFile), "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	wikiCfg := extractEditWikiHandlerConfig(t, lockContent)

	// Verify target-repo is in the handler config
	targetRepo, exists := wikiCfg["target-repo"]
	assert.True(t, exists, "edit_wiki handler config should contain target-repo")
	assert.Equal(t, "org/target-repo", targetRepo, "target-repo should be org/target-repo")
}

func TestEditWikiIfNoChanges(t *testing.T) {
	tests := []struct {
		name         string
		ifNoChanges  string
		expectInJSON string
	}{
		{
			name:         "error value",
			ifNoChanges:  "error",
			expectInJSON: `"if_no_changes":"error"`,
		},
		{
			name:         "ignore value",
			ifNoChanges:  "ignore",
			expectInJSON: `"if_no_changes":"ignore"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")

			testMarkdown := `---
on:
  issues:
    types: [opened]
safe-outputs:
  edit-wiki:
    if-no-changes: ` + tt.ifNoChanges + `
  noop:
    report-as-issue: false
---

# Test Edit Wiki If No Changes
`

			mdFile := filepath.Join(tmpDir, "test-edit-wiki-if-no-changes.md")
			require.NoError(t, os.WriteFile(mdFile, []byte(testMarkdown), 0644), "Failed to write test markdown file")

			compiler := NewCompiler()
			require.NoError(t, compiler.CompileWorkflow(mdFile), "Failed to compile workflow")

			lockFile := stringutil.MarkdownToLockFile(mdFile)
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read lock file")

			lockContentStr := string(lockContent)
			assert.True(t,
				strings.Contains(lockContentStr, tt.expectInJSON) ||
					strings.Contains(lockContentStr, strings.ReplaceAll(tt.expectInJSON, `"`, `\"`)),
				"Generated workflow should contain if_no_changes=%s in handler config", tt.ifNoChanges,
			)
		})
	}
}

func TestEditWikiPermissions(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	testMarkdown := `---
on:
  issues:
    types: [opened]
safe-outputs:
  edit-wiki:
  noop:
    report-as-issue: false
---

# Test Edit Wiki Permissions
`

	mdFile := filepath.Join(tmpDir, "test-edit-wiki-permissions.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(testMarkdown), 0644), "Failed to write test markdown file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdFile), "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContentStr := string(lockContent)
	safeOutputsJobSection := extractJobSection(lockContentStr, "safe_outputs")
	require.NotEmpty(t, safeOutputsJobSection, "safe_outputs job section should be present")

	// Edit-wiki only requires contents: write (not pull-requests: write)
	assert.Contains(t, safeOutputsJobSection, "contents: write", "Generated workflow should have contents: write permission for edit-wiki")
	assert.NotContains(t, safeOutputsJobSection, "pull-requests: write", "Generated workflow should NOT have pull-requests: write permission for edit-wiki")
}

func TestUsesPatchesAndCheckoutsIncludesEditWiki(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expected    bool
	}{
		{
			name:        "nil safe outputs",
			safeOutputs: nil,
			expected:    false,
		},
		{
			name: "edit-wiki configured",
			safeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{},
			},
			expected: true,
		},
		{
			name: "edit-wiki staged returns false",
			safeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true}},
			},
			expected: false,
		},
		{
			name: "edit-wiki with globally staged returns false",
			safeOutputs: &SafeOutputsConfig{
				Staged:   true,
				EditWiki: &EditWikiConfig{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usesPatchesAndCheckouts(tt.safeOutputs)
			assert.Equal(t, tt.expected, result, "usesPatchesAndCheckouts should return expected value")
		})
	}
}

func TestUsesPRCheckoutExcludesEditWiki(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expected    bool
	}{
		{
			name:        "nil safe outputs",
			safeOutputs: nil,
			expected:    false,
		},
		{
			name: "edit-wiki only returns false (no PR checkout needed)",
			safeOutputs: &SafeOutputsConfig{
				EditWiki: &EditWikiConfig{},
			},
			expected: false,
		},
		{
			name: "create-pull-request returns true",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: true,
		},
		{
			name: "push-to-pull-request-branch returns true",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expected: true,
		},
		{
			name: "edit-wiki with create-pull-request returns true",
			safeOutputs: &SafeOutputsConfig{
				EditWiki:           &EditWikiConfig{},
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usesPRCheckout(tt.safeOutputs)
			assert.Equal(t, tt.expected, result, "usesPRCheckout should return expected value")
		})
	}
}
