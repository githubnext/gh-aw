package workflowcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDailyPerformanceSummaryUsesStableWindowMetrics verifies that the daily
// performance prompt keeps updatedAt pagination separate from lifecycle metrics.
// The tokens below are the stable contract surface; changing them requires
// updating both the workflow source and this test.
func TestDailyPerformanceSummaryUsesStableWindowMetrics(t *testing.T) {
	t.Parallel()
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "daily-performance-summary.md"))
	if err != nil {
		t.Fatalf("failed to read daily performance summary workflow source: %v", err)
	}
	text := string(content)

	for _, token := range []string{
		"window_start",
		"mergedAt >= window_start",
		"closedAt >= window_start",
		"github-pr-query with state: \"open\", limit: 1000",
		"github-issue-query with state: \"open\", limit: 1000",
		"/tmp/gh-aw/python/data/open_prs.json",
		"/tmp/gh-aw/python/data/open_issues.json",
		"open_prs = load_json_data",
		"open_issues = load_json_data",
		"(pr_df['mergedAt'] >= ninety_days_ago)",
		"(issue_df['closedAt'] >= ninety_days_ago)",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("expected daily-performance-summary.md to contain token %q", token)
		}
	}
}
