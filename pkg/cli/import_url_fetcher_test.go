//go:build !integration

package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	assert.Equal(t, []byte(jsonContent), res.Body)
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
	t.Setenv("GH_TOKEN", "my-secret-token")

	req, _ := http.NewRequest(http.MethodGet, "https://example.com/workflow.md", nil)
	attachImportAuthHeader(req, "https://example.com/workflow.md")
	// Token must NOT be attached for non-GitHub hosts.
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_GitHub(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-xyz")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer gh-token-xyz", req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_FallbackToGITHUB_TOKEN(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "github-token-abc")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Equal(t, "Bearer github-token-abc", req.Header.Get("Authorization"))
}

func TestAttachImportAuthHeader_NoToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	req, _ := http.NewRequest(http.MethodGet, "https://github.com/owner/repo/raw/main/wf.md", nil)
	attachImportAuthHeader(req, "https://github.com/owner/repo/raw/main/wf.md")
	assert.Empty(t, req.Header.Get("Authorization"))
}
