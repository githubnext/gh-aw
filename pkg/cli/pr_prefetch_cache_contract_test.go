//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedPRDiffDataFetchValidatesHeadSHAForCacheHit(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "shared", "pr-diff-data-fetch.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read shared pr-diff-data-fetch workflow")

	text := string(content)
	assert.Contains(t, text, "pr-data-head-sha.txt", "Shared PR prefetch should persist head SHA marker")
	assert.Contains(t, text, "--json number,title,body,headRefName,headRefOid,additions,deletions,changedFiles,files", "Shared PR prefetch should capture head SHA in metadata")
	assert.Contains(t, text, "Cache hit: using pre-fetched PR data for head", "Shared PR prefetch should verify cache by current head SHA")
	assert.Contains(t, text, `gh pr diff "$PR_NUMBER" --repo "$EXPR_GITHUB_REPOSITORY"`, "Shared PR prefetch should quote repository on gh pr diff")
	assert.Contains(t, text, `--repo "$EXPR_GITHUB_REPOSITORY" \`, "Shared PR prefetch should quote repository on gh pr view")
}

func TestTopReviewWorkflowsHaveHeadAwarePRDataCacheKeys(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	mattWorkflowPath := filepath.Join(repoRoot, ".github", "workflows", "mattpocock-skills-reviewer.md")
	mattContent, err := os.ReadFile(mattWorkflowPath)
	require.NoError(t, err, "Should read mattpocock-skills-reviewer workflow")
	assert.Contains(t, string(mattContent), "key: pr-prefetch-${{ github.event.pull_request.head.sha || github.event.issue.number }}", "Matt reviewer should use head-aware key with issue fallback")

	sentinelWorkflowPath := filepath.Join(repoRoot, ".github", "workflows", "test-quality-sentinel.md")
	sentinelContent, err := os.ReadFile(sentinelWorkflowPath)
	require.NoError(t, err, "Should read test-quality-sentinel workflow")
	text := string(sentinelContent)
	assert.Contains(t, text, "key: pr-test-prefetch-${{ github.event.pull_request.head.sha || github.event.issue.number }}", "Test Quality Sentinel should define a head-aware cache key")
	assert.Contains(t, text, "test-data-head-sha.txt", "Test Quality Sentinel should persist cache head SHA marker")
}

func TestDailyGeoOptimizerQuotesRepositoryURL(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "daily-geo-optimizer.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read daily-geo-optimizer workflow")

	assert.Contains(t, string(content), `geo audit --url "https://github.com/${{ github.repository }}" --format json`, "Daily geo optimizer should quote the repository URL to avoid shell word splitting")
}
