package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// fakeGHBin writes a shell script to a temp dir, prepends the dir to PATH, and
// returns a cleanup function that restores the original PATH.
func fakeGHBin(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestResolveCommitRefSHASuccess verifies that resolveCommitRefSHA returns the
// SHA printed to stdout and does not treat stderr noise as part of the SHA.
func TestResolveCommitRefSHASuccess(t *testing.T) {
	const want = "abcdef0123456789abcdef0123456789abcdef01"
	// The fake gh prints the SHA on stdout and noisy text on stderr; only
	// stdout must end up in the returned SHA.
	fakeGHBin(t, `echo '`+want+`'
echo 'stderr noise' >&2
exit 0
`)
	got, err := resolveCommitRefSHA(context.Background(), "owner/repo", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestResolveCommitRefSHANonzeroExit verifies that a nonzero exit from gh is
// surfaced as a plain error and not misreported as notFullCommitSHAError.
func TestResolveCommitRefSHANonzeroExit(t *testing.T) {
	fakeGHBin(t, `echo 'something went wrong' >&2
exit 1
`)
	sha, err := resolveCommitRefSHA(context.Background(), "owner/repo", "bad-ref")
	if err == nil {
		t.Fatalf("expected error on nonzero exit, got sha=%q", sha)
	}
	var badSHA *notFullCommitSHAError
	if errors.As(err, &badSHA) {
		t.Errorf("nonzero exit should not produce notFullCommitSHAError, got %v", err)
	}
}

// TestResolveCommitRefSHAMalformedOutput verifies that unexpected stdout
// (not a 40-char hex SHA) is classified as notFullCommitSHAError so callers
// can emit an "unexpected response" diagnostic.
func TestResolveCommitRefSHAMalformedOutput(t *testing.T) {
	fakeGHBin(t, `echo 'not-a-sha'
exit 0
`)
	sha, err := resolveCommitRefSHA(context.Background(), "owner/repo", "main")
	if err == nil {
		t.Fatalf("expected error for malformed output, got sha=%q", sha)
	}
	var badSHA *notFullCommitSHAError
	if !errors.As(err, &badSHA) {
		t.Errorf("malformed output should produce notFullCommitSHAError, got %T: %v", err, err)
	}
	if !strings.Contains(badSHA.response, "not-a-sha") {
		t.Errorf("notFullCommitSHAError.response = %q, want it to contain %q", badSHA.response, "not-a-sha")
	}
}

// TestResolveCommitRefSHAForcesGHHost verifies that GH_HOST is set to
// github.com regardless of the ambient environment, so a GHE host in the
// caller's environment cannot hijack the lookup.
func TestResolveCommitRefSHAForcesGHHost(t *testing.T) {
	const want = "abcdef0123456789abcdef0123456789abcdef01"
	logFile := filepath.Join(t.TempDir(), "env.log")
	// The fake gh records its environment and prints a valid SHA.
	fakeGHBin(t, `env >> '`+logFile+`'
echo '`+want+`'
exit 0
`)
	t.Setenv("GH_HOST", "ghe.example.com")

	got, err := resolveCommitRefSHA(context.Background(), "owner/repo", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	envBytes, readErr := os.ReadFile(logFile)
	if readErr != nil {
		t.Fatalf("read env log: %v", readErr)
	}
	envStr := string(envBytes)
	if !strings.Contains(envStr, "GH_HOST=github.com") {
		t.Errorf("GH_HOST was not forced to github.com; env log:\n%s", envStr)
	}
}
