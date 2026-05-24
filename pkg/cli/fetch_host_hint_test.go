//go:build !integration

package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestHostResolutionHintForNotFound(t *testing.T) {
	t.Run("suggests full github URL for GHE shorthand 404", func(t *testing.T) {
		t.Setenv("GITHUB_SERVER_URL", "")
		t.Setenv("GITHUB_ENTERPRISE_HOST", "ghe.example.com")
		t.Setenv("GITHUB_HOST", "")
		t.Setenv("GH_HOST", "")

		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "", errors.New("gh: Not Found (HTTP 404)"),
		)

		if !ok {
			t.Fatal("expected hint for GHE shorthand 404")
		}
		if !strings.Contains(hint, "resolved on ghe.example.com") {
			t.Fatalf("expected host in hint, got: %s", hint)
		}
		if !strings.Contains(hint, "https://github.com/githubnext/agentics/blob/main/workflows/daily-repo-status.md") {
			t.Fatalf("expected full github.com URL hint, got: %s", hint)
		}
	})

	t.Run("does not suggest for github.com host", func(t *testing.T) {
		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "github.com", errors.New("gh: Not Found (HTTP 404)"),
		)
		if ok || hint != "" {
			t.Fatalf("expected no hint for github.com host, got ok=%v hint=%q", ok, hint)
		}
	})

	t.Run("does not suggest for nil or non-404 errors", func(t *testing.T) {
		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "", nil,
		)
		if ok || hint != "" {
			t.Fatalf("expected no hint for nil error, got ok=%v hint=%q", ok, hint)
		}

		t.Setenv("GITHUB_SERVER_URL", "")
		t.Setenv("GITHUB_ENTERPRISE_HOST", "ghe.example.com")
		t.Setenv("GITHUB_HOST", "")
		t.Setenv("GH_HOST", "")
		hint, ok = hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "/workflows/daily-repo-status.md", "", errors.New("gh: forbidden (HTTP 403)"),
		)
		if ok || hint != "" {
			t.Fatalf("expected no hint for non-404 error, got ok=%v hint=%q", ok, hint)
		}
	})

	t.Run("normalizes leading slash in workflow path", func(t *testing.T) {
		t.Setenv("GITHUB_SERVER_URL", "")
		t.Setenv("GITHUB_ENTERPRISE_HOST", "ghe.example.com")
		t.Setenv("GITHUB_HOST", "")
		t.Setenv("GH_HOST", "")

		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "/workflows/daily-repo-status.md", "", errors.New("HTTP 404"),
		)
		if !ok {
			t.Fatal("expected hint for 404 with normalized path")
		}
		if strings.Contains(hint, "blob/main//workflows") {
			t.Fatalf("expected normalized path without double slash, got: %s", hint)
		}
	})
}
