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

	assertContains("github.event.issue.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'issue' && fromJSON(github.event.inputs.aw_context || '{}').item_number)")
	assertContains("github.event.discussion.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'discussion' && fromJSON(github.event.inputs.aw_context || '{}').item_number)")
	assertContains("github.event.pull_request.number || (fromJSON(github.event.inputs.aw_context || '{}').item_type == 'pull_request' && fromJSON(github.event.inputs.aw_context || '{}').item_number)")
	assertContains("github.event.comment.id || fromJSON(github.event.inputs.aw_context || '{}').comment_id")
}
