package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortfolioAnalystAvoidsInvalidSentryAggregations(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "portfolio-analyst.md"))
	if err != nil {
		t.Fatalf("failed to read portfolio analyst workflow source: %v", err)
	}

	text := string(content)
	for _, fragment := range []string{
		"Do not use Sentry `sum()`, `avg()`, percentile, or other numeric aggregations",
		"`gh-aw.aic`, `gh_aw.aic`, `agent_usage.aic`, `aic`, `gh-aw.action_minutes`, or `action_minutes`",
		"If Sentry reports a field-type or query-syntax 400",
		"aggregate numeric-looking values locally in Python",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected portfolio analyst workflow to contain %q", fragment)
		}
	}
}

func TestSharedSentryImportFailsFastOnRepeatedQuery400s(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "shared", "mcp", "sentry.md"))
	if err != nil {
		t.Fatalf("failed to read shared Sentry MCP import: %v", err)
	}

	text := string(content)
	for _, fragment := range []string{
		"Treat repeated Sentry HTTP 400 responses for the same query shape as terminal",
		"After the first field-type or query-syntax 400",
		"if an equivalent 400 occurs again, stop retrying that query family",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected shared Sentry MCP import to contain %q", fragment)
		}
	}
}
