//go:build !integration

package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalContentType(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"application/json", "application/json"},
		{"application/json; charset=utf-8", "application/json"},
		{"text/markdown", "text/markdown"},
		{"TEXT/MARKDOWN", "text/markdown"},
		{"text/x-markdown; charset=utf-8", "text/x-markdown"},
		{"application/vnd.api+json", "application/vnd.api+json"},
		{"", ""},
		{"not-valid;;;", "not-valid"},
	}
	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			got := canonicalContentType(tc.raw)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFetchImportURL_Markdown(t *testing.T) {
	const markdownContent = "---\ndescription: test\n---\n\n# Hello\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "text/markdown")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte(markdownContent))
	}))
	defer srv.Close()

	res, err := FetchImportURL(context.Background(), srv.URL+"/workflow.md", FetchOptions{HTTPClient: srv.Client()})
	require.NoError(t, err)
	assert.Equal(t, "text/markdown", res.ContentType)
	assert.Equal(t, []byte(markdownContent), res.Body)
}

func TestFetchImportURL_JSON(t *testing.T) {
	const jsonContent = `{"id":"my-wf","name":"My Workflow"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(jsonContent))
	}))
	defer srv.Close()

	res, err := FetchImportURL(context.Background(), srv.URL+"/workflow.json", FetchOptions{HTTPClient: srv.Client()})
	require.NoError(t, err)
	assert.Equal(t, "application/json", res.ContentType)
	assert.JSONEq(t, jsonContent, string(res.Body))
}

func TestFetchImportURL_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchImportURL(context.Background(), srv.URL+"/missing.md", FetchOptions{HTTPClient: srv.Client()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFetchImportURL_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := FetchImportURL(context.Background(), srv.URL+"/private.md", FetchOptions{HTTPClient: srv.Client()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestFetchImportURL_SizeCap(t *testing.T) {
	large := make([]byte, importURLMaxBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write(large)
	}))
	defer srv.Close()

	_, err := FetchImportURL(context.Background(), srv.URL+"/big.md", FetchOptions{HTTPClient: srv.Client()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "size limit")
}

func TestFetchImportURL_HeadFallbackToGET(t *testing.T) {
	const markdownContent = "---\n---\n\n# Workflow\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			// Server doesn't support HEAD.
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// GET returns Content-Type.
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte(markdownContent))
	}))
	defer srv.Close()

	res, err := FetchImportURL(context.Background(), srv.URL+"/workflow.md", FetchOptions{HTTPClient: srv.Client()})
	require.NoError(t, err)
	assert.Equal(t, "text/markdown", res.ContentType)
}

func TestAttachImportAuthHeader_NonGitHub(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "my-secret-token")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://example.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://example.com/workflow.md")
	// Token must NOT be attached for non-GitHub hosts.
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_GitHub(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-xyz")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_GitHubCopilotNonAutomation(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-xyz")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://api.githubcopilot.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://api.githubcopilot.com/workflow.md")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
	assert.Empty(t, req.Header.Get("Copilot-Integration-Id"), "Copilot-Integration-Id should not be set for non-automation URLs")
}

func TestAttachImportAuthHeader_GitHubCopilotAutomation(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-xyz")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://api.githubcopilot.com/agents/repos/octocat/hello-world/automations/00000000-0000-0000-0000-000000000001", nil)
	attachImportAuthHeader(req, "https://api.githubcopilot.com/agents/repos/octocat/hello-world/automations/00000000-0000-0000-0000-000000000001")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
	assert.Equal(t, "agentic-workflows", req.Header.Get("Copilot-Integration-Id"))
}

func TestIsCopilotAutomationImportURL(t *testing.T) {
	tests := []struct {
		name  string
		input *url.URL
		want  bool
	}{
		{name: "nil URL", input: nil, want: false},
		{
			name:  "valid automation URL",
			input: mustParseURL(t, "https://api.githubcopilot.com/agents/repos/octocat/hello-world/automations/00000000-0000-0000-0000-000000000001"),
			want:  true,
		},
		{
			name:  "hostname match is case-insensitive",
			input: mustParseURL(t, "https://API.GITHUBCOPILOT.COM/agents/repos/octocat/hello-world/automations/00000000-0000-0000-0000-000000000001"),
			want:  true,
		},
		{
			name:  "non-automation path",
			input: mustParseURL(t, "https://api.githubcopilot.com/workflow.json"),
			want:  false,
		},
		{
			name:  "missing repo segment",
			input: mustParseURL(t, "https://api.githubcopilot.com/agents/repos/octocat//automations/00000000-0000-0000-0000-000000000001"),
			want:  false,
		},
		{
			name:  "missing automation ID",
			input: mustParseURL(t, "https://api.githubcopilot.com/agents/repos/octocat/hello-world/automations/"),
			want:  false,
		},
		{
			name:  "wrong segment count",
			input: mustParseURL(t, "https://api.githubcopilot.com/agents/repos/octocat/hello-world/automations/00000000-0000-0000-0000-000000000001/extra"),
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isCopilotAutomationImportURL(tc.input))
		})
	}
}

// mustParseURL parses a URL for table-driven tests and fails the test if parsing errors.
func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err, "Failed to parse URL: %s", raw)
	return u
}

func TestAttachImportAuthHeader_RawGitHubContent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-xyz")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://raw.githubusercontent.com/owner/repo/main/workflow.md", nil)
	attachImportAuthHeader(req, "https://raw.githubusercontent.com/owner/repo/main/workflow.md")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_GitHubUserContentWildcard(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-xyz")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://media.githubusercontent.com/media/owner/repo/main/workflow.md", nil)
	attachImportAuthHeader(req, "https://media.githubusercontent.com/media/owner/repo/main/workflow.md")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_GitHubObjects(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-xyz")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://objects.githubusercontent.com/github-production-release-asset-2e65be/owner/repo/workflow.md", nil)
	attachImportAuthHeader(req, "https://objects.githubusercontent.com/github-production-release-asset-2e65be/owner/repo/workflow.md")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_FallbackToGITHUB_TOKEN(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "github-token-abc")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer github-token-abc", req.Header.Get("Authorization"))
}

// TestAttachImportAuthHeader_GHTokenPreferredOverGHAuthToken verifies that an
// explicit GH_TOKEN env var wins over the gh auth token CLI fallback.
func TestAttachImportAuthHeader_GHTokenPreferredOverGHAuthToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "explicit-gh-token")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer explicit-gh-token", req.Header.Get("Authorization"))
}

// ── Security boundary tests ───────────────────────────────────────────────────

// Token must NEVER be sent over plain HTTP even to github.com.
func TestAttachImportAuthHeader_HTTP_GitHub_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "super-secret")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "http://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "http://github.com/owner/repo/raw/main/wf.md")
	assert.Empty(t, req.Header.Get("Authorization"), "token must not be sent over plain HTTP")
}

// Subdomains of github.com must not receive the token.
func TestAttachImportAuthHeader_GitHubSubdomain_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "super-secret")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://evil.github.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://evil.github.com/workflow.md")
	assert.Empty(t, req.Header.Get("Authorization"), "subdomain of github.com must not match")
}

// A hostname that ends with "github.com" but is a different domain must not match.
func TestAttachImportAuthHeader_SuffixConfusion_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "super-secret")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://notgithub.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://notgithub.com/workflow.md")
	assert.Empty(t, req.Header.Get("Authorization"), "hostname suffix confusion must not match")
}

// A hostname like "github.com.evil.com" must not match.
func TestAttachImportAuthHeader_DotAppended_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "super-secret")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com.evil.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://github.com.evil.com/workflow.md")
	assert.Empty(t, req.Header.Get("Authorization"), "github.com.evil.com must not match github.com")
}

func TestAttachImportAuthHeader_GitHubUserContentSuffixConfusion_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "super-secret")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://githubusercontent.com.evil.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://githubusercontent.com.evil.com/workflow.md")
	assert.Empty(t, req.Header.Get("Authorization"), "githubusercontent.com.evil.com must not match *.githubusercontent.com")
}

func TestAttachImportAuthHeader_DocsGitHub_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "super-secret")
	t.Setenv("GH_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://docs.github.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://docs.github.com/workflow.md")
	assert.Empty(t, req.Header.Get("Authorization"), "docs.github.com must not receive import auth token")
}

// ── GHE host tests ────────────────────────────────────────────────────────────

// GH_HOST set as a bare hostname (no scheme).
func TestAttachImportAuthHeader_GHE_BareHostname(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghe-token")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_HOST", "ghe.example.com")

	req, _ := http.NewRequest(http.MethodGet, "https://ghe.example.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://ghe.example.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer ghe-token", req.Header.Get("Authorization"), "bare GH_HOST hostname must be allowed")
}

// GH_HOST set with https:// scheme prefix.
func TestAttachImportAuthHeader_GHE_HTTPSScheme(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghe-token")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_HOST", "https://ghe.example.com")

	req, _ := http.NewRequest(http.MethodGet, "https://ghe.example.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://ghe.example.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer ghe-token", req.Header.Get("Authorization"), "GH_HOST with https:// prefix must be allowed")
}

// GH_HOST set with http:// scheme prefix — token must still only be sent over HTTPS requests.
func TestAttachImportAuthHeader_GHE_HTTPSchemePrefix(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghe-token")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_HOST", "http://ghe.example.com")

	// HTTPS request → token sent.
	req, _ := http.NewRequest(http.MethodGet, "https://ghe.example.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://ghe.example.com/workflow.md")
	assert.Equal(t, "Bearer ghe-token", req.Header.Get("Authorization"), "HTTPS request to GHE must receive token")

	// HTTP request → token NOT sent, even though GH_HOST matches.
	req2, _ := http.NewRequest(http.MethodGet, "http://ghe.example.com/workflow.md", nil)
	attachImportAuthHeader(req2, "http://ghe.example.com/workflow.md")
	assert.Empty(t, req2.Header.Get("Authorization"), "HTTP request to GHE must not receive token")
}

// GH_HOST is set but the request targets a different host — no token.
func TestAttachImportAuthHeader_GHE_DifferentHost(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghe-token")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_HOST", "ghe.example.com")

	req, _ := http.NewRequest(http.MethodGet, "https://other.example.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://other.example.com/workflow.md")
	assert.Empty(t, req.Header.Get("Authorization"), "different host must not receive token even when GH_HOST is set")
}

// github.com still works alongside a configured GH_HOST.
func TestAttachImportAuthHeader_GitHubAlongsideGHE(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "dual-token")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GH_HOST", "ghe.example.com")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer dual-token", req.Header.Get("Authorization"), "github.com must still be allowed when GH_HOST is also set")
}

// TestBuildRequestLogString_RedactsAuthorization verifies that the request formatter
// never exposes the raw token and shows the correct redacted form.
func TestBuildRequestLogString_RedactsAuthorization(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.githubcopilot.com/agents/repos/octocat/hello-world/automations/00000000-0000-0000-0000-000000000001", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token")

	output := buildRequestLogString(req)

	assert.Contains(t, output, "Bearer supe***", "Authorization must show first 4 chars then ***")
	assert.NotContains(t, output, "super-secret-token", "raw token must never appear in log output")
	assert.Contains(t, output, "Host: api.githubcopilot.com", "Host header must be logged")
	assert.Contains(t, output, "/agents/repos/octocat/hello-world/automations/", "request path must be logged")
}

// TestFetchImportURL_400ReturnsError verifies that a 400 response is returned as
// an error with the status code, and that the body is consumed (not leaked).
func TestFetchImportURL_400ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"missing_integration_id"}`))
	}))
	defer srv.Close()

	_, err := FetchImportURL(context.Background(), srv.URL+"/automation", FetchOptions{
		HTTPClient: srv.Client(),
	})

	require.Error(t, err, "400 must be returned as an error")
	assert.Contains(t, err.Error(), "400", "error must mention the status code")
}
