package workflow

import (
	"context"
	"errors"
	"testing"
)

// TestResolveGhAwRefFullSHAShortCircuit verifies that a full SHA is returned
// unchanged without querying the GitHub API.
func TestResolveGhAwRefFullSHAShortCircuit(t *testing.T) {
	const sha = "abcdef0123456789abcdef0123456789abcdef01"
	got, err := ResolveGhAwRef(context.Background(), sha)
	if err != nil {
		t.Fatalf("ResolveGhAwRef returned error: %v", err)
	}
	if got != sha {
		t.Errorf("ResolveGhAwRef(%q) = %q, want unchanged", sha, got)
	}
}

// TestResolveCommitRefSHACancelledContext verifies that the shared commit-ref
// helper surfaces command failures (rather than notFullCommitSHAError) when
// the gh invocation itself fails.
func TestResolveCommitRefSHACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sha, err := resolveCommitRefSHA(ctx, "github/gh-aw", "main")
	if err == nil {
		t.Fatalf("expected error with cancelled context, got sha=%q", sha)
	}
	var badSHA *notFullCommitSHAError
	if errors.As(err, &badSHA) {
		t.Errorf("command failure should not report notFullCommitSHAError, got %v", err)
	}
	if sha != "" {
		t.Errorf("expected empty SHA on error, got %q", sha)
	}
}
