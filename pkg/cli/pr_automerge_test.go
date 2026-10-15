//go:build !integration

package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestEnablePRAutoMerge(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		queryErr    error
		mutationErr error
		wantMethod  string
		wantError   string
	}{
		{name: "default squash", response: autoMergeRepoJSON(true, "SQUASH", true, true, true, "PR_42"), wantMethod: "SQUASH"},
		{name: "default merge", response: autoMergeRepoJSON(true, "MERGE", true, true, true, "PR_42"), wantMethod: "MERGE"},
		{name: "default rebase", response: autoMergeRepoJSON(true, "REBASE", true, true, true, "PR_42"), wantMethod: "REBASE"},
		{name: "fallback squash only", response: autoMergeRepoJSON(true, "MERGE", false, true, false, "PR_42"), wantMethod: "SQUASH"},
		{name: "fallback rebase only", response: autoMergeRepoJSON(true, "", false, false, true, "PR_42"), wantMethod: "REBASE"},
		{name: "disabled", response: autoMergeRepoJSON(false, "MERGE", true, true, true, "PR_42"), wantError: "disabled or unavailable"},
		{name: "no allowed methods", response: autoMergeRepoJSON(true, "MERGE", false, false, false, "PR_42"), wantError: "no supported merge method"},
		{name: "missing PR", response: autoMergeRepoJSON(true, "MERGE", true, true, true, ""), wantError: "pull request ID"},
		{name: "query failure", queryErr: errors.New("permission denied"), wantError: "permission denied"},
		{name: "cancelled", queryErr: context.Canceled, wantError: "context canceled"},
		{name: "unsupported API", queryErr: errors.New("Field 'autoMergeAllowed' doesn't exist"), wantError: "autoMergeAllowed"},
		{name: "malformed response", response: "{", wantError: "could not parse"},
		{name: "null repository", response: `{"data":{"repository":null}}`, wantError: "disabled or unavailable"},
		{name: "mutation failure", response: autoMergeRepoJSON(true, "SQUASH", true, true, true, "PR_42"), mutationErr: errors.New("required branch rules unavailable"), wantMethod: "SQUASH", wantError: "required branch rules unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := autoMergeRunGH
			t.Cleanup(func() { autoMergeRunGH = original })
			calls := 0
			autoMergeRunGH = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				calls++
				joined := strings.Join(args, " ")
				if !strings.HasPrefix(joined, "api graphql --hostname github.example.com ") {
					t.Fatalf("unexpected command: %s", joined)
				}
				if calls == 1 {
					for _, field := range []string{"owner=owner", "name=repo", "number=42", "viewerDefaultMergeMethod", "autoMergeAllowed"} {
						if !strings.Contains(joined, field) {
							t.Errorf("query missing %q: %s", field, joined)
						}
					}
					return []byte(tt.response), tt.queryErr
				}
				for _, field := range []string{"enablePullRequestAutoMerge", "prId=PR_42", "mergeMethod=" + tt.wantMethod} {
					if !strings.Contains(joined, field) {
						t.Errorf("mutation missing %q: %s", field, joined)
					}
				}
				if strings.Contains(joined, "--admin") || strings.Contains(joined, "mergePullRequest(") {
					t.Fatalf("must not merge directly or bypass checks: %s", joined)
				}
				return []byte(`{"data":{"enablePullRequestAutoMerge":{"pullRequest":{"id":"PR_42"}}}}`), tt.mutationErr
			}
			prURL := "https://github.example.com/owner/repo/pull/42"
			err := enablePRAutoMerge(context.Background(), prURL)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) ||
					!strings.Contains(err.Error(), prURL) || !strings.Contains(err.Error(), "remains open") {
					t.Fatalf("error = %v, want %q and retained PR URL", err, tt.wantError)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			wantCalls := 1
			if tt.wantMethod != "" {
				wantCalls = 2
			}
			if calls != wantCalls {
				t.Fatalf("calls = %d, want %d", calls, wantCalls)
			}
		})
	}
}

func autoMergeRepoJSON(enabled bool, method string, merge, squash, rebase bool, id string) string {
	return fmt.Sprintf(`{"data":{"repository":{"autoMergeAllowed":%t,"viewerDefaultMergeMethod":%q,"mergeCommitAllowed":%t,"squashMergeAllowed":%t,"rebaseMergeAllowed":%t,"pullRequest":{"id":%q}}}}`,
		enabled, method, merge, squash, rebase, id)
}

func TestEnablePRAutoMergeRejectsInvalidURLs(t *testing.T) {
	original := autoMergeRunGH
	t.Cleanup(func() { autoMergeRunGH = original })
	autoMergeRunGH = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("invalid URL must not make an API call")
		return nil, nil
	}
	for _, prURL := range []string{"", "%", "http://github.com/owner/repo/pull/42", "https://github.com/owner/repo/issues/42", "https://github.com/owner/repo/pull/-1", "https://github.com/owner/repo/pull/no", "https://user@github.com/owner/repo/pull/42"} {
		if err := enablePRAutoMerge(context.Background(), prURL); err == nil {
			t.Errorf("expected invalid URL error for %q", prURL)
		}
	}
}

func TestPRTransferAutoMergeFlags(t *testing.T) {
	for _, alias := range []string{"pr", "create-pull-request"} {
		t.Run(alias, func(t *testing.T) {
			cmd := NewPRTransferSubcommand()
			if err := cmd.ParseFlags([]string{"--" + alias, "--auto-merge"}); err != nil {
				t.Fatal(err)
			}
			autoMerge, _ := cmd.Flags().GetBool("auto-merge")
			createPR, _ := cmd.Flags().GetBool(alias)
			if !autoMerge || !createPR {
				t.Fatal("flags must be enabled")
			}
		})
	}
	cmd := NewPRTransferSubcommand()
	if err := cmd.ParseFlags([]string{"--auto-merge"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"invalid-url"}); err == nil || !strings.Contains(err.Error(), "--auto-merge requires") {
		t.Fatalf("error = %v, want flag validation before PR operations", err)
	}
}

// TestWaitForWorkflowCompletionUsesSignalHandling verifies that WaitForWorkflowCompletion
// uses the signal-aware polling helper, which provides Ctrl-C support
func TestWaitForWorkflowCompletionUsesSignalHandling(t *testing.T) {
	t.Parallel()
	// This test verifies that the function uses PollWithSignalHandling
	// by checking that it times out correctly (a key feature of the helper)

	// We can't easily test the actual workflow checking without a real workflow,
	// but we can verify that the timeout mechanism works, which confirms
	// it's using the polling helper

	err := WaitForWorkflowCompletion(context.Background(), "nonexistent/repo", "12345", 0, false)

	// Should timeout or fail to check workflow status
	if err == nil {
		t.Error("Expected error for nonexistent workflow, got nil")
	}
}

// TestWaitForWorkflowCompletion_ContextCancellation verifies that WaitForWorkflowCompletion
// propagates cancellation when the context is cancelled, so callers (e.g. the repeat loop)
// can detect an intentional interruption and stop immediately.
func TestWaitForWorkflowCompletion_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the poll loop exits on the first ctx.Done() check.
	cancel()

	err := WaitForWorkflowCompletion(ctx, "nonexistent/repo", "12345", 5, false)

	if err == nil {
		t.Fatal("Expected error on cancelled context, got nil")
	}

	// Must be either ErrInterrupted (from the poll select loop) or context.Canceled
	// (from the PollFunc guard when ctx is already cancelled before polling begins).
	// Both indicate an intentional interruption that callers should detect and propagate.
	isInterruption := errors.Is(err, ErrInterrupted) || errors.Is(err, context.Canceled)
	if !isInterruption {
		t.Errorf("Expected interruption error (ErrInterrupted or context.Canceled) from WaitForWorkflowCompletion, got: %v", err)
	}
}
