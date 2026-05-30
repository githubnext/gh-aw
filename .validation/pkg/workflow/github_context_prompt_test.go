//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestGitHubContextPrompt_UsesAwContextFallbacks(t *testing.T) {
	assertContains := func(expected string) {
		t.Helper()
		if !strings.Contains(githubContextPromptText, expected) {
			t.Fatalf("expected github context prompt to contain %q", expected)
		}
	}

	assertContains("github.event.issue.number || (github.aw.context.item_type == 'issue' && github.aw.context.item_number)")
	assertContains("github.event.discussion.number || (github.aw.context.item_type == 'discussion' && github.aw.context.item_number)")
	assertContains("github.event.pull_request.number || (github.aw.context.item_type == 'pull_request' && github.aw.context.item_number)")
	assertContains("github.event.comment.id || github.aw.context.comment_id")
}
