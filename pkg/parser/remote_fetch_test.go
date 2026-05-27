//go:build !integration

package parser

import "testing"

func TestBuildCommitLookupAPIPath(t *testing.T) {
	t.Run("escapes refs containing slash", func(t *testing.T) {
		got := buildCommitLookupAPIPath("owner", "repo", "feature/github-agentic-workflows")
		want := "/repos/owner/repo/commits/feature%2Fgithub-agentic-workflows"
		if got != want {
			t.Fatalf("buildCommitLookupAPIPath() = %q, want %q", got, want)
		}
	})

	t.Run("keeps plain refs readable", func(t *testing.T) {
		got := buildCommitLookupAPIPath("owner", "repo", "main")
		want := "/repos/owner/repo/commits/main"
		if got != want {
			t.Fatalf("buildCommitLookupAPIPath() = %q, want %q", got, want)
		}
	})
}
