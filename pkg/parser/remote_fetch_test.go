//go:build !integration

package parser

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestBuildCommitLookupAPIPath(t *testing.T) {
	t.Run("escapes refs containing slash", func(t *testing.T) {
		got := buildCommitLookupAPIPath("owner", "repo", "feature/github-agentic-workflows")
		want := "repos/owner/repo/commits/feature%2Fgithub-agentic-workflows"
		if got != want {
			t.Fatalf("buildCommitLookupAPIPath() = %q, want %q", got, want)
		}
	})

	t.Run("keeps plain refs readable", func(t *testing.T) {
		got := buildCommitLookupAPIPath("owner", "repo", "main")
		want := "repos/owner/repo/commits/main"
		if got != want {
			t.Fatalf("buildCommitLookupAPIPath() = %q, want %q", got, want)
		}
	})
}

func TestBuildContentsAPIPath(t *testing.T) {
	t.Run("escapes refs with reserved query chars", func(t *testing.T) {
		got := buildContentsAPIPath("owner", "repo", ".github/workflows/demo.md", "release+candidate#1")
		want := "repos/owner/repo/contents/.github/workflows/demo.md?ref=release%2Bcandidate%231"
		if got != want {
			t.Fatalf("buildContentsAPIPath() = %q, want %q", got, want)
		}
	})

	t.Run("keeps plain refs readable", func(t *testing.T) {
		got := buildContentsAPIPath("owner", "repo", ".github/workflows/demo.md", "main")
		want := "repos/owner/repo/contents/.github/workflows/demo.md?ref=main"
		if got != want {
			t.Fatalf("buildContentsAPIPath() = %q, want %q", got, want)
		}
	})

	t.Run("escapes path segments with reserved chars", func(t *testing.T) {
		got := buildContentsAPIPath("owner", "repo", "skills/path with spaces/file#100%.md", "main")
		want := "repos/owner/repo/contents/skills/path%20with%20spaces/file%23100%25.md?ref=main"
		if got != want {
			t.Fatalf("buildContentsAPIPath() = %q, want %q", got, want)
		}
	})
}

func TestGitFallbackRequiresNonEmptyRef(t *testing.T) {
	t.Run("all files fallback validates ref", func(t *testing.T) {
		_, err := listDirAllFilesViaGitForHost(context.Background(), "owner", "repo", "", "skills/demo", "")
		if err == nil {
			t.Fatal("expected error for empty ref")
		}
		if !strings.Contains(err.Error(), "non-empty ref") {
			t.Fatalf("expected non-empty ref error, got %q", err)
		}
	})

	t.Run("subdirs fallback validates ref", func(t *testing.T) {
		_, err := listDirSubdirsViaGitForHost(context.Background(), "owner", "repo", "   ", "skills", "")
		if err == nil {
			t.Fatal("expected error for empty ref")
		}
		if !strings.Contains(err.Error(), "non-empty ref") {
			t.Fatalf("expected non-empty ref error, got %q", err)
		}
	})
}

func TestListContentsRecursivelyWithDepth_MaxDepthGuard(t *testing.T) {
	_, err := listContentsRecursivelyWithDepth(t.Context(), nil, "owner", "repo", "main", "skills/demo/deep", 11, 10)
	if err == nil {
		t.Fatal("expected depth limit error")
	}
	if !strings.Contains(err.Error(), "maximum skill directory recursion depth exceeded") {
		t.Fatalf("expected depth limit error, got %q", err)
	}
}

type fakeCommitResolver struct {
	do func(ctx context.Context, method string, path string, body io.Reader, response any) error
}

func (f fakeCommitResolver) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response any) error {
	return f.do(ctx, method, path, body, response)
}

func TestResolveRefToSHAWithFallbacks_UsesRESTClient(t *testing.T) {
	client := fakeCommitResolver{
		do: func(_ context.Context, method string, path string, body io.Reader, response any) error {
			if method != http.MethodGet {
				t.Fatalf("DoWithContext() method = %q, want %q", method, http.MethodGet)
			}
			if path != "repos/owner/repo/commits/feature%2Fbranch" {
				t.Fatalf("DoWithContext() path = %q, want %q", path, "repos/owner/repo/commits/feature%2Fbranch")
			}
			if body != nil {
				t.Fatal("DoWithContext() body should be nil for commit lookup")
			}
			resp, ok := response.(*commitLookupResponse)
			if !ok {
				t.Fatalf("response type = %T, want *commitLookupResponse", response)
			}
			resp.SHA = "0123456789abcdef0123456789abcdef01234567"
			return nil
		},
	}

	gitCalled := false
	publicCalled := false
	sha, err := resolveRefToSHAWithFallbacks(
		context.Background(),
		client,
		"owner",
		"repo",
		"feature/branch",
		"github.com",
		func(context.Context, string, string, string, string) (string, error) {
			gitCalled = true
			return "", nil
		},
		func(context.Context, string, string, string) (string, error) {
			publicCalled = true
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("resolveRefToSHAWithFallbacks() error = %v", err)
	}
	if sha != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("resolveRefToSHAWithFallbacks() SHA = %q", sha)
	}
	if gitCalled {
		t.Fatal("git fallback should not run on successful REST lookup")
	}
	if publicCalled {
		t.Fatal("public fallback should not run on successful REST lookup")
	}
}

func TestResolveRefToSHAWithFallbacks_UsesGitFallbackOnAuthError(t *testing.T) {
	client := fakeCommitResolver{
		do: func(_ context.Context, method string, path string, body io.Reader, response any) error {
			return &api.HTTPError{StatusCode: http.StatusUnauthorized, Message: "Unauthorized"}
		},
	}

	publicCalled := false
	sha, err := resolveRefToSHAWithFallbacks(
		context.Background(),
		client,
		"owner",
		"repo",
		"main",
		"github.com",
		func(context.Context, string, string, string, string) (string, error) {
			return "0123456789abcdef0123456789abcdef01234567", nil
		},
		func(context.Context, string, string, string) (string, error) {
			publicCalled = true
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("resolveRefToSHAWithFallbacks() error = %v", err)
	}
	if sha != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("resolveRefToSHAWithFallbacks() SHA = %q", sha)
	}
	if publicCalled {
		t.Fatal("public fallback should not run when git fallback succeeds")
	}
}

func TestResolveRefToSHAWithFallbacks_UsesPublicFallbackAfterGitFallbackOnGitHubDotCom(t *testing.T) {
	client := fakeCommitResolver{
		do: func(_ context.Context, method string, path string, body io.Reader, response any) error {
			return &api.HTTPError{StatusCode: http.StatusForbidden, Message: "Forbidden"}
		},
	}

	gitCalled := false
	publicCalled := false
	sha, err := resolveRefToSHAWithFallbacks(
		context.Background(),
		client,
		"owner",
		"repo",
		"main",
		"",
		func(context.Context, string, string, string, string) (string, error) {
			gitCalled = true
			return "", errors.New("git fallback failed")
		},
		func(context.Context, string, string, string) (string, error) {
			publicCalled = true
			return "fedcba9876543210fedcba9876543210fedcba98", nil
		},
	)
	if err != nil {
		t.Fatalf("resolveRefToSHAWithFallbacks() error = %v", err)
	}
	if sha != "fedcba9876543210fedcba9876543210fedcba98" {
		t.Fatalf("resolveRefToSHAWithFallbacks() SHA = %q", sha)
	}
	if !gitCalled {
		t.Fatal("git fallback should run on GitHub API auth errors")
	}
	if !publicCalled {
		t.Fatal("public fallback should run when git fallback also fails on github.com")
	}
}

func TestResolveRefToSHA_ClientCreationAuthError_UsesGitFallback(t *testing.T) {
	origFactory := createRESTClientForHostFunc
	origGit := resolveRefToSHAViaGitFunc
	t.Cleanup(func() {
		createRESTClientForHostFunc = origFactory
		resolveRefToSHAViaGitFunc = origGit
	})

	authErr := errors.New("unauthorized: authentication required")
	createRESTClientForHostFunc = func(host string) (*api.RESTClient, error) {
		return nil, authErr
	}
	gitCalled := false
	resolveRefToSHAViaGitFunc = func(_ context.Context, owner, repo, ref, host string) (string, error) {
		gitCalled = true
		return "aabbccddeeff00112233445566778899aabbccdd", nil
	}

	sha, err := resolveRefToSHA(context.Background(), "owner", "repo", "main", "github.com")
	if err != nil {
		t.Fatalf("resolveRefToSHA() unexpected error = %v", err)
	}
	if sha != "aabbccddeeff00112233445566778899aabbccdd" {
		t.Fatalf("resolveRefToSHA() SHA = %q, want git-fallback SHA", sha)
	}
	if !gitCalled {
		t.Fatal("git ls-remote fallback should be invoked when client creation returns an auth error")
	}
}

func TestResolveRefToSHA_ClientCreationAuthError_GithubDotCom_UsesPublicAPIWhenGitFails(t *testing.T) {
	origFactory := createRESTClientForHostFunc
	origGit := resolveRefToSHAViaGitFunc
	t.Cleanup(func() {
		createRESTClientForHostFunc = origFactory
		resolveRefToSHAViaGitFunc = origGit
	})

	createRESTClientForHostFunc = func(host string) (*api.RESTClient, error) {
		return nil, errors.New("unauthorized: authentication required")
	}
	resolveRefToSHAViaGitFunc = func(_ context.Context, _, _, _, _ string) (string, error) {
		return "", errors.New("git ls-remote failed")
	}

	// When host is github.com and both REST client creation and git fail,
	// resolveRefToSHA should attempt the unauthenticated public API.
	// The public API call will fail in a unit-test environment; we verify the
	// error does NOT contain "failed to create GitHub REST client" (which would
	// indicate the code returned from the factory-error path without falling back).
	_, err := resolveRefToSHA(context.Background(), "owner", "repo", "main", "github.com")
	if err != nil && strings.Contains(err.Error(), "failed to create GitHub REST client") {
		t.Fatal("client creation auth error should not bubble up directly — public API fallback must be attempted first")
	}
}
