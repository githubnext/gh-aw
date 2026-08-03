package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPortfolioAnalystAvoidsInvalidSentryAggregations verifies the guard that
// prevents the agent from using sum()/avg()/percentile aggregations on
// unconfirmed-numeric Sentry fields.  The tokens below are the stable contract
// surface; changing them requires updating both the workflow source and this test.
func TestPortfolioAnalystAvoidsInvalidSentryAggregations(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "portfolio-analyst.md"))
	if err != nil {
		t.Fatalf("failed to read portfolio analyst workflow source: %v", err)
	}
	text := string(content)

	// Semantic contract tokens – deliberately minimal so legitimate rewording
	// of surrounding prose does not cause spurious failures.
	for _, token := range []string{
		"sum()",
		"400",
		"aggregate",
		"count()",
		"schema type is not confirmed numeric",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("expected portfolio-analyst.md to contain token %q", token)
		}
	}

	// Guard must survive compilation: the lock file should import the source.
	lockContent, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "portfolio-analyst.lock.yml"))
	if err != nil {
		t.Fatalf("failed to read portfolio-analyst.lock.yml: %v", err)
	}
	if !strings.Contains(string(lockContent), "runtime-import .github/workflows/portfolio-analyst.md") {
		t.Fatalf("portfolio-analyst.lock.yml does not import the workflow source; the guard may have been dropped during compilation")
	}
}

// TestSharedSentryImportFailsFastOnRepeatedQuery400s verifies the shared Sentry
// import contains the retry-termination guard and fallback chain.
// The tokens below are the stable contract surface; changing them requires
// updating both the shared import and this test.
func TestSharedSentryImportFailsFastOnRepeatedQuery400s(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "shared", "mcp", "sentry.md"))
	if err != nil {
		t.Fatalf("failed to read shared Sentry MCP import: %v", err)
	}
	text := string(content)

	// Semantic contract tokens.
	for _, token := range []string{
		"400",
		"terminal",
		"count()",
		"fallback chain",
		"stop retrying",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("expected shared/mcp/sentry.md to contain token %q", token)
		}
	}

	// Guard must survive compilation: the portfolio-analyst lock file should
	// include a runtime-import for the shared sentry import.
	lockContent, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "portfolio-analyst.lock.yml"))
	if err != nil {
		t.Fatalf("failed to read portfolio-analyst.lock.yml: %v", err)
	}
	if !strings.Contains(string(lockContent), "runtime-import .github/workflows/shared/mcp/sentry.md") {
		t.Fatalf("portfolio-analyst.lock.yml does not import shared/mcp/sentry.md; the guard may have been dropped during compilation")
	}
}
